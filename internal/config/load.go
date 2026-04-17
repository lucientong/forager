package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Load reads a config file and applies environment variable overrides.
// If configPath is empty, it returns DefaultConfig with env overrides.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	// Apply environment variable overrides.
	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	// GitHub.
	if v := os.Getenv("FORAGER_GITHUB_TOKEN"); v != "" {
		cfg.GitHub.Token = v
	}
	if v := os.Getenv("FORAGER_GITHUB_API_URL"); v != "" {
		cfg.GitHub.APIURL = v
	}

	// Server.
	if v := os.Getenv("FORAGER_WEBHOOK_SECRET"); v != "" {
		cfg.Server.WebhookSecret = v
	}
	if v := os.Getenv("FORAGER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	// Provider API keys — inject into existing provider entries.
	if v := os.Getenv("FORAGER_ANTHROPIC_API_KEY"); v != "" {
		applyProviderKey(cfg, "anthropic", v)
	}
	if v := os.Getenv("FORAGER_OPENAI_API_KEY"); v != "" {
		applyProviderKey(cfg, "openai", v)
	}
	// Ollama base URL.
	if v := os.Getenv("FORAGER_OLLAMA_URL"); v != "" {
		if p, ok := cfg.Providers["ollama"]; ok {
			p.BaseURL = v
			cfg.Providers["ollama"] = p
		}
	}

	// Legacy single-provider env vars for simple setups.
	if v := os.Getenv("FORAGER_LLM_API_KEY"); v != "" {
		if p, ok := cfg.Providers[cfg.Agents.Default]; ok {
			p.APIKey = v
			cfg.Providers[cfg.Agents.Default] = p
		}
	}

	// Agent default override.
	if v := os.Getenv("FORAGER_AGENTS_DEFAULT"); v != "" {
		cfg.Agents.Default = v
	}

	// Logging.
	if v := os.Getenv("FORAGER_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
}

func applyProviderKey(cfg *Config, name, key string) {
	if cfg.Providers == nil {
		cfg.Providers = make(ProvidersConfig)
	}
	p, ok := cfg.Providers[name]
	if !ok {
		// Provider not in config file — create a minimal entry.
		// User can set model via config file; this just injects the key.
		return
	}
	p.APIKey = key
	cfg.Providers[name] = p
}
