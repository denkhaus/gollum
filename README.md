# Gollum

A powerful multi-agent system built with Go that leverages the Model Context Protocol (MCP) to create, manage, and orchestrate AI agents with advanced tooling capabilities.

## Overview

Gollum is a framework for building sophisticated AI agent systems. It provides:

- **Multi-Agent Architecture**: Spawn and manage hierarchical agent networks with parent-child relationships
- **MCP Integration**: Full support for Model Context Protocol servers and tools
- **Built-in Tools**: Comprehensive toolset including file operations, code search (grep/glob), bash execution, and more
- **Hook System**: Extensible hooks for logging, security, and custom behaviors
- **State Management**: File watching and state tracking with change detection
- **Dependency Injection**: Clean architecture using `samber/do` for service management
- **LLM Agnostic**: Works with multiple LLM providers via `m-mizutani/gollem`

## Features

### Agent Management
- Spawn new agents with custom system prompts
- Hierarchical agent relationships with automatic cleanup
- Agent limits and resource management
- Message passing between agents

### Available Tools
- **SpawnAgentTool**: Create new subagents with custom configurations
- **RemoveAgentTool**: Remove agents recursively
- **CurrentTimeTool**: Provide time information
- **BashTool**: Execute shell commands
- **EditTool**: Edit files with precise operations
- **GrepTool**: Search code with regex patterns
- **GlobTool**: Find files by pattern
- **ReadFileTool/WriteFileTool**: File I/O operations

### Hook System
- **LoggingHook**: Automatic request/response logging
- **SecurityHook**: Permission validation for tool operations
- **Custom Hooks**: Easy extension point for custom behaviors

### State Management
- File watching and change detection
- State locking for concurrent access
- Integration with file operations

## Installation

### Prerequisites
- Go 1.25.0 or later
- Access to an LLM provider (Anthropic Claude, OpenAI, etc.)

### Build from Source

```bash
# Clone the repository
git clone https://git.cluster.mirtuell.net/denkhaus/gollum.git
cd gollum

# Build the application
go build -o gollum cmd/gollum/main.go

# Or use go install
go install github.com/denkhaus/gollum/cmd/gollum@latest
```

## Quick Start

### Basic Usage

```bash
# Run gollum with default configuration
./gollum
```

### Configuration

Gollum uses environment variables for configuration (via `kelseyhightower/envconfig`):

```bash
# Set your LLM provider API key
export ANTHROPIC_API_KEY="your-api-key-here"

# Run with custom settings
./gollum
```

### Example: Creating an Agent

```go
import "github.com/denkhaus/gollum/pkg/tools"

// Spawn a new agent with a custom prompt
spawnTool := tools.NewSpawnAgentTool(registry, senderID)
result := spawnTool.Execute(ctx, request)
```

## Development

### Project Structure

```
gollum/
├── cmd/
│   └── gollum/           # CLI entry point
├── pkg/
│   ├── agents/           # Agent implementations and factories
│   ├── app/              # Application service
│   ├── builtin/          # Built-in hooks (Logging, Security)
│   ├── config/           # Configuration service
│   ├── di/               # Dependency injection container
│   ├── hooks/            # Hook system and manager
│   ├── llm/              # LLM client providers
│   ├── mcp/              # MCP server integration
│   ├── registry/         # Agent registry
│   ├── state/            # File state management
│   ├── tools/            # Tool implementations
│   ├── ui/               # User interface components
│   └── mocks/            # Centralized mock generation
├── docs/                 # Additional documentation
└── go.mod
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=/tmp/coverage.out ./...
go tool cover -html=/tmp/coverage.out

# Run tests for specific package
go test ./pkg/tools/...
```

### Linting

```bash
# Run golangci-lint
golangci-lint run ./...

# Run go vet
go vet ./...
```

### Mock Generation

Mocks are centrally managed in `pkg/mocks/generate.go`:

```bash
# Regenerate all mocks
go generate ./...
```

## Development Workflow

1. **Branch Strategy**: Create branches from `main` for features/fixes
2. **Testing**: Maintain >= 80% test coverage
3. **Code Quality**: All PRs must pass `go vet` and `golangci-lint`
4. **Commits**: Use meaningful commit messages with issue references

## Documentation

For detailed documentation on specific topics:

- **Agent Task System**: See [docs/agent-work.md](docs/agent-work.md) for complete guide on the Task tool
- **Project Guidelines**: See [CLAUDE.md](CLAUDE.md) for development practices
- **Dependencies**: See [go.mod](go.mod) for Go module requirements

## Architecture

Gollum follows clean architecture principles:

- **Dependency Injection**: All services managed through `samber/do` container
- **Interface-Based Design**: Easy mocking and testing with `go.uber.org/mock`
- **Hook Middleware**: Extensible behavior through pre/post hooks
- **State Management**: Thread-safe file watching and state tracking

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Ensure all quality checks pass
5. Submit a pull request

## License

See LICENSE file for details.

## Acknowledgments

Built with:
- [m-mizutani/gollem](https://github.com/m-mizutani/gollem) - LLM abstraction layer
- [samber/do](https://github.com/samber/do) - Dependency injection
- [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) - Anthropic API client
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) - MCP SDK
