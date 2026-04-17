package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/lucientong/forager/internal/github"
	"github.com/lucientong/forager/internal/pipeline"
)

// NewWebhookHandler creates an HTTP handler for GitHub PR webhooks.
func NewWebhookHandler(p *pipeline.Pipeline, webhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read body (limit to 10MB).
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify signature.
		sig := r.Header.Get("X-Hub-Signature-256")
		if err := github.VerifySignature(body, sig, webhookSecret); err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		// Check event type.
		eventType := r.Header.Get("X-GitHub-Event")
		if eventType != "pull_request" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Parse event.
		event, err := github.ParsePullRequestEvent(body)
		if err != nil {
			slog.Error("failed to parse webhook", "err", err)
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}

		// Filter actions.
		if !event.IsReviewable() {
			w.WriteHeader(http.StatusOK)
			return
		}

		slog.Info("received reviewable PR event",
			"action", event.Action,
			"pr", event.Number,
			"repo", event.Repository.Name,
		)

		// Respond 202 immediately, run pipeline async.
		w.WriteHeader(http.StatusAccepted)

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := p.Run(ctx, event); err != nil {
				slog.Error("pipeline failed", "pr", event.Number, "err", err)
			}
		}()
	}
}
