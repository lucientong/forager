# Forager

[![Go Reference](https://pkg.go.dev/badge/github.com/lucientong/forager.svg)](https://pkg.go.dev/github.com/lucientong/forager)
[![CI](https://github.com/lucientong/forager/actions/workflows/ci.yml/badge.svg)](https://github.com/lucientong/forager/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucientong/forager)](https://goreportcard.com/report/github.com/lucientong/forager)
[![codecov](https://codecov.io/gh/lucientong/forager/branch/master/graph/badge.svg)](https://codecov.io/gh/lucientong/forager)
[![Docker Pulls](https://img.shields.io/docker/pulls/lucientong/forager)](https://hub.docker.com/r/lucientong/forager)
[![Go version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

[English](README.md) | 中文

> 基于 [waggle](https://github.com/lucientong/waggle) 构建的 AI 代码审查流水线。
> 命名来自 Forager Bee（侦察蜂）—— 为蜂巢探索资源的先锋。

Forager 接收 GitHub PR Webhook，将多个专家审查 Agent 并行分发，汇总审查结果，并将结构化的审查评论发回 PR。

## 架构

```
GitHub Webhook (PR opened/synchronized)
    |
    v
FetchAgent [事件 -> PR数据]
    | 调用 GitHub API：获取 diff、文件列表、commit 信息
    |
    v  (并行 — 4 个 Agent 同时运行)
+------------------------------------------------------+
| SecurityAgent | StyleAgent  | LogicAgent  | PerfAgent |
| 安全审查       | 风格审查     | 逻辑审查     | 性能审查   |
+------------------------------------------------------+
    |
    v  (ParallelThen: 并行 + 合并一步完成)
MergeAgent [审查结果 -> 聚合报告]
    | 去重、排序、评分
    |
    v
SummaryAgent [聚合报告 -> 聚合报告]
    | 通过 LLM 生成人类可读的 Markdown 总结
    |
    v
PostAgent [聚合报告 -> 发布结果]  ← Guardrail 保护
    | 通过 GitHub API 发布审查评论
    v
完成
```

## 快速开始

### 前置条件

- Go 1.26+
- 有仓库权限的 GitHub Token
- LLM API Key（Anthropic / OpenAI / 或本地 Ollama）

### 构建与运行

```bash
# 构建
go build -o forager ./cmd/forager

# 配置（复制并编辑）
cp configs/config.yaml my-config.yaml

# 使用环境变量运行（单 provider 简单模式）
export FORAGER_GITHUB_TOKEN="ghp_..."
export FORAGER_ANTHROPIC_API_KEY="sk-ant-..."
export FORAGER_WEBHOOK_SECRET="your-webhook-secret"
./forager --config my-config.yaml

# 或使用 Ollama（无需 API Key）
# 在 config.yaml 中配置 providers.ollama 并设置 agents.default: "ollama"
export FORAGER_GITHUB_TOKEN="ghp_..."
./forager --config my-config.yaml
```

### Docker

```bash
# 使用官方镜像
docker pull lucientong/forager:latest
docker run -p 8080:8080 \
  -e FORAGER_GITHUB_TOKEN="ghp_..." \
  -e FORAGER_ANTHROPIC_API_KEY="sk-ant-..." \
  -e FORAGER_WEBHOOK_SECRET="..." \
  lucientong/forager:latest

# 或自行构建
docker build -t forager .
docker run -p 8080:8080 -p 8081:8081 \
  -e FORAGER_GITHUB_TOKEN="ghp_..." \
  -e FORAGER_ANTHROPIC_API_KEY="sk-ant-..." \
  forager
```

## 配置

配置文件使用 YAML 格式，支持环境变量覆盖。

### 多 Provider 配置

每个审查 Agent 可以使用不同的 LLM Provider。例如安全审查用 Claude（更谨慎），风格审查用 GPT-4o（更快/更便宜）：

```yaml
providers:
  anthropic:
    api_key: ""                        # 或设置 FORAGER_ANTHROPIC_API_KEY
    model: "claude-3-5-sonnet-20241022"
  openai:
    api_key: ""                        # 或设置 FORAGER_OPENAI_API_KEY
    model: "gpt-4o"
  ollama:
    model: "llama3.2"
    base_url: "http://localhost:11434"

agents:
  default: "anthropic"                 # 默认 provider
  security: "anthropic"                # 安全审查 → Claude（更谨慎）
  style: "openai"                      # 风格审查 → GPT-4o（更快更便宜）
  logic: "anthropic"                   # 逻辑审查 → Claude
  performance: "openai"                # 性能审查 → GPT-4o
  summary: "openai"                    # 总结生成 → GPT-4o
  fallback_order: ["anthropic", "openai", "ollama"]  # 自动降级
```

三种使用方式：
1. **简单模式** — 只配一个 provider + `default`，开箱即用
2. **分 Agent 优化** — 安全用贵的 Claude，风格用便宜的 GPT-4o，省成本
3. **高可用** — 配多个 provider + `fallback_order`，任何 provider 故障自动降级

### 环境变量

| 环境变量 | 说明 |
|---|---|
| `FORAGER_GITHUB_TOKEN` | GitHub 个人访问令牌 |
| `FORAGER_WEBHOOK_SECRET` | Webhook HMAC 签名密钥 |
| `FORAGER_ANTHROPIC_API_KEY` | Anthropic API Key |
| `FORAGER_OPENAI_API_KEY` | OpenAI API Key |
| `FORAGER_OLLAMA_URL` | Ollama 服务器地址 |
| `FORAGER_LLM_API_KEY` | 兼容旧版：设置默认 provider 的 API Key |
| `FORAGER_PORT` | HTTP 端口（默认 8080） |
| `FORAGER_LOG_LEVEL` | 日志级别：`debug`、`info`、`warn`、`error` |

完整配置参考 `configs/config.yaml`。

## API 端点

| 方法 | 路径 | 端口 | 说明 |
|---|---|---|---|
| `POST` | `/webhook` | 8080 | GitHub Webhook 接收端点 |
| `GET` | `/healthz` | 8080 | 健康检查 |
| `GET` | `/metrics` | 8080 | Prometheus 指标 |
| `GET` | `/` | 8081* | Waggle 可视化面板（需配置 `web_port`） |
| `GET` | `/api/events` | 8081* | SSE 实时 Agent 状态流 |
| `GET` | `/api/metrics` | 8081* | Agent 指标 JSON |

*可视化面板端口通过 `server.web_port` 配置，设为 0 禁用。

## 使用的 Waggle 特性

| 特性 | 用途 |
|---|---|
| `agent.Func` | 从函数创建轻量级 Agent |
| `agent.WithRetry` | 指数退避重试 |
| `agent.WithTimeout` | 单次调用超时控制 |
| `agent.PipelineContext` | 跨 Stage 类型安全的上下文传递 (v0.4.0) |
| `waggle.ParallelThen` | 并行执行 + 结果合并一步完成 (v0.4.0) |
| `guardrail.WithInputExtractGuard` | 任意类型的内容安全检查 (v0.4.0) |
| `output.NewStructuredAgent` | LLM 输出的类型安全 JSON 解析 |
| `llm.NewLLMAgent` | 自由文本生成（总结） |
| `llm.NewAnthropic/OpenAI/Ollama` | 多 Provider 支持 |
| `llm.NewRouter` | Provider 间 Failover 路由 |
| `prompt.Template` | 不可变的 Prompt 模板 |
| `memory.NewWindowStore` | SummaryAgent 保留 PR 审查历史上下文 |
| `stream.Observer` | 实时 Agent 执行进度通知 |
| `web.NewServer` | 内嵌可视化面板 + SSE 实时事件流 |
| `observe.Metrics` + `PrometheusHandler` | 每个 Agent 的可观测性指标 |

## 开发

```bash
# 运行测试
go test ./... -v

# 静态检查
go vet ./...

# 构建
go build -o forager ./cmd/forager
```

## 项目结构

```
forager/
├── cmd/forager/main.go              # 入口：配置加载、依赖注入、优雅关闭
├── internal/
│   ├── models/models.go             # 核心类型：PRRef, PRData, FileChange, Review
│   ├── config/                      # YAML 配置 + 环境变量覆盖 + 校验
│   ├── github/                      # GitHub API 客户端 + Webhook 解析
│   ├── llm/provider.go              # LLM Provider 注册中心（多 Provider + Failover）
│   ├── prompts/templates.go         # Waggle Prompt 模板（安全/风格/逻辑/性能/总结）
│   ├── agents/                      # 所有 Waggle Agent
│   │   ├── fetch.go                 # FetchAgent: Webhook → PRData
│   │   ├── review.go                # 审查 Agent 工厂（每个文件调用 StructuredAgent）
│   │   ├── reviewers.go             # 安全/风格/逻辑/性能 Agent 构造器
│   │   ├── merge.go                 # MergeAgent: 去重、排序、评分、格式化
│   │   ├── summary.go               # SummaryAgent: LLM 生成 Markdown 总结
│   │   └── post.go                  # PostAgent: 发布到 GitHub
│   ├── pipeline/pipeline.go         # 全流程编排（ParallelThen + PipelineContext + Guardrail）
│   └── server/                      # HTTP 服务器 + Webhook 处理器
├── configs/config.yaml              # 配置示例
├── .github/workflows/               # CI/CD 配置
├── Dockerfile                       # 多阶段构建
├── WAGGLE_FEEDBACK.md               # Waggle 集成反馈
└── README.md / README_zh.md         # 文档
```

## Waggle 反馈

参见 [WAGGLE_FEEDBACK.md](WAGGLE_FEEDBACK.md)，记录了开发过程中发现的 waggle 问题及 v0.4.0 的修复情况。
Forager 同时作为 waggle 的集成测试项目。

## 开源协议

Apache License 2.0
