# Technology Stack

**Analysis Date:** 2025-01-31

## Languages

**Primary:**
- Go 1.25.0 - Core language for the agent system
  - Used in: All packages including `pkg/`, `cmd/gollum/`
  - Pattern: Structured, concurrent, test-driven development

**Secondary:**
- None detected - Pure Go codebase

## Runtime

**Environment:**
- Go Runtime - Standard Go runtime
- Executable: `cmd/gollum/main.go` - CLI entry point

**Package Manager:**
- Go Modules - Direct dependency management via `go.mod`
- Lockfile: `go.sum` - Present

## Frameworks

**Core:**
- Standalone Go application - No web framework
  - Entry point: `cmd/gollum/main.go`

**Dependency Injection:**
- samber/do/v2 v2.0.0 - Dependency injection framework
  - Location: `pkg/di/container.go`
  - Purpose: Service registration and management

**LLM Client Framework:**
- m-mizutani/gollem v0.17.3 - LLM client framework
  - Supports: Anthropic Claude, OpenAI, Google Gemini
  - Location: `pkg/llm/provider.go`

## Key Dependencies

**Critical:**
- github.com/m-mizutani/gollem v0.17.3 - Unified LLM client for Claude, OpenAI, Gemini
  - Why it matters: Core functionality for AI agent communication

**Infrastructure:**
- github.com/samber/do/v2 v2.0.0 - Dependency injection
- github.com/kelseyhightower/envconfig v1.4.0 - Environment configuration
  - Purpose: Configuration management via environment variables

**File System & Observability:**
- github.com/fsnotify/fsnotify v1.9.0 - File system watching
- go.uber.org/zap v1.27.1 - Structured logging
  - Purpose: File state tracking and application logging

**UI/Rendering:**
- github.com/charmbracelet/lipgloss v1.1.0 - Terminal UI styling
  - Purpose: Rich terminal output formatting

**Testing & Mocking:**
- github.com/stretchr/testify v1.11.1 - Test assertions
- go.uber.org/mock v0.6.0 - Code generation for mocks
  - Purpose: Test doubles and mocking framework

**Utilities:**
- github.com/google/uuid v1.6.0 - UUID generation
- golang.org/x/term v0.37.0 - Terminal utilities

## Configuration

**Environment:**
- Environment variables via envconfig
- Prefix: `GOLLUM_` (e.g., `GOLLUM_ANTHROPIC_API_KEY`)
- Config file: `pkg/config/service.go`

**Build:**
- Go build system via `go build`
- Linting: golangci-lint with `.golangci.yml`
- Formatters: gofmt, goimports

## Platform Requirements

**Development:**
- Go 1.25.0 or compatible
- golangci-lint for linting
- Standard Go toolchain

**Production:**
- Go runtime environment
- No specific OS requirements (cross-platform)
- Command-line interface execution

---

*Stack analysis: 2025-01-31*
```