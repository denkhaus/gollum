# Architecture

**Analysis Date:** 2026-01-31

## Pattern Overview

**Overall:** Clean Architecture with Hexagonal design pattern

**Key Characteristics:**
- Strict dependency inversion using dependency injection
- Separation of concerns through layered architecture
- Inter-Agent communication with parent-child relationships
- Hook system for cross-cutting concerns
- Event-driven execution with cancellation support

## Layers

**Application Layer (`pkg/app`)**:
- Purpose: Application orchestration and CLI interaction
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/app/`
- Contains: ApplicationService, interactive loop, terminal management
- Depends on: Domain layer through interfaces
- Used by: Main CLI entry point

**Domain Layer (`pkg/shared`, `pkg/registry`, `pkg/state`)**:
- Purpose: Core business logic and domain entities
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/shared/`, `/home/denkhaus/dev/gomodules/gollum/pkg/registry/`, `/home/denkhaus/dev/gomodules/gollum/pkg/state/`
- Contains: Agent interfaces, registry management, state management
- Depends on: Nothing (pure Go interfaces and types)
- Used by: Application, Infrastructure, and Tools layers

**Infrastructure Layer (`pkg/tools`, `pkg/mcp`, `pkg/llm`)**:
- Purpose: External integrations and tool implementations
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/tools/`, `/home/denkhaus/dev/gomodules/gollum/pkg/mcp/`, `/home/denkhaus/dev/gomodules/gollum/pkg/llm/`
- Contains: Tool implementations, LLM clients, MCP integrations
- Depends on: Domain layer through interfaces
- Used by: Domain layer and Application layer

**Cross-Cutting Layer (`pkg/di`, `pkg/config`, `pkg/logger`, `pkg/hooks`)**:
- Purpose: Dependency injection, configuration, logging, and hooks
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/di/`, `/home/denkhaus/dev/gomodules/gollum/pkg/config/`, `/home/denkhaus/dev/gomodules/gollum/pkg/logger/`, `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/`
- Contains: DI container, configuration management, logging service, hook system
- Depends on: Application layer
- Used by: All layers

## Data Flow

**Agent Execution Flow:**

1. **User Input** → Application layer reads from stdin in raw mode
2. **Create Agent** → AgentFactory creates agent with specific configuration
3. **Register Agent** → AgentRegistry tracks all active agents with parent-child relationships
4. **Execute Agent** → Agent executes using gollem framework with configured tools
5. **Handle Output** → Agent execution results handled through middleware
6. **Background Agents** → Async execution with result tracking in registry

**State Management Flow:**

1. **Prime FSM** → FileStateManager scans working directory at startup
2. **Track Changes** → Watcher monitors file system changes
3. **Manage Locks** → Locking mechanism prevents concurrent modifications
4. **Persist State** → State snapshots stored during execution

**Hook Execution Flow:**

1. **Before Hook Point** → Pre-execution hooks run (e.g., security checks)
2. **Execute Core Logic** → Main functionality runs
3. **After Hook Point** → Post-execution hooks run (e.g., logging, cleanup)

## Key Abstractions

**Agent Interface:**
- Purpose: Abstract agent behavior with minimal interface
- Examples: `/home/denkhaus/dev/gomodules/gollum/pkg/shared/agent.go`
- Pattern: Interface with ID, Config, Session, and Execute methods

**AgentRegistry:**
- Purpose: Centralized management of agent lifecycle
- Examples: `/home/denkhaus/dev/gomodules/gollum/pkg/registry/registry.go`
- Pattern: Thread-safe registry with parent-child relationships

**FileStateManager:**
- Purpose: Track file system state during execution
- Examples: `/home/denkhaus/dev/gomodules/gollum/pkg/state/file_state_manager.go`
- Pattern: State machine with snapshots and change tracking

**HookManager:**
- Purpose: Extensible hook system for cross-cutting concerns
- Examples: `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/manager.go`
- Pattern: Observer pattern with pre/post hooks

**Dependency Injection Container:**
- Purpose: Manage service lifecycle and dependencies
- Examples: `/home/denkhaus/dev/gomodules/gollum/pkg/di/container.go`
- Pattern: Service locator with dependency graph

## Entry Points

**Main CLI Entry Point:**
- Location: `/home/denkhaus/dev/gomodules/gollum/cmd/gollum/main.go`
- Triggers: Application startup and service initialization
- Responsibilities: CLI setup, signal handling, graceful shutdown

**Application Service:**
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/app/service.go`
- Triggers: Main application loop and agent execution
- Responsibilities: Agent creation, interactive loop, terminal management

**Agent Factory:**
- Location: `/home/denkhaus/dev/gomodules/gollum/pkg/shared/factory.go`
- Triggers: New agent creation
- Responsibilities: Agent instantiation with configuration

## Error Handling

**Strategy:** Custom error types with context

**Patterns:**
- Error types in `/home/denkhaus/dev/gomodules/gollum/pkg/errs/errors.go`
- Context-aware errors with WithContext method
- Error wrapping for proper stack traces
- Agent-specific error handling (timeout, permission, etc.)

**Error Categories:**
- Configuration errors
- Agent lifecycle errors
- Tool execution errors
- State management errors
- Permission errors

## Cross-Cutting Concerns

**Logging:** Structured logging with zap logger
**Validation:** Interface validation and permission checks
**Authentication:** Parent-child relationship permissions
**Caching:** Agent result caching for background execution
**Concurrency:** Mutex-based locking for shared resources

---

*Architecture analysis: 2026-01-31*