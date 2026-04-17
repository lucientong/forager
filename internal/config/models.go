// Package config provides configuration loading and validation for Forager.
package config

import "time"

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	GitHub    GitHubConfig    `yaml:"github"`
	Providers ProvidersConfig `yaml:"providers"`
	Agents    AgentsConfig    `yaml:"agents"`
	Pipeline  PipelineConfig  `yaml:"pipeline"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int           `yaml:"port"`
	WebhookSecret   string        `yaml:"webhook_secret"`
	WebPort         int           `yaml:"web_port"`          // waggle visualization panel (0 = disabled)
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// GitHubConfig holds GitHub API settings.
type GitHubConfig struct {
	Token  string `yaml:"token"`
	APIURL string `yaml:"api_url"`
}

// ProviderConfig holds settings for a single LLM provider.
type ProviderConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
	MaxTokens int    `yaml:"max_tokens"`
}

// ProvidersConfig is a map of named provider configurations.
// Keys are provider names: "anthropic", "openai", "ollama".
type ProvidersConfig map[string]ProviderConfig

// AgentsConfig controls which provider each review agent uses.
type AgentsConfig struct {
	Default     string   `yaml:"default"`        // fallback provider for agents not listed
	Security    string   `yaml:"security"`       // provider name for security agent
	Style       string   `yaml:"style"`          // provider name for style agent
	Logic       string   `yaml:"logic"`          // provider name for logic agent
	Performance string   `yaml:"performance"`    // provider name for performance agent
	Summary     string   `yaml:"summary"`        // provider name for summary agent
	Fallback    []string `yaml:"fallback_order"` // failover order, e.g. ["anthropic", "openai", "ollama"]
}

// ProviderForAgent returns the configured provider name for a given agent role.
// Falls back to Default if the agent-specific field is empty.
func (a *AgentsConfig) ProviderForAgent(agentRole string) string {
	var specific string
	switch agentRole {
	case "security":
		specific = a.Security
	case "style":
		specific = a.Style
	case "logic":
		specific = a.Logic
	case "performance":
		specific = a.Performance
	case "summary":
		specific = a.Summary
	}
	if specific != "" {
		return specific
	}
	return a.Default
}

// PipelineConfig holds code review pipeline settings.
type PipelineConfig struct {
	MaxRetries     int `yaml:"max_retries"`
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxFilesPerPR  int `yaml:"max_files_per_pr"`
	MemorySize     int `yaml:"memory_size"` // per-PR conversation history window (0 = disabled)
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		GitHub: GitHubConfig{
			APIURL: "https://api.github.com",
		},
		Providers: ProvidersConfig{
			"anthropic": {
				Model:     "claude-3-5-sonnet-20241022",
				MaxTokens: 8096,
			},
		},
		Agents: AgentsConfig{
			Default:  "anthropic",
			Fallback: []string{"anthropic", "openai", "ollama"},
		},
		Pipeline: PipelineConfig{
			MaxRetries:     2,
			TimeoutSeconds: 120,
			MaxFilesPerPR:  50,
			MemorySize:     20,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
