// Package models defines the core data types used throughout the Forager pipeline.
package models

// PRRef identifies a specific pull request.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

// PRData holds all the data fetched for a pull request.
type PRData struct {
	PRRef
	Title   string
	Body    string
	Diff    string
	Files   []FileChange
	Commits []string
}

// FileChange represents a single changed file in a pull request.
type FileChange struct {
	Filename string
	Language string
	Patch    string
	Status   string // added, modified, removed, renamed
}

// Category is the type of review issue.
type Category string

const (
	CategorySecurity    Category = "security"
	CategoryStyle       Category = "style"
	CategoryLogic       Category = "logic"
	CategoryPerformance Category = "performance"
)

// Severity indicates how serious an issue is.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Review is a single finding from a review agent.
type Review struct {
	File       string   `json:"file"`
	Category   Category `json:"category"`
	Severity   Severity `json:"severity"`
	Line       int      `json:"line,omitempty"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// AggregatedReview is the combined output of all review agents for a PR.
type AggregatedReview struct {
	PRRef   PRRef
	Score   int      `json:"score"`
	Issues  []Review `json:"issues"`
	Summary string   `json:"summary"`
}
