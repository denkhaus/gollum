# Proposal: Resolve Import Cycle in Session/Repository

## Current State

### The Cycle
```
pkg/session → pkg/session/repository → pkg/channel → pkg/session
              ↑                                            ↓
              └────────────────────────────────────────────┘
```

### Dependencies
1. `pkg/session/repository/repository.go` imports:
   - `github.com/denkhaus/gollum/pkg/channel` (for `channel.Message`)

2. `pkg/channel` imports:
   - `github.com/denkhaus/gollum/pkg/session` (from various files)

3. `pkg/session/manager.go` imports:
   - `github.com/denkhaus/gollum/pkg/session/repository` (for `repository.SessionRepository`)

### Root Cause
The `SessionRepository` interface is defined in a subpackage (`pkg/session/repository`) but depends on types from `pkg/channel`, while the parent `pkg/session` package needs to use this interface. When `pkg/channel` also imports `pkg/session`, a cycle is formed.

## Proposed Solution

### Option 1: Move Message to Shared Package (RECOMMENDED)

Move `channel.Message` to `pkg/shared` as it represents a core data structure used across multiple packages.

**Rationale:**
- Message already embeds `shared.SessionContext`
- Messages are consumed by multiple subsystems (TUI, logging, agents, tools)
- The message concept is not unique to channels - it's a fundamental communication primitive

**Changes:**
1. Move `Message` and `MessageType` from `pkg/channel/interface.go` to `pkg/shared/message.go`
2. Update all imports across the codebase:
   - `channel.Message` → `shared.Message`
   - `channel.MessageType*` → `shared.MessageType*`
3. The repository interface can then use `shared.Message` instead of `channel.Message`
4. No cycle: `pkg/session/repository` → `pkg/shared` (no cycle)

**Files to modify:**
- Create: `pkg/shared/message.go`
- Modify: All files using `channel.Message` (~200+ references)
- Modify: `pkg/session/repository/repository.go`

### Option 2: Flatten Repository Package Structure

Move the `SessionRepository` interface from `pkg/session/repository` to `pkg/session` (parent package).

**Rationale:**
- Removes the subpackage dependency entirely
- The interface is conceptually part of the session management API
- Implementation can stay in `pkg/session/persistence` or similar

**Changes:**
1. Move `SessionRepository` interface to `pkg/session/repository.go` (new file)
2. Keep implementation in `pkg/session/persistence/entrepo.go`
3. Update imports in `pkg/session/manager.go`

**Files to modify:**
- Create: `pkg/session/repository.go` (interface only)
- Move: `pkg/session/repository/repository.go` → `pkg/session/persistence/interface.go`
- Modify: `pkg/session/manager.go`

**Risk:** Still has `channel.Message` dependency in the interface, which may cause issues.

### Option 3: Dependency Inversion

Create an abstraction layer that both packages can depend on.

**Rationale:**
- Follows SOLID principles (Dependency Inversion)
- Decouples packages completely

**Changes:**
1. Create `pkg/domain/message.go` with `Message` interface/struct
2. Both `pkg/channel` and `pkg/session/repository` depend on `pkg/domain`
3. Adapter pattern for implementation

**Risk:** More complex, may be over-engineering for this use case.

## Recommendation

**Go with Option 1** for these reasons:

1. **Semantic correctness:** Messages are a cross-cutting concern, not channel-specific
2. **Already partially there:** Message embeds `shared.SessionContext`
3. **Simpler architecture:** Removes the subpackage complexity
4. **Aligns with existing patterns:** Shared types like `SessionContext` already exist in `pkg/shared`

## Migration Steps

1. Create `pkg/shared/message.go` with Message types
2. Update `pkg/session/repository/repository.go` to use `shared.Message`
3. Run tests to verify no cycle
4. Gradually migrate other packages from `channel.Message` to `shared.Message`
5. Remove `Message` from `pkg/channel/interface.go`
6. Run full test suite

## Impact Assessment

- **High risk:** Message is used extensively (~200+ references)
- **Medium effort:** Requires systematic find-replace across codebase
- **High value:** Resolves architectural debt and prevents future cycles
- **Low runtime impact:** Type move, no behavior change
