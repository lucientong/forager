// Package pipeline composes all Forager agents into the full review pipeline.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lucientong/forager/internal/agents"
	"github.com/lucientong/forager/internal/config"
	"github.com/lucientong/forager/internal/github"
	llmpkg "github.com/lucientong/forager/internal/llm"
	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/guardrail"
	"github.com/lucientong/waggle/pkg/observe"
	"github.com/lucientong/waggle/pkg/stream"
	"github.com/lucientong/waggle/pkg/waggle"
)

// Pipeline orchestrates the full PR review workflow.
type Pipeline struct {
	fetchAgent   agent.Agent[*github.PullRequestEvent, *models.PRData]
	reviewMerge  agent.Agent[[]models.FileChange, *models.AggregatedReview]
	summaryAgent agent.Agent[*models.AggregatedReview, *models.AggregatedReview]
	postAgent    agent.Agent[*models.AggregatedReview, bool]
	metrics      *observe.Metrics
	events       chan observe.Event
	observer     stream.Observer
}

// New creates a new Pipeline from the given configuration and dependencies.
// Each review agent gets its own LLM provider resolved from the registry.
func New(cfg *config.Config, ghClient *github.Client, registry *llmpkg.Registry) (*Pipeline, error) {
	timeout := time.Duration(cfg.Pipeline.TimeoutSeconds) * time.Second

	// Build individual review agents — each with its own provider.
	secAgent := agents.NewSecurityAgent(registry.ForAgent("security"))
	styAgent := agents.NewStyleAgent(registry.ForAgent("style"))
	logAgent := agents.NewLogicAgent(registry.ForAgent("logic"))
	prfAgent := agents.NewPerformanceAgent(registry.ForAgent("performance"))

	// Compose parallel review + merge using waggle.ParallelThen (v0.4.0).
	reviewMerge := waggle.ParallelThen[[]models.FileChange, []models.Review, *models.AggregatedReview](
		"review-and-merge",
		agents.MergeReviewsFunc(),
		secAgent, styAgent, logAgent, prfAgent,
	)

	// Wrap agents with resilience.
	fetchAgent := agent.WithTimeout(
		agent.WithRetry(
			agents.NewFetchAgent(ghClient, cfg.Pipeline.MaxFilesPerPR),
			agent.WithMaxAttempts(cfg.Pipeline.MaxRetries+1),
		),
		30*time.Second,
	)

	summaryAgent := agent.WithTimeout(
		agents.NewSummaryAgent(registry.ForAgent("summary"), cfg.Pipeline.MemorySize),
		timeout,
	)

	// PostAgent with guardrails (v0.4.0 WithInputExtractGuard):
	postAgent := guardrail.WithInputExtractGuard(
		agent.WithRetry(
			agents.NewPostAgent(ghClient),
			agent.WithMaxAttempts(3),
		),
		func(review *models.AggregatedReview) string {
			return agents.FormatReviewComment(review)
		},
		guardrail.MaxLength(65000),
		guardrail.PIIEmail,
		guardrail.ContentFilter([]string{"api_key", "password", "secret_key", "private_key"}),
	)

	// Observability: event channel + metrics + stream observer.
	events := make(chan observe.Event, 256)
	metrics := observe.NewMetrics()
	go metrics.ConsumeEvents(events)

	observer := stream.ObserverFunc(func(s stream.Step) {
		slog.Debug("pipeline step", "agent", s.AgentName, "type", s.Type)
	})

	return &Pipeline{
		fetchAgent:   fetchAgent,
		reviewMerge:  reviewMerge,
		summaryAgent: summaryAgent,
		postAgent:    postAgent,
		metrics:      metrics,
		events:       events,
		observer:     observer,
	}, nil
}

// Metrics returns the pipeline's metrics collector.
func (p *Pipeline) Metrics() *observe.Metrics {
	return p.metrics
}

// Events returns the event channel for SSE broadcasting (web visualization).
func (p *Pipeline) Events() chan observe.Event {
	return p.events
}

