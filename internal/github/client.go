// Package github provides a GitHub API client for Forager.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lucientong/forager/internal/config"
	"github.com/lucientong/forager/internal/models"
)

// Client is a GitHub API client.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient(cfg config.GitHubConfig) *Client {
	baseURL := cfg.APIURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{
		token:   cfg.Token,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// prResponse is the GitHub API response for a pull request.
type prResponse struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// commitResponse is the GitHub API response for a commit in a PR.
type commitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

// fileResponse is the GitHub API response for a changed file in a PR.
type fileResponse struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Patch    string `json:"patch"`
}

// GetPullRequest fetches PR metadata, diff, files, and commits.
func (c *Client) GetPullRequest(ctx context.Context, ref models.PRRef) (*models.PRData, error) {
	pr := &models.PRData{PRRef: ref}

	// Fetch PR metadata.
	var prResp prResponse
	if err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number), &prResp); err != nil {
		return nil, fmt.Errorf("get PR metadata: %w", err)
	}
	pr.Title = prResp.Title
	pr.Body = prResp.Body

	// Fetch diff.
	diff, err := c.getDiff(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get PR diff: %w", err)
	}
	pr.Diff = diff

	// Fetch changed files.
	files, err := c.getFiles(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get PR files: %w", err)
	}
	pr.Files = files

	// Fetch commits.
	commits, err := c.getCommits(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get PR commits: %w", err)
	}
	pr.Commits = commits

	return pr, nil
}

// PostReviewComment posts an issue comment on the PR.
func (c *Client) PostReviewComment(ctx context.Context, ref models.PRRef, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, ref.Owner, ref.Repo, ref.Number)

	payload := fmt.Sprintf(`{"body":%q}`, body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post comment: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// getDiff fetches the unified diff for a PR.
func (c *Client) getDiff(ctx context.Context, ref models.PRRef) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, ref.Owner, ref.Repo, ref.Number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getFiles fetches the changed files for a PR, handling pagination.
func (c *Client) getFiles(ctx context.Context, ref models.PRRef) ([]models.FileChange, error) {
	var all []models.FileChange
	page := 1
	for {
		var files []fileResponse
		path := fmt.Sprintf("/repos/%s/%s/pulls/%d/files?per_page=100&page=%d", ref.Owner, ref.Repo, ref.Number, page)
		if err := c.getJSON(ctx, path, &files); err != nil {
			return nil, err
		}
		if len(files) == 0 {
			break
		}
		for _, f := range files {
			all = append(all, models.FileChange{
				Filename: f.Filename,
				Language: InferLanguage(f.Filename),
				Patch:    f.Patch,
				Status:   f.Status,
			})
		}
		if len(files) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// getCommits fetches commit messages for a PR.
func (c *Client) getCommits(ctx context.Context, ref models.PRRef) ([]string, error) {
	var commits []commitResponse
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits?per_page=100", ref.Owner, ref.Repo, ref.Number)
	if err := c.getJSON(ctx, path, &commits); err != nil {
		return nil, err
	}
	msgs := make([]string, len(commits))
	for i, c := range commits {
		msgs[i] = c.Commit.Message
	}
	return msgs, nil
}

// getJSON makes a GET request and decodes JSON into v.
func (c *Client) getJSON(ctx context.Context, path string, v any) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api: status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// setHeaders adds common headers for GitHub API requests.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
}
