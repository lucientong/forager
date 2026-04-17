// Package agents provides the individual waggle agents for the Forager pipeline.
package agents

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lucientong/forager/internal/github"
	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
)

// NewFetchAgent creates an agent that fetches PR data from GitHub.
// Type: Agent[*github.PullRequestEvent, *models.PRData]
func NewFetchAgent(ghClient *github.Client, maxFiles int) agent.Agent[*github.PullRequestEvent, *models.PRData] {
	return agent.Func[*github.PullRequestEvent, *models.PRData]("fetch", func(ctx context.Context, event *github.PullRequestEvent) (*models.PRData, error) {
		ref := event.PRRef()

		slog.Info("fetching PR data", "owner", ref.Owner, "repo", ref.Repo, "number", ref.Number)

		prData, err := ghClient.GetPullRequest(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("fetch PR %d: %w", ref.Number, err)
		}

		// Limit the number of files to review.
		if maxFiles > 0 && len(prData.Files) > maxFiles {
			slog.Warn("truncating files list",
				"total", len(prData.Files),
				"limit", maxFiles,
				"pr", ref.Number,
			)
			prData.Files = prData.Files[:maxFiles]
		}

		slog.Info("fetched PR data",
			"pr", ref.Number,
			"files", len(prData.Files),
			"commits", len(prData.Commits),
		)

		return prData, nil
	})
}
