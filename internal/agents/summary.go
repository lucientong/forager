package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/forager/internal/prompts"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/llm"
	"github.com/lucientong/waggle/pkg/memory"
)

// NewSummaryAgent creates an agent that generates a human-readable summary
// using an LLM. If memorySize > 0, it retains PR context so summaries
// can reference prior review history within the same session.
// Type: Agent[*models.AggregatedReview, *models.AggregatedReview]
func NewSummaryAgent(provider llm.Provider, memorySize int) agent.Agent[*models.AggregatedReview, *models.AggregatedReview] {
	// Build LLMAgent with optional memory.
	opts := []llm.LLMAgentOption{}
	if memorySize > 0 {
		opts = append(opts, llm.WithMemory(memory.NewWindowStore(memorySize)))
	}

	summarizer := llm.NewLLMAgent[string]("summary-llm", provider,
		func(ctx context.Context, renderedPrompt string) ([]llm.Message, error) {
			return []llm.Message{
				{Role: llm.RoleSystem, Content: "You are a senior code reviewer writing a concise review summary. " +
					"If you have context from prior reviews in this session, reference patterns or recurring issues."},
				{Role: llm.RoleUser, Content: renderedPrompt},
			}, nil
		},
		opts...,
	)

	return agent.Func[*models.AggregatedReview, *models.AggregatedReview]("summary", func(ctx context.Context, review *models.AggregatedReview) (*models.AggregatedReview, error) {
		critical, warning, info := CountBySeverity(review.Issues)

		issuesJSON, err := json.Marshal(review.Issues)
		if err != nil {
			return nil, fmt.Errorf("marshal issues: %w", err)
		}

		rendered, err := prompts.SummaryTemplate.
			WithVar("issue_count", fmt.Sprintf("%d", len(review.Issues))).
			WithVar("critical_count", fmt.Sprintf("%d", critical)).
			WithVar("warning_count", fmt.Sprintf("%d", warning)).
			WithVar("info_count", fmt.Sprintf("%d", info)).
			WithVar("issues_json", string(issuesJSON)).
			Render()
		if err != nil {
			return nil, fmt.Errorf("render summary prompt: %w", err)
		}

		summary, err := summarizer.Run(ctx, rendered)
		if err != nil {
			slog.Warn("summary generation failed", "err", err)
			review.Summary = "Summary generation failed."
			return review, nil // non-fatal
		}

		review.Summary = summary
		return review, nil
	})
}
