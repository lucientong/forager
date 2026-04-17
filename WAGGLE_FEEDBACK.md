# Waggle Feedback from Forager

This file tracks waggle limitations, friction points, and feature requests
discovered during Forager development. Forager serves as both a product
and an integration test for waggle.

---

## All Items Resolved in v0.4.0

v0.4.0 解决了 Forager 开发中发现的所有 6 项 feedback。以下逐项说明。

### ~~[FRICTION] Chain Length Limit (Chain5 max)~~

**Resolved**: `agent.NewPipeline()` + `Add()` supports arbitrary-length pipelines.
We still use manual orchestration for Forager (cleaner for our data-flow pattern),
but the option now exists.

### ~~[FRICTION] Context Threading in Chains~~

**Resolved**: `agent.PipelineContext` — typed key-value bag that flows via `context.Context`.
Forager uses it to pass `PRRef` from FetchAgent to MergeAgent without polluting
intermediate types. The `PipelineGet[T]()` generic helper is especially clean.

### ~~[FRICTION] Parallel Output Type~~

**Resolved**: `waggle.ParallelThen[I, O, R](name, mergeFn, agents...)` composes
parallel + merge in one step. Forager uses it to replace the old
`Parallel` + manual `MergeInput` wrapper pattern. Much cleaner.

### ~~[WISH] Guardrail Type Constraints~~

**Resolved**: `guardrail.WithInputExtractGuard[I, O]()` and `WithOutputExtractGuard[I, O]()`
allow guardrails on arbitrary types via an extract function. Forager uses
`WithInputExtractGuard` on PostAgent to validate the formatted review comment
(length limit, PII filter, secrets filter) before posting to GitHub.

### ~~[WISH] Not Every Stage Needs an Agent~~

**Resolved**: v0.4.0 README now includes a dedicated "When to Use Agents vs Plain Functions"
section with clear guidance. Exactly what we suggested.

### ~~[WISH] StructuredAgent Inner Usage Pattern~~

**Resolved**: v0.4.0 README documents the "agents inside other agents" pattern with
a concrete example. Also clarifies that inner agents' metrics/traces are recorded
independently, and recommends PipelineContext for correlation.

---

## v0.4.0 New Feature Adoption Summary

| Feature | Where Used in Forager | Impact |
|---|---|---|
| `PipelineContext` | `pipeline.go` — passes PRRef | Eliminated `MergeInput` wrapper type |
| `ParallelThen` | `pipeline.go` — review+merge | Replaced 2-step Parallel+manual-merge with 1-step |
| `WithInputExtractGuard` | `pipeline.go` — PostAgent | Content safety (length, PII, secrets) before GitHub post |
| `Pipeline` builder | Available but not used | Manual orchestration still cleaner for our data-flow |
| "When to Use Agents" docs | Validated our SplitAgent decision | Confirms plain functions are preferred for trivial transforms |
| Inner agent docs | Validated our review agent pattern | Confirms agent-in-agent is supported and metrics work |