// Run executes the full review pipeline for a PR event.
func (p *Pipeline) Run(ctx context.Context, event *github.PullRequestEvent) error {
	ref := event.PRRef()
	slog.Info("pipeline started", "pr", event.Number, "action", event.Action)

	// Emit workflow start event.
	p.emitEvent(observe.Event{
		Type:       observe.EventWorkflowStart,
		WorkflowID: fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.Number),
		Timestamp:  time.Now(),
	})

	// Set up PipelineContext (v0.4.0) to carry PRRef across stages.
	pctx := agent.NewPipelineContext()
	pctx.Set(agents.PRRefKey, ref)
	ctx = agent.WithPipelineCtx(ctx, pctx)

	// Stage 1: Fetch PR data.
	p.emitAgentStart("fetch", ref)
	start := time.Now()
	prData, err := p.fetchAgent.Run(ctx, event)
	if err != nil {
		p.emitAgentError("fetch", ref, time.Since(start), err)
		return fmt.Errorf("fetch: %w", err)
	}
	p.emitAgentEnd("fetch", ref, time.Since(start))

	// Stage 2: Skip if no reviewable files.
	if len(prData.Files) == 0 {
		slog.Info("no reviewable files, skipping", "pr", prData.Number)
		return nil
	}

	// Stage 3+4: Parallel review + merge.
	p.emitAgentStart("review-and-merge", ref)
	start = time.Now()
	aggregated, err := p.reviewMerge.Run(ctx, prData.Files)
	if err != nil {
		p.emitAgentError("review-and-merge", ref, time.Since(start), err)
		return fmt.Errorf("review+merge: %w", err)
	}
	aggregated.PRRef = ref
	p.emitAgentEnd("review-and-merge", ref, time.Since(start))

	// Stage 5: Summary.
	if len(aggregated.Issues) > 0 {
		p.emitAgentStart("summary", ref)
		start = time.Now()
		aggregated, err = p.summaryAgent.Run(ctx, aggregated)
		if err != nil {
			slog.Warn("summary generation failed", "err", err)
			aggregated.Summary = "Summary generation failed."
		}
		p.emitAgentEnd("summary", ref, time.Since(start))
	} else {
		aggregated.Summary = "No issues found. Code looks good!"
	}

	// Stage 6: Post to GitHub.
	p.emitAgentStart("post", ref)
	start = time.Now()
	_, err = p.postAgent.Run(ctx, aggregated)
	if err != nil {
		p.emitAgentError("post", ref, time.Since(start), err)
		return fmt.Errorf("post: %w", err)
	}
	p.emitAgentEnd("post", ref, time.Since(start))

	// Emit workflow end.
	p.emitEvent(observe.Event{
		Type:       observe.EventWorkflowEnd,
		WorkflowID: fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.Number),
		Timestamp:  time.Now(),
	})

	slog.Info("pipeline completed",
		"pr", prData.Number,
		"score", aggregated.Score,
		"issues", len(aggregated.Issues),
	)

	return nil
}

func (p *Pipeline) emitEvent(e observe.Event) {
	select {
	case p.events <- e:
	default:
		// Non-blocking: drop events if channel is full.
	}
}

func (p *Pipeline) emitAgentStart(name string, ref models.PRRef) {
	p.emitEvent(observe.NewAgentStartEvent(
		fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.Number), name, 0))
	p.observer.OnStep(stream.Step{AgentName: name, Type: stream.StepStarted, Timestamp: time.Now()})
}

func (p *Pipeline) emitAgentEnd(name string, ref models.PRRef, d time.Duration) {
	p.emitEvent(observe.NewAgentEndEvent(
		fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.Number), name, d, 0))
	p.observer.OnStep(stream.Step{AgentName: name, Type: stream.StepCompleted, Timestamp: time.Now()})
}

func (p *Pipeline) emitAgentError(name string, ref models.PRRef, d time.Duration, err error) {
	p.emitEvent(observe.NewAgentErrorEvent(
		fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.Number), name, d, err))
	p.observer.OnStep(stream.Step{AgentName: name, Type: stream.StepError, Content: err.Error(), Timestamp: time.Now()})
}
