package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load empty path: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Agents.Default != "anthropic" {
		t.Errorf("expected default agent provider anthropic, got %s", cfg.Agents.Default)
	}
	if cfg.Pipeline.MaxFilesPerPR != 50 {
		t.Errorf("expected default max_files_per_pr 50, got %d", cfg.Pipeline.MaxFilesPerPR)
	}
	if _, ok := cfg.Providers["anthropic"]; !ok {
		t.Error("expected anthropic provider in defaults")
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
server:
  port: 9090
  webhook_secret: "test-secret"
github:
  token: "gh-token"
  api_url: "https://github.example.com/api/v3"
providers:
  anthropic:
    api_key: "sk-ant-test"
    model: "claude-3-5-sonnet-20241022"
    max_tokens: 4096
  openai:
    api_key: "sk-openai-test"
    model: "gpt-4o"
agents:
  default: "anthropic"
  security: "anthropic"
  style: "openai"
  fallback_order: ["anthropic", "openai"]
pipeline:
  max_retries: 3
  timeout_seconds: 60
  max_files_per_pr: 25
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port: want 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.WebhookSecret != "test-secret" {
		t.Errorf("webhook_secret: want test-secret, got %s", cfg.Server.WebhookSecret)
	}
	if cfg.GitHub.Token != "gh-token" {
		t.Errorf("github.token: want gh-token, got %s", cfg.GitHub.Token)
	}
	if cfg.Providers["anthropic"].APIKey != "sk-ant-test" {
		t.Errorf("providers.anthropic.api_key: want sk-ant-test, got %s", cfg.Providers["anthropic"].APIKey)
	}
	if cfg.Providers["openai"].Model != "gpt-4o" {
		t.Errorf("providers.openai.model: want gpt-4o, got %s", cfg.Providers["openai"].Model)
	}
	if cfg.Agents.Style != "openai" {
		t.Errorf("agents.style: want openai, got %s", cfg.Agents.Style)
	}
	if cfg.Pipeline.MaxFilesPerPR != 25 {
		t.Errorf("pipeline.max_files_per_pr: want 25, got %d", cfg.Pipeline.MaxFilesPerPR)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("FORAGER_GITHUB_TOKEN", "env-token")
	t.Setenv("FORAGER_ANTHROPIC_API_KEY", "env-ant-key")
	t.Setenv("FORAGER_PORT", "3000")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GitHub.Token != "env-token" {
		t.Errorf("github.token: want env-token, got %s", cfg.GitHub.Token)
	}
	if cfg.Providers["anthropic"].APIKey != "env-ant-key" {
		t.Errorf("anthropic.api_key: want env-ant-key, got %s", cfg.Providers["anthropic"].APIKey)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("port: want 3000, got %d", cfg.Server.Port)
	}
}

func TestLegacyLLMAPIKeyEnv(t *testing.T) {
	t.Setenv("FORAGER_LLM_API_KEY", "legacy-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Legacy env should apply to the default provider.
	if cfg.Providers["anthropic"].APIKey != "legacy-key" {
		t.Errorf("legacy api_key: want legacy-key, got %s", cfg.Providers["anthropic"].APIKey)
	}
}

func TestProviderForAgent(t *testing.T) {
	agents := AgentsConfig{
		Default:  "anthropic",
		Security: "anthropic",
		Style:    "openai",
	}

	tests := []struct {
		role string
		want string
	}{
		{"security", "anthropic"},
		{"style", "openai"},
		{"logic", "anthropic"},       // falls back to default
		{"performance", "anthropic"}, // falls back to default
		{"summary", "anthropic"},     // falls back to default
	}

	for _, tt := range tests {
		got := agents.ProviderForAgent(tt.role)
		if got != tt.want {
			t.Errorf("ProviderForAgent(%q) = %q, want %q", tt.role, got, tt.want)
		}
	}
}

func TestValidation(t *testing.T) {
	validCfg := func() Config {
		cfg := DefaultConfig()
		cfg.GitHub.Token = "t"
		cfg.Providers = ProvidersConfig{
			"anthropic": {APIKey: "k", Model: "claude-3-5-sonnet-20241022"},
		}
		cfg.Agents = AgentsConfig{Default: "anthropic", Fallback: []string{"anthropic"}}
		return cfg
	}

	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"valid", func(c *Config) {}, false},
		{"missing github token", func(c *Config) { c.GitHub.Token = "" }, true},
		{"missing api_key for anthropic", func(c *Config) {
			c.Providers["anthropic"] = ProviderConfig{Model: "m"} // no key
		}, true},
		{"ollama no key needed", func(c *Config) {
			c.Providers = ProvidersConfig{"ollama": {Model: "llama3.2"}}
			c.Agents.Default = "ollama"
			c.Agents.Fallback = []string{"ollama"}
		}, false},
		{"unknown provider name", func(c *Config) {
			c.Providers["grok"] = ProviderConfig{APIKey: "k", Model: "m"}
		}, true},
		{"agent references missing provider", func(c *Config) {
			c.Agents.Security = "openai" // openai not in providers
		}, true},
		{"fallback references missing provider", func(c *Config) {
			c.Agents.Fallback = []string{"anthropic", "openai"} // openai not in providers
		}, true},
		{"bad port", func(c *Config) { c.Server.Port = 0 }, true},
		{"no providers", func(c *Config) { c.Providers = ProvidersConfig{} }, true},
		{"missing default", func(c *Config) { c.Agents.Default = "" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validCfg()
			tt.modify(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
