# Codebase Structure

**Analysis Date:** 2026-01-31

## Directory Layout

```
/home/denkhaus/dev/gomodules/gollum/
├── cmd/                     # Command-line applications
│   └── gollum/            # Main CLI entry point
│       └── main.go        # Application startup and shutdown
├── pkg/                    # Application packages
│   ├── app/               # Application orchestration
│   ├── agents/            # Agent implementations and factories
│   ├── builtin/           # Built-in hooks
│   ├── config/            # Configuration management
│   ├── di/                # Dependency injection container
│   ├── errs/              # Custom error types
│   ├── hooks/             # Hook system implementation
│   ├── llm/               # LLM provider clients
│   ├── logger/            # Logging service
│   ├── mcp/               # MCP protocol integration
│   ├── middleware/        # UI and display middleware
│   ├── mocks/             # Centralized mocks (generated)
│   ├── prompt/            # Prompt management
│   ├── prompt/templates/  # Predefined prompt templates
│   ├── registry/          # Agent registry management
│   ├── shared/            # Shared types and interfaces
│   ├── state/             # State management
│   ├── tools/             # Tool implementations
│   └── ui/                # User interface components
├── docs/                  # Documentation
└── .planning/codebase/    # Generated analysis documents
```

## Directory Purposes

**cmd/gollum/:
- Purpose: Application entry point and startup logic
- Contains: CLI setup, signal handling, service initialization
- Key files: `main.go` (application startup with DI container)

**pkg/app/:
- Purpose: Application orchestration and main loop
- Contains: Application service, interactive CLI handling
- Key files: `service.go` (ApplicationService implementation)

**pkg/agents/:
- Purpose: Agent creation and management
- Contains: Default agent configuration, factory implementations
- Key files: `default.go`, `factory.go`

**pkg/builtin/:
- Purpose: Pre-implemented hooks for common functionality
- Contains: Security logging, utility hooks
- Key files: `security_hook.go`, `logging_hook.go`

**pkg/config/:
- Purpose: Configuration management and validation
- Contains: Agent limits, service configuration
- Key files: `service.go`, `service_test.go`

**pkg/di/:
- Purpose: Dependency injection container management
- Contains: Service registration and lifecycle management
- Key files: `container.go` (service registration order)

**pkg/errs/:
- Purpose: Custom error types with context
- Contains: Error definitions and factory methods
- Key files: `errors.go`, `errors_test.go`

**pkg/hooks/:
- Purpose: Hook system for extensible cross-cutting concerns
- Contains: Hook lifecycle management, point implementations
- Key files: `manager.go`, `lifecycle.go`, file/tool/LLM hooks

**pkg/llm/:
- Purpose: LLM provider integration
- Contains: Client provider interface implementations
- Key files: `provider.go`

**pkg/logger/:
- Purpose: Logging service with buffer support
- Contains: Structured logging with zap integration
- Key files: `buffer.go`, `buffer_test.go`

**pkg/mcp/:
- Purpose: Model Context Protocol integration
- Contains: Brain MCP client implementation
- Key files: `brain.go`

**pkg/middleware/:
- Purpose: UI and display handling
- Contains: Middleware for agent output formatting
- Key files: `summary.go`

**pkg/mocks/:
- Purpose: Centralized mock implementations
- Contains: Generated mocks for all interfaces
- Key files: `generate.go`, `mock_registry.go`, etc.

**pkg/prompt/:
- Purpose: Prompt template management
- Contains: Prompt loading and management
- Key files: `templates/` contains system prompts

**pkg/registry/:
- Purpose: Agent lifecycle and relationship management
- Contains: Thread-safe agent registry with parent-child support
- Key files: `registry.go`, multiple test files

**pkg/shared/:
- Purpose: Common types and interfaces
- Contains: Agent interfaces, configuration types, constants
- Key files: `agent.go`, `shared.go`, `factory.go`

