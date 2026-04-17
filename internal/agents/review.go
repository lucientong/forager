package agents

import (
	"context"
	"log/slog"
	"strings"

	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/llm"
	"github.com/lucientong/waggle/pkg/output"
	"github.com/lucientong/waggle/pkg/prompt"
)

// newReviewAgent creates a review agent for a specific category.
// Each agent takes []FileChange, iterates over files, calls an LLM
// structured agent per file, and collects all reviews.
// Type: Agent[[]models.FileChange, []models.Review]
func newReviewAgent(
	name string,
	category models.Category,
	provider llm.Provider,
	promptTpl *prompt.Template,
) agent.Agent[[]models.FileChange, []models.Review] {
	// Create a per-file structured agent that parses []Review from LLM output.
	fileReviewer := output.NewStructuredAgent[string, []models.Review](
		name+"-llm",
		provider,
		func(renderedPrompt string) string { return renderedPrompt },
		output.WithMaxRetries(2),
	)

	return agent.Func[[]models.FileChange, []models.Review](name, func(ctx context.Context, files []models.FileChange) ([]models.Review, error) {
		var allReviews []models.Review

		for _, file := range files {
			if shouldSkip(file) {
				continue
			}

			rendered, err := promptTpl.
				WithVar("language", file.Language).
				WithVar("filename", file.Filename).
				WithVar("patch", file.Patch).
				Render()
			if err != nil {
				slog.Warn("failed to render prompt", "agent", name, "file", file.Filename, "err", err)
				continue
			}

			reviews, err := fileReviewer.Run(ctx, rendered)
			if err != nil {
				slog.Warn("review failed for file", "agent", name, "file", file.Filename, "err", err)
				continue // Don't fail the whole batch for one file.
			}

			// Ensure category is set correctly on all reviews.
			for i := range reviews {
				reviews[i].Category = category
				if reviews[i].File == "" {
					reviews[i].File = file.Filename
				}
			}

			allReviews = append(allReviews, reviews...)
		}

		return allReviews, nil
	})
}

// shouldSkip returns true for files that shouldn't be reviewed.
func shouldSkip(file models.FileChange) bool {
	// Skip removed files.
	if file.Status == "removed" {
		return true
	}
	// Skip empty patches.
	if file.Patch == "" {
		return true
	}
	// Skip vendor / generated / binary-like files.
	skipPrefixes := []string{"vendor/", "node_modules/", "dist/", "build/", ".git/"}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(file.Filename, prefix) {
			return true
		}
	}
	skipSuffixes := []string{".lock", ".sum", ".min.js", ".min.css", ".pb.go", ".gen.go", "_generated.go"}
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(file.Filename, suffix) {
			return true
		}
	}
	return false
}
