package config

import "fmt"

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if err := c.GitHub.Validate(); err != nil {
		return fmt.Errorf("github config: %w", err)
	}
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}
	if err := c.Pipeline.Validate(); err != nil {
		return fmt.Errorf("pipeline config: %w", err)
	}
	if err := c.validateProviders(); err != nil {
		return fmt.Errorf("providers config: %w", err)
	}
	if err := c.validateAgents(); err != nil {
		return fmt.Errorf("agents config: %w", err)
	}
	return nil
}

// Validate checks if GitHub config is valid.
func (g *GitHubConfig) Validate() error {
	if g.Token == "" {
		return fmt.Errorf("token is required (set FORAGER_GITHUB_TOKEN)")
	}
	return nil
}

// Validate checks if Server config is valid.
func (s *ServerConfig) Validate() error {
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// Validate checks if Pipeline config is valid.
func (p *PipelineConfig) Validate() error {
	if p.MaxFilesPerPR <= 0 {
		return fmt.Errorf("max_files_per_pr must be positive")
	}
	if p.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout_seconds must be positive")
	}
	return nil
}

// validateProviders ensures at least one provider is configured and each
// non-ollama provider has an API key.
func (c *Config) validateProviders() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}
	for name, p := range c.Providers {
		switch name {
		case "anthropic", "openai", "ollama":
		default:
			return fmt.Errorf("unknown provider %q (must be anthropic, openai, or ollama)", name)
		}
		if name != "ollama" && p.APIKey == "" {
			return fmt.Errorf("provider %q requires api_key (set FORAGER_%s_API_KEY)", name, toEnvName(name))
		}
		if p.Model == "" {
			return fmt.Errorf("provider %q requires a model", name)
		}
	}
	return nil
}

// validateAgents ensures the default and each agent-specific provider reference
// exists in the providers map.
func (c *Config) validateAgents() error {
	if c.Agents.Default == "" {
		return fmt.Errorf("agents.default is required")
	}
	if _, ok := c.Providers[c.Agents.Default]; !ok {
		return fmt.Errorf("agents.default references unknown provider %q", c.Agents.Default)
	}
	// Validate agent-specific overrides.
	agentRoles := map[string]string{
		"security":    c.Agents.Security,
		"style":       c.Agents.Style,
		"logic":       c.Agents.Logic,
		"performance": c.Agents.Performance,
		"summary":     c.Agents.Summary,
	}
	for role, prov := range agentRoles {
		if prov == "" {
			continue // will use default
		}
		if _, ok := c.Providers[prov]; !ok {
			return fmt.Errorf("agents.%s references unknown provider %q", role, prov)
		}
	}
	// Validate fallback order references.
	for _, prov := range c.Agents.Fallback {
		if _, ok := c.Providers[prov]; !ok {
			return fmt.Errorf("fallback_order references unknown provider %q", prov)
		}
	}
	return nil
}

func toEnvName(provider string) string {
	switch provider {
	case "anthropic":
		return "ANTHROPIC"
	case "openai":
		return "OPENAI"
	default:
		return "LLM"
	}
}
