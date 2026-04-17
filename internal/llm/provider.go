// Package llm provides a provider registry that creates waggle LLM providers
// from Forager config. It supports per-agent provider selection with fallback routing.
package llm

import (
	"fmt"
	"log/slog"

	"github.com/lucientong/forager/internal/config"
	waggle_llm "github.com/lucientong/waggle/pkg/llm"
)

// Registry holds named LLM providers and resolves per-agent routing.
type Registry struct {
	providers map[string]waggle_llm.Provider
	agents    config.AgentsConfig
}

// NewRegistry builds all configured providers and returns a Registry.
func NewRegistry(cfg *config.Config) (*Registry, error) {
	providers := make(map[string]waggle_llm.Provider, len(cfg.Providers))

	for name, pcfg := range cfg.Providers {
		p, err := buildProvider(name, pcfg)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", name, err)
		}
		providers[name] = p
		slog.Info("registered LLM provider", "name", name, "model", pcfg.Model)
	}

	// Build failover routers: for each provider, create a router that tries
	// that provider first then falls back through the fallback order.
	if len(cfg.Agents.Fallback) > 1 {
		for name := range providers {
			ordered := buildFallbackOrder(name, cfg.Agents.Fallback, providers)
			if len(ordered) > 1 {
				providers[name] = waggle_llm.NewRouter(ordered, waggle_llm.WithRoutingStrategy(waggle_llm.StrategyFailover))
				slog.Info("provider wrapped with failover router", "primary", name, "fallback_count", len(ordered)-1)
			}
		}
	}

	return &Registry{
		providers: providers,
		agents:    cfg.Agents,
	}, nil
}

// ForAgent returns the LLM provider configured for the given agent role.
// Agent roles: "security", "style", "logic", "performance", "summary".
func (r *Registry) ForAgent(agentRole string) waggle_llm.Provider {
	name := r.agents.ProviderForAgent(agentRole)
	if p, ok := r.providers[name]; ok {
		return p
	}
	// Fallback to default (should not happen after validation).
	return r.providers[r.agents.Default]
}

// Default returns the default provider.
func (r *Registry) Default() waggle_llm.Provider {
	return r.providers[r.agents.Default]
}

// buildProvider creates a single waggle LLM provider from config.
func buildProvider(name string, cfg config.ProviderConfig) (waggle_llm.Provider, error) {
	switch name {
	case "anthropic":
		opts := []waggle_llm.AnthropicOption{}
		if cfg.Model != "" {
			opts = append(opts, waggle_llm.WithAnthropicModel(cfg.Model))
		}
		if cfg.MaxTokens > 0 {
			opts = append(opts, waggle_llm.WithAnthropicMaxTokens(cfg.MaxTokens))
		}
		if cfg.BaseURL != "" {
			opts = append(opts, waggle_llm.WithAnthropicBaseURL(cfg.BaseURL))
		}
		return waggle_llm.NewAnthropic(cfg.APIKey, opts...), nil

	case "openai":
		opts := []waggle_llm.OpenAIOption{}
		if cfg.Model != "" {
			opts = append(opts, waggle_llm.WithOpenAIModel(cfg.Model))
		}
		if cfg.BaseURL != "" {
			opts = append(opts, waggle_llm.WithOpenAIBaseURL(cfg.BaseURL))
		}
		return waggle_llm.NewOpenAI(cfg.APIKey, opts...), nil

	case "ollama":
		opts := []waggle_llm.OllamaOption{}
		if cfg.Model != "" {
			opts = append(opts, waggle_llm.WithOllamaModel(cfg.Model))
		}
		if cfg.BaseURL != "" {
			opts = append(opts, waggle_llm.WithOllamaBaseURL(cfg.BaseURL))
		}
		return waggle_llm.NewOllama(opts...), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %q", name)
	}
}

// buildFallbackOrder returns providers ordered for failover:
// primary first, then the rest of fallback order (skipping primary and missing).
func buildFallbackOrder(primary string, fallbackOrder []string, providers map[string]waggle_llm.Provider) []waggle_llm.Provider {
	seen := map[string]bool{}
	var ordered []waggle_llm.Provider

	// Primary first.
	if p, ok := providers[primary]; ok {
		ordered = append(ordered, p)
		seen[primary] = true
	}

	// Then fallback order.
	for _, name := range fallbackOrder {
		if seen[name] {
			continue
		}
		if p, ok := providers[name]; ok {
			ordered = append(ordered, p)
			seen[name] = true
		}
	}

	return ordered
}
