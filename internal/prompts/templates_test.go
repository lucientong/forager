package prompts

import (
	"strings"
	"testing"

	"github.com/lucientong/waggle/pkg/prompt"
)

func TestTemplatesRender(t *testing.T) {
	templates := map[string]*prompt.Template{
		"security":    SecurityTemplate,
		"style":       StyleTemplate,
		"logic":       LogicTemplate,
		"performance": PerformanceTemplate,
	}

	for name, tmpl := range templates {
		t.Run(name, func(t *testing.T) {
			rendered, err := tmpl.
				WithVar("language", "go").
				WithVar("filename", "main.go").
				WithVar("patch", "+func hello() {}").
				Render()
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			if !strings.Contains(rendered, "main.go") {
				t.Error("rendered template should contain filename")
			}
			if !strings.Contains(rendered, "+func hello() {}") {
				t.Error("rendered template should contain patch")
			}
		})
	}
}

func TestSummaryTemplateRender(t *testing.T) {
	rendered, err := SummaryTemplate.
		WithVar("issue_count", "5").
		WithVar("critical_count", "1").
		WithVar("warning_count", "2").
		WithVar("info_count", "2").
		WithVar("issues_json", "[]").
		Render()
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(rendered, "5") {
		t.Error("rendered template should contain issue count")
	}
}