**pkg/state/:
- Purpose: File system state tracking
- Contains: File state manager, watcher, locking
- Key files: `file_state_manager.go`, `types.go`

**pkg/tools/:
- Purpose: Tool implementations for agent capabilities
- Contains: File operations, agents management, bash, grep, glob
- Key files: `tools.go` lists all tool providers

**pkg/ui/:
- Purpose: User interface components
- Contains: Agent messenger interface
- Key files: Interface definitions

## Key File Locations

**Entry Points:**
- `/home/denkhaus/dev/gomodules/gollum/cmd/gollum/main.go`: Application startup
- `/home/denkhaus/dev/gomodules/gollum/pkg/app/service.go`: Main application service

**Configuration:**
- `/home/denkhaus/dev/gomodules/gollum/pkg/config/service.go`: Configuration management
- `/home/denkhaus/dev/gomodules/gollum/pkg/di/container.go`: DI service registration

**Core Logic:**
- `/home/denkhaus/dev/gomodules/gollum/pkg/registry/registry.go`: Agent registry
- `/home/denkhaus/dev/gomodules/gollum/pkg/state/file_state_manager.go`: State management
- `/home/denkhaus/dev/gomodules/gollum/pkg/shared/shared.go`: Common types

**Testing:**
- `/home/denkhaus/dev/gomodules/gollum/pkg/mocks/generate.go`: Mock generation
- Test files co-located with implementation files

## Naming Conventions

**Files:**
- Interfaces: `[Name]Service`, `[Name]Provider`, `[Name]Manager`
- Implementations: `service.go`, `manager.go`, `provider.go`
- Factories: `factory.go`
- Test files: `_test.go`, `_spec.go`

**Directories:**
- Lowercase, underscores for multi-word
- Plural for service directories (tools, agents)
- Singular for functional areas (registry, state, config)

**Go Package Names:**
- Lowercase, same as directory name
- No underscores

**Variable Names:**
- Interface receivers: `receiver` (e.g., `r *registry`)
- Function parameters: descriptive names
- Private members: lowercase
- Public members: PascalCase

## Where to Add New Code

**New Tool:**
- Implementation: `/home/denkhaus/dev/gomodules/gollum/pkg/tools/[tool_name].go`
- Provider: Add to tools.go
- Test: `[tool_name]_test.go`
- Mock: Update `/home/denkhaus/dev/gomodules/gollum/pkg/mocks/generate.go`

**New Agent Type:**
- Interface: Update `/home/denkhaus/dev/gomodules/gollum/pkg/shared/agent.go`
- Implementation: Add to `/home/denkhaus/dev/gomodules/gollum/pkg/agents/`
- Factory: Update factory.go
- Registry: Add to default agent config

**New Hook Point:**
- Definition: Add to `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/types.go`
- Manager: Add hook methods to `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/manager.go`
- Hook implementation: Add to builtin/ or custom package

**New Configuration:**
- Types: Add to `/home/denkhaus/dev/gomodules/gollum/pkg/config/service.go`
- Default values: Update config loading
- Validation: Add validation logic

**New State Management:**
- Types: Add to `/home/denkhaus/dev/gomodules/gollum/pkg/state/types.go`
- Implementation: Add to `/home/denkhaus/dev/gomodules/gollum/pkg/state/`
- Registry: Register in DI container

## Special Directories

**pkg/mocks/:**
- Purpose: Centralized mock generation
- Generated: Yes (via go generate)
- Committed: Yes (generated files)
- Pattern: `mock_[InterfaceName].go`

**pkg/prompt/templates/:**
- Purpose: Predefined prompt templates
- Generated: No
- Committed: Yes
- Contains: System prompts for different agent types

**.planning/codebase/:**
- Purpose: Generated analysis documents
- Generated: Yes (by this analysis)
- Committed: Yes
- Contains: ARCHITECTURE.md, STRUCTURE.md

---

*Structure analysis: 2026-01-31*