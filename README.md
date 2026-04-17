# Forager

[![Go Reference](https://pkg.go.dev/badge/github.com/lucientong/forager.svg)](https://pkg.go.dev/github.com/lucientong/forager)
[![CI](https://github.com/lucientong/forager/actions/workflows/ci.yml/badge.svg)](https://github.com/lucientong/forager/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucientong/forager)](https://goreportcard.com/report/github.com/lucientong/forager)
[![codecov](https://codecov.io/gh/lucientong/forager/branch/master/graph/badge.svg)](https://codecov.io/gh/lucientong/forager)
[![Docker Pulls](https://img.shields.io/docker/pulls/lucientong/forager)](https://hub.docker.com/r/lucientong/forager)
[![Go version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

> AI-powered code review pipeline built on [waggle](https://github.com/lucientong/waggle).
> Named after the forager bee — the scout that ventures out to find resources for the hive.

Forager receives GitHub PR webhooks, dispatches multiple specialist review agents in parallel, aggregates their findings, and posts a structured review comment back to the PR.

## Architecture

```
GitHub Webhook (PR opened/synchronized)
    |
    v
FetchAgent [event -> PRData]
    | GitHub API: fetch diff, file list, commit messages
    |
    v  (Parallel — 4 agents concurrently)
+---------------------------------------------------+
| SecurityAgent | StyleAgent | LogicAgent | PerfAgent |
| [[]FileChange | [[]FileChange | [[]FileChange | [[]FileChange |
|  -> []Review] |  -> []Review] | -> []Review] | -> []Review] |
+---------------------------------------------------+
    |
    v
MergeAgent [[][]Review -> AggregatedReview]
    | Flatten, deduplicate, sort, score
    |
    v
SummaryAgent [AggregatedReview -> AggregatedReview]
    | Generate human-readable markdown summary via LLM
    |
    v
PostAgent [AggregatedReview -> bool]
    | Post review comment via GitHub API
    v
Done
```

## Quick Start

### Prerequisites

- Go 1.26+
- A GitHub token with repo access
- An LLM API key (Anthropic, OpenAI, or local Ollama)

### Build & Run

```bash
# Build
go build -o forager ./cmd/forager

# Configure (copy and edit)
cp configs/config.yaml my-config.yaml
# Edit my-config.yaml with your settings, or use env vars:

# Run with single provider (simple setup)
export FORAGER_GITHUB_TOKEN="ghp_..."
export FORAGER_ANTHROPIC_API_KEY="sk-ant-..."
export FORAGER_WEBHOOK_SECRET="your-webhook-secret"
./forager --config my-config.yaml

# Or with Ollama (no API key needed)
# Edit config.yaml to set providers.ollama and agents.default: "ollama"
export FORAGER_GITHUB_TOKEN="ghp_..."
./forager --config my-config.yaml
```

### Docker

```bash
# Use the official image
docker pull lucientong/forager:latest
docker run -p 8080:8080 \
  -e FORAGER_GITHUB_TOKEN="ghp_..." \
  -e FORAGER_ANTHROPIC_API_KEY="sk-ant-..." \
  -e FORAGER_WEBHOOK_SECRET="..." \
  lucientong/forager:latest

# Or build locally
docker build -t forager .
```

## Configuration

Configuration is loaded from a YAML file with environment variable overrides.

### Multi-Provider Setup

Each review agent can use a different LLM provider. For example, security reviews
use Claude (more cautious) while style reviews use GPT-4o (faster/cheaper):

```yaml
providers:
  anthropic:
    api_key: ""    # or set FORAGER_ANTHROPIC_API_KEY
    model: "claude-3-5-sonnet-20241022"
  openai:
    api_key: ""    # or set FORAGER_OPENAI_API_KEY
    model: "gpt-4o"

agents:
  default: "anthropic"
  security: "anthropic"    # Claude for security (more cautious)
  style: "openai"          # GPT-4o for style (faster/cheaper)
  logic: "anthropic"
  performance: "openai"
  summary: "openai"
  fallback_order: ["anthropic", "openai"]  # auto-failover
```

### Environment Variables

| Environment Variable | Description |
|---|---|
| `FORAGER_GITHUB_TOKEN` | GitHub personal access token |
| `FORAGER_WEBHOOK_SECRET` | Webhook HMAC secret |
| `FORAGER_ANTHROPIC_API_KEY` | Anthropic API key |
| `FORAGER_OPENAI_API_KEY` | OpenAI API key |
| `FORAGER_OLLAMA_URL` | Ollama server URL |
| `FORAGER_LLM_API_KEY` | Legacy: sets API key on the default provider |
| `FORAGER_PORT` | HTTP port (default: 8080) |
| `FORAGER_LOG_LEVEL` | `debug`, `info`, `warn`, `error` |

See `configs/config.yaml` for the full configuration reference.

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/webhook` | GitHub webhook receiver |
| `GET` | `/healthz` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

## Waggle Features Used

- **`agent.Func`** — Lightweight agents from functions
- **`agent.WithRetry`** — Exponential backoff with jitter
- **`agent.WithTimeout`** — Per-call deadline enforcement
- **`agent.PipelineContext`** — Typed context threading across pipeline stages (v0.4.0)
- **`waggle.ParallelThen`** — Concurrent fan-out + merge in one step (v0.4.0)
- **`guardrail.WithInputExtractGuard`** — Content safety on arbitrary types (v0.4.0)
- **`output.NewStructuredAgent`** — Type-safe JSON parsing from LLM output
- **`llm.NewLLMAgent`** — Free-form text generation (summaries)
- **`llm.NewAnthropic/OpenAI/Ollama`** — Multi-provider LLM support
- **`llm.NewRouter`** — Failover routing across providers
- **`prompt.Template`** — Immutable prompt templates with variable substitution
- **`observe.Metrics`** + **`PrometheusHandler`** — Per-agent observability
- **`memory.NewWindowStore`** — Per-PR conversation history for SummaryAgent
- **`stream.Observer`** — Real-time agent execution progress
- **`web.NewServer`** — Embedded visualization panel with SSE events

## Development

```bash
# Run tests
go test ./... -v

# Vet
go vet ./...

# Build
go build -o forager ./cmd/forager
```

## Waggle Feedback

See [WAGGLE_FEEDBACK.md](WAGGLE_FEEDBACK.md) for limitations and feature requests
discovered during development. Forager serves as an integration test for waggle.

## License

Apache License 2.0
