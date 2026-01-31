# External Integrations

**Analysis Date:** 2025-01-31

## APIs & External Services

**LLM Provider APIs:**
- **Anthropic Claude API**
  - SDK: github.com/m-mizutani/gollem/llm/claude
  - Auth: `ANTHROPIC_API_KEY` environment variable
  - Configuration: `pkg/config/service.go` - `AnthropicConfig`
  - Base URL override: `ANTHROPIC_BASE_URL`

- **OpenAI GPT API**
  - SDK: github.com/m-mizutani/gollem/llm/openai
  - Auth: `OPENAI_API_KEY` environment variable
  - Configuration: `pkg/config/service.go` - `OpenAIConfig`
  - Base URL override: `OPENAI_BASE_URL`

- **Google Gemini API**
  - SDK: github.com/m-mizutani/gollem/llm/gemini
  - Auth: Service account via `GEMINI_PROJECT_ID`, `GEMINI_LOCATION`
  - Configuration: `pkg/config/service.go` - `GeminiConfig`

## Data Storage

**Databases:**
- None detected - State management is in-memory and file-based

**File Storage:**
- Local filesystem only
  - Managed via: `pkg/state/file_state_manager.go`
  - Watching: `github.com/fsnotify/fsnotify v1.9.0`

**Caching:**
- In-memory caching for file state
  - Location: `pkg/state/file_state_manager.go`
  - No external cache service

## Authentication & Identity

**Auth Provider:**
- Custom token-based authentication
  - Implementation: API keys for each LLM provider
  - No centralized auth system

## Monitoring & Observability

**Error Tracking:**
- Not detected - Standard error handling via Go errors

**Logs:**
- Zap logger (go.uber.org/zap v1.27.1)
  - Location: `pkg/logger/`
  - Configuration: `pkg/config/service.go` - `LoggingConfig`
  - Session log buffering: In-memory logs with configurable size

**Metrics:**
- Optional Prometheus-compatible metrics
  - Enabled via: `HOOKS_METRICS_ENABLED` environment variable
  - Implementation: `pkg/hooks/` with MetricsHook

## CI/CD & Deployment

**Hosting:**
- Not detected - CLI tool, not web service

**CI Pipeline:**
- GitHub Actions: `.github/workflows/claude-code.yaml`
- Forgejo Actions: `.forgejo/workflows/claude-code-forgejo.yaml`
- Codecov: `.github/workflows/codecov.yml`

## Environment Configuration

**Required env vars:**
- `GOLLUM_ANTHROPIC_API_KEY` - Anthropic API key
- `GOLLUM_OPENAI_API_KEY` - OpenAI API key
- `GOLLUM_GEMINI_PROJECT_ID` - Gemini project ID
- `GOLLUM_GEMINI_LOCATION` - Gemini location

**Optional env vars:**
- `GOLLUM_ANTHROPIC_BASE_URL` - Anthropic API base URL
- `GOLLUM_OPENAI_BASE_URL` - OpenAI API base URL
- `GOLLUM_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `GOLLUM_DEVELOPMENT` - Development mode flag

**Secrets location:**
- Environment variables
- No secrets committed to repository

## Webhooks & Callbacks

**Incoming:**
- MCP (Model Context Protocol) endpoint
  - URL: `http://localhost:8555/mcp`
  - Implementation: `pkg/mcp/brain.go`
  - Purpose: External tool integration

**Outgoing:**
- No outgoing webhooks detected

---

*Integration audit: 2025-01-31*
```