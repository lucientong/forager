package agents

import (
	"context"
	"fmt"
	"sort"

	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/waggle"
)

// PRRefKey is the PipelineContext key for storing/retrieving PRRef.
const PRRefKey = "pr_ref"

// MergeReviews is a waggle.MergeFunc that flattens, deduplicates, sorts,
// and scores parallel review results. It reads PRRef from PipelineContext.
//
// Designed for use with waggle.ParallelThen:
//
//	reviewPipeline := waggle.ParallelThen("reviewers", agents.MergeReviewsFunc(), ...)
func MergeReviewsFunc() waggle.MergeFunc[[]models.Review, *models.AggregatedReview] {
	return func(pr waggle.ParallelResults[[]models.Review]) (*models.AggregatedReview, error) {
		// Flatten all successful results.
		var all []models.Review
		for i, batch := range pr.Results {
			if pr.Errors[i] != nil {
				continue // skip failed agents
			}
			all = append(all, batch...)
		}

		all = dedup(all)
		sortReviews(all)
		score := computeScore(all)

		return &models.AggregatedReview{
			Score:  score,
			Issues: all,
		}, nil
	}
}

// NewMergeAgent creates a standalone merge agent for use outside ParallelThen.
// Reads PRRef from PipelineContext.
// Type: Agent[[][]models.Review, *models.AggregatedReview]
func NewMergeAgent() agent.Agent[[][]models.Review, *models.AggregatedReview] {
	return agent.Func[[][]models.Review, *models.AggregatedReview]("merge", func(ctx context.Context, batches [][]models.Review) (*models.AggregatedReview, error) {
		var all []models.Review
		for _, batch := range batches {
			all = append(all, batch...)
		}

		all = dedup(all)
		sortReviews(all)
		score := computeScore(all)

		result := &models.AggregatedReview{
			Score:  score,
			Issues: all,
		}

		// Read PRRef from PipelineContext if available.
		if pctx := agent.PipelineCtxFrom(ctx); pctx != nil {
			if ref, ok := agent.PipelineGet[models.PRRef](pctx, PRRefKey); ok {
				result.PRRef = ref
			}
		}

		return result, nil
	})
}

// dedup removes duplicate reviews (same file+line+category),
// keeping the one with highest severity.
func dedup(reviews []models.Review) []models.Review {
	type key struct {
		File     string
		Line     int
		Category models.Category
	}
	best := make(map[key]models.Review)
	for _, r := range reviews {
		k := key{r.File, r.Line, r.Category}
		existing, ok := best[k]
		if !ok || severityRank(r.Severity) > severityRank(existing.Severity) {
			best[k] = r
		}
	}
	result := make([]models.Review, 0, len(best))
	for _, r := range best {
		result = append(result, r)
	}
	return result
}

// sortReviews sorts by severity (critical first) then by file name.
func sortReviews(reviews []models.Review) {
	sort.Slice(reviews, func(i, j int) bool {
		ri, rj := severityRank(reviews[i].Severity), severityRank(reviews[j].Severity)
		if ri != rj {
			return ri > rj
		}
		if reviews[i].File != reviews[j].File {
			return reviews[i].File < reviews[j].File
		}
		return reviews[i].Line < reviews[j].Line
	})
}

func severityRank(s models.Severity) int {
	switch s {
	case models.SeverityCritical:
		return 3
	case models.SeverityWarning:
		return 2
	case models.SeverityInfo:
		return 1
	default:
		return 0
	}
}

// computeScore calculates a quality score from 1-10.
func computeScore(reviews []models.Review) int {
	score := 10
	for _, r := range reviews {
		switch r.Severity {
		case models.SeverityCritical:
			score -= 3
		case models.SeverityWarning:
			score -= 1
		}
	}
	infoCount := 0
	for _, r := range reviews {
		if r.Severity == models.SeverityInfo {
			infoCount++
		}
	}
	score -= infoCount / 4
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}
	return score
}

// CountBySeverity returns counts of issues by severity.
func CountBySeverity(reviews []models.Review) (critical, warning, info int) {
	for _, r := range reviews {
		switch r.Severity {
		case models.SeverityCritical:
			critical++
		case models.SeverityWarning:
			warning++
		case models.SeverityInfo:
			info++
		}
	}
	return
}

// FormatReviewComment generates a markdown-formatted review comment.
func FormatReviewComment(review *models.AggregatedReview) string {
	critical, warning, info := CountBySeverity(review.Issues)

	scoreEmoji := "🟢"
	if review.Score <= 3 {
		scoreEmoji = "🔴"
	} else if review.Score <= 6 {
		scoreEmoji = "🟡"
	}

	s := fmt.Sprintf("## %s Forager Code Review\n\n", scoreEmoji)
	s += fmt.Sprintf("**Score: %d/10**\n\n", review.Score)

	if review.Summary != "" {
		s += "### Summary\n\n"
		s += review.Summary + "\n\n"
	}

	total := len(review.Issues)
	if total == 0 {
		s += "No issues found. Code looks good! 🎉\n"
	} else {
		s += fmt.Sprintf("### Issues Found (%d)\n\n", total)

		if critical > 0 {
			s += fmt.Sprintf("#### 🔴 Critical (%d)\n\n", critical)
			for _, r := range review.Issues {
				if r.Severity == models.SeverityCritical {
					s += formatIssue(r)
				}
			}
		}
		if warning > 0 {
			s += fmt.Sprintf("#### 🟡 Warning (%d)\n\n", warning)
			for _, r := range review.Issues {
				if r.Severity == models.SeverityWarning {
					s += formatIssue(r)
				}
			}
		}
		if info > 0 {
			s += fmt.Sprintf("#### ℹ️ Info (%d)\n\n", info)
			for _, r := range review.Issues {
				if r.Severity == models.SeverityInfo {
					s += formatIssue(r)
				}
			}
		}
	}

	s += "\n---\n*Generated by [Forager](https://github.com/lucientong/forager) — Powered by [waggle](https://github.com/lucientong/waggle)*\n"
	return s
}

func formatIssue(r models.Review) string {
	location := r.File
	if r.Line > 0 {
		location = fmt.Sprintf("%s:%d", r.File, r.Line)
	}
	s := fmt.Sprintf("- **%s** — %s\n", location, r.Message)
	if r.Suggestion != "" {
		s += fmt.Sprintf("  > 💡 %s\n", r.Suggestion)
	}
	return s
}
