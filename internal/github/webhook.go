package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lucientong/forager/internal/models"
)

// PullRequestEvent represents a GitHub pull_request webhook event.
type PullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"pull_request"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

// PRRef extracts a models.PRRef from the event.
func (e *PullRequestEvent) PRRef() models.PRRef {
	return models.PRRef{
		Owner:  e.Repository.Owner.Login,
		Repo:   e.Repository.Name,
		Number: e.Number,
	}
}

// IsReviewable returns true if this event should trigger a review.
func (e *PullRequestEvent) IsReviewable() bool {
	switch e.Action {
	case "opened", "synchronize", "reopened":
		return true
	default:
		return false
	}
}

// ParsePullRequestEvent parses a webhook payload into a PullRequestEvent.
func ParsePullRequestEvent(body []byte) (*PullRequestEvent, error) {
	var event PullRequestEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("parse webhook event: %w", err)
	}
	return &event, nil
}

// VerifySignature validates X-Hub-Signature-256 against the request body.
func VerifySignature(payload []byte, signature string, secret string) error {
	if secret == "" {
		return nil // no secret configured, skip verification
	}
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
