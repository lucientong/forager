package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucientong/forager/internal/config"
	"github.com/lucientong/forager/internal/github"
	llmpkg "github.com/lucientong/forager/internal/llm"
	"github.com/lucientong/forager/internal/pipeline"
	"github.com/lucientong/forager/internal/server"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Set up logging.
	setupLogging(cfg.Logging)

	// Validate config.
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

	// Create LLM provider registry (per-agent routing + fallback).
	registry, err := llmpkg.NewRegistry(cfg)
	if err != nil {
		slog.Error("failed to create LLM registry", "err", err)
		os.Exit(1)
	}

	// Create dependencies.
	ghClient := github.NewClient(cfg.GitHub)

	// Build pipeline.
	p, err := pipeline.New(cfg, ghClient, registry)
	if err != nil {
		slog.Error("failed to create pipeline", "err", err)
		os.Exit(1)
	}

	// Create main HTTP server.
	srv := server.New(cfg, p)

	// Optionally start waggle web visualization panel.
	webSrv := server.NewWebServer(cfg, p)
	if webSrv != nil {
		go func() {
			if err := webSrv.Start(); err != nil && err != http.ErrServerClosed {
				slog.Error("web server error", "err", err)
			}
		}()
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if webSrv != nil {
			_ = webSrv.Shutdown()
		}
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "err", err)
		}
	}()

	// Start server.
	slog.Info("forager starting",
		"port", cfg.Server.Port,
		"default_provider", cfg.Agents.Default,
		"providers", len(cfg.Providers),
	)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	slog.Info("forager stopped")
}

func setupLogging(cfg config.LoggingConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
