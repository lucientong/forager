package agents

import (
	"context"
	"log/slog"

	"github.com/lucientong/forager/internal/github"
	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
)

// NewPostAgent creates an agent that posts the review comment to GitHub.
// Type: Agent[*models.AggregatedReview, bool]
func NewPostAgent(ghClient *github.Client) agent.Agent[*models.AggregatedReview, bool] {
	return agent.Func[*models.AggregatedReview, bool]("post", func(ctx context.Context, review *models.AggregatedReview) (bool, error) {
		body := FormatReviewComment(review)

		slog.Info("posting review comment",
			"owner", review.PRRef.Owner,
			"repo", review.PRRef.Repo,
			"pr", review.PRRef.Number,
			"score", review.Score,
			"issues", len(review.Issues),
		)

		if err := ghClient.PostReviewComment(ctx, review.PRRef, body); err != nil {
			return false, err
		}
		return true, nil
	})
}
