package agents

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/waggle"
)

// helper to construct ParallelResults for testing.
func waggleParallelResults(results [][]models.Review, errs []error) waggle.ParallelResults[[]models.Review] {
	return waggle.ParallelResults[[]models.Review]{Results: results, Errors: errs}
}

func TestMergeAgent(t *testing.T) {
	mergeAgent := NewMergeAgent()

	// Set up PipelineContext with PRRef.
	pctx := agent.NewPipelineContext()
	pctx.Set(PRRefKey, models.PRRef{Owner: "test", Repo: "repo", Number: 1})
	ctx := agent.WithPipelineCtx(context.Background(), pctx)

	input := [][]models.Review{
		{
			{File: "main.go", Category: models.CategorySecurity, Severity: models.SeverityCritical, Line: 10, Message: "SQL injection"},
			{File: "main.go", Category: models.CategorySecurity, Severity: models.SeverityInfo, Line: 10, Message: "duplicate lower sev"},
		},
		{
			{File: "main.go", Category: models.CategoryStyle, Severity: models.SeverityWarning, Line: 5, Message: "bad naming"},
		},
		{}, // empty batch
		{
			{File: "util.go", Category: models.CategoryPerformance, Severity: models.SeverityInfo, Line: 20, Message: "could cache"},
		},
	}

	result, err := mergeAgent.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have deduplicated: main.go:10:security -> keep critical.
	if len(result.Issues) != 3 {
		t.Errorf("expected 3 issues after dedup, got %d", len(result.Issues))
	}

	// First issue should be the critical one.
	if result.Issues[0].Severity != models.SeverityCritical {
		t.Errorf("expected first issue to be critical, got %s", result.Issues[0].Severity)
	}

	// Score should be reduced from 10.
	if result.Score >= 10 {
		t.Errorf("expected score < 10 (has critical issue), got %d", result.Score)
	}

	// PRRef should be read from PipelineContext.
	if result.PRRef.Owner != "test" {
		t.Errorf("expected PRRef.Owner=test, got %s", result.PRRef.Owner)
	}
}

func TestMergeReviewsFunc(t *testing.T) {
	// Test the MergeFunc used with ParallelThen.
	mergeFn := MergeReviewsFunc()

	input := waggleParallelResults(
		[][]models.Review{
			{{File: "a.go", Category: models.CategorySecurity, Severity: models.SeverityCritical, Message: "issue1"}},
			{{File: "b.go", Category: models.CategoryStyle, Severity: models.SeverityInfo, Message: "issue2"}},
		},
		[]error{nil, nil},
	)

	result, err := mergeFn(input)
	if err != nil {
		t.Fatalf("MergeReviewsFunc: %v", err)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Score >= 10 {
		t.Errorf("expected score < 10, got %d", result.Score)
	}
}

func TestMergeReviewsFuncSkipsErrors(t *testing.T) {
	mergeFn := MergeReviewsFunc()

	input := waggleParallelResults(
		[][]models.Review{
			{{File: "a.go", Category: models.CategorySecurity, Severity: models.SeverityWarning, Message: "ok"}},
			nil, // failed agent
		},
		[]error{nil, fmt.Errorf("agent failed")},
	)

	result, err := mergeFn(input)
	if err != nil {
		t.Fatalf("MergeReviewsFunc: %v", err)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue (skipping failed agent), got %d", len(result.Issues))
	}
}

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name    string
		reviews []models.Review
		want    int
	}{
		{"no issues", nil, 10},
		{"one critical", []models.Review{{Severity: models.SeverityCritical}}, 7},
		{"four criticals", []models.Review{
			{Severity: models.SeverityCritical},
			{Severity: models.SeverityCritical},
			{Severity: models.SeverityCritical},
			{Severity: models.SeverityCritical},
		}, 1}, // clamped to 1
		{"five infos", []models.Review{
			{Severity: models.SeverityInfo},
			{Severity: models.SeverityInfo},
			{Severity: models.SeverityInfo},
			{Severity: models.SeverityInfo},
			{Severity: models.SeverityInfo},
		}, 9}, // 5/4 = 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeScore(tt.reviews)
			if got != tt.want {
				t.Errorf("computeScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		file models.FileChange
		want bool
	}{
		{models.FileChange{Filename: "main.go", Status: "added", Patch: "+code"}, false},
		{models.FileChange{Filename: "main.go", Status: "removed"}, true},
		{models.FileChange{Filename: "main.go", Status: "modified", Patch: ""}, true},
		{models.FileChange{Filename: "vendor/lib/x.go", Status: "added", Patch: "+code"}, true},
		{models.FileChange{Filename: "go.sum", Status: "modified", Patch: "+hash"}, true},
		{models.FileChange{Filename: "api.pb.go", Status: "modified", Patch: "+gen"}, true},
	}
	for _, tt := range tests {
		if got := shouldSkip(tt.file); got != tt.want {
			t.Errorf("shouldSkip(%q) = %v, want %v", tt.file.Filename, got, tt.want)
		}
	}
}

func TestFormatReviewComment(t *testing.T) {
	review := &models.AggregatedReview{
		PRRef:   models.PRRef{Owner: "test", Repo: "repo", Number: 1},
		Score:   7,
		Summary: "The code has a few issues.",
		Issues: []models.Review{
			{File: "main.go", Category: models.CategorySecurity, Severity: models.SeverityCritical, Line: 10, Message: "SQL injection", Suggestion: "Use parameterized queries"},
			{File: "util.go", Category: models.CategoryStyle, Severity: models.SeverityInfo, Message: "Consider renaming"},
		},
	}

	body := FormatReviewComment(review)
	if !strings.Contains(body, "7/10") {
		t.Error("expected score in output")
	}
	if !strings.Contains(body, "SQL injection") {
		t.Error("expected critical issue in output")
	}
	if !strings.Contains(body, "parameterized queries") {
		t.Error("expected suggestion in output")
	}
	if !strings.Contains(body, "Forager") {
		t.Error("expected footer in output")
	}
}
