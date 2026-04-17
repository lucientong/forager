package agents

import (
	"github.com/lucientong/forager/internal/models"
	"github.com/lucientong/forager/internal/prompts"
	"github.com/lucientong/waggle/pkg/agent"
	"github.com/lucientong/waggle/pkg/llm"
)

// NewSecurityAgent creates a security review agent.
func NewSecurityAgent(provider llm.Provider) agent.Agent[[]models.FileChange, []models.Review] {
	return newReviewAgent("security-reviewer", models.CategorySecurity, provider, prompts.SecurityTemplate)
}

// NewStyleAgent creates a style review agent.
func NewStyleAgent(provider llm.Provider) agent.Agent[[]models.FileChange, []models.Review] {
	return newReviewAgent("style-reviewer", models.CategoryStyle, provider, prompts.StyleTemplate)
}

// NewLogicAgent creates a logic review agent.
func NewLogicAgent(provider llm.Provider) agent.Agent[[]models.FileChange, []models.Review] {
	return newReviewAgent("logic-reviewer", models.CategoryLogic, provider, prompts.LogicTemplate)
}

// NewPerformanceAgent creates a performance review agent.
func NewPerformanceAgent(provider llm.Provider) agent.Agent[[]models.FileChange, []models.Review] {
	return newReviewAgent("perf-reviewer", models.CategoryPerformance, provider, prompts.PerformanceTemplate)
}
