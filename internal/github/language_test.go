package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestInferLanguage(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"types.ts", "typescript"},
		{"Main.java", "java"},
		{"code.rs", "rust"},
		{"style.css", "css"},
		{"query.sql", "sql"},
		{"main.tf", "terraform"},
		{"unknown.xyz", ""},
		{"Makefile", ""},
	}

	for _, tt := range tests {
		got := InferLanguage(tt.filename)
		if got != tt.expected {
			t.Errorf("InferLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
		}
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened"}`)

	// Empty signature with secret configured -> error.
	if err := VerifySignature(body, "", secret); err == nil {
		t.Error("expected error for empty signature")
	}

	// No secret configured -> skip verification.
	if err := VerifySignature(body, "", ""); err != nil {
		t.Errorf("expected no error when secret is empty, got %v", err)
	}

	// Valid signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if err := VerifySignature(body, validSig, secret); err != nil {
		t.Errorf("expected valid signature, got %v", err)
	}

	// Invalid signature.
	if err := VerifySignature(body, "sha256=invalid", secret); err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestParsePullRequestEvent(t *testing.T) {
	payload := []byte(`{
		"action": "opened",
		"number": 42,
		"pull_request": {
			"title": "Fix bug",
			"body": "This fixes the bug"
		},
		"repository": {
			"name": "forager",
			"owner": {"login": "lucientong"}
		}
	}`)

	event, err := ParsePullRequestEvent(payload)
	if err != nil {
		t.Fatalf("ParsePullRequestEvent: %v", err)
	}

	if event.Action != "opened" {
		t.Errorf("action: want opened, got %s", event.Action)
	}
	if event.Number != 42 {
		t.Errorf("number: want 42, got %d", event.Number)
	}
	if !event.IsReviewable() {
		t.Error("expected IsReviewable=true for 'opened' action")
	}

	ref := event.PRRef()
	if ref.Owner != "lucientong" || ref.Repo != "forager" || ref.Number != 42 {
		t.Errorf("PRRef: got %+v", ref)
	}
}
