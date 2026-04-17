// Package server provides the HTTP server for Forager.
package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/lucientong/forager/internal/config"
	"github.com/lucientong/forager/internal/pipeline"
	"github.com/lucientong/waggle/pkg/observe"
	"github.com/lucientong/waggle/pkg/web"
)

// New creates a configured HTTP server.
func New(cfg *config.Config, p *pipeline.Pipeline) *http.Server {
	mux := http.NewServeMux()

	// Webhook endpoint.
	mux.HandleFunc("POST /webhook", NewWebhookHandler(p, cfg.Server.WebhookSecret))

	// Health check.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Prometheus metrics.
	mux.Handle("GET /metrics", observe.PrometheusHandler(p.Metrics()))

	slog.Info("server configured", "port", cfg.Server.Port)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}

// NewWebServer creates the waggle visualization panel server (optional).
// Returns nil if web_port is 0 or not configured.
func NewWebServer(cfg *config.Config, p *pipeline.Pipeline) *web.Server {
	if cfg.Server.WebPort <= 0 {
		return nil
	}

	webCfg := web.DefaultConfig()
	webCfg.Addr = fmt.Sprintf(":%d", cfg.Server.WebPort)

	ws := web.NewServer(webCfg, nil, p.Metrics())

	// Connect pipeline events to the web SSE hub.
	go func() {
		for event := range p.Events() {
			ws.PublishEvent(event)
		}
	}()

	slog.Info("web visualization server configured", "port", cfg.Server.WebPort)
	return ws
}
