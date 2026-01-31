---
phase: 01-core-types-and-store-layer
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/prompt/types.go
  - pkg/prompt/store/store.go
  - pkg/prompt/store/memory_store.go
  - pkg/prompt/store/file_store.go
  - pkg/prompt/store/provider.go
  - pkg/prompt/store/store_test.go
  - pkg/mocks/generate.go
autonomous: true
must_haves:
  truths:
    - Prompt struct with SemVer versioning can be created and stored
    - InMemory store implementation persists and retrieves prompts correctly in tests
    - File store implementation persists prompts as JSON with file locking for concurrent safety
    - Alias resolution converts shortcuts like "subagent" to versioned IDs like "subagent@1.0.0"
    - Store returns nil (not error) when prompts are not found
  artifacts:
    - path: pkg/prompt/types.go
      provides: Core types (Prompt, PromptContext, SubAgentContext, AgentContext)
      contains: Prompt struct with Version field, PromptID constants
    - path: pkg/prompt/store/store.go
      provides: PromptStore interface
      exports: [SaveNewVersion, Load, Delete, List, Exists, ListTags, ResolveAlias, ListVersions, SetLatestAlias]
    - path: pkg/prompt/store/memory_store.go
      provides: In-memory PromptStore implementation
      exports: [NewMemoryStore]
    - path: pkg/prompt/store/file_store.go
      provides: File-based PromptStore implementation
      exports: [NewFileStore]
    - path: pkg/prompt/store/provider.go
      provides: DI provider for PromptStore
      exports: [NewPromptStoreProvider]
    - path: pkg/prompt/store/store_test.go
      provides: Test coverage for store implementations
    - path: pkg/mocks/generate.go
      provides: Centralized mock generation command
  key_links:
    - from: pkg/prompt/store/memory_store.go
      to: pkg/prompt/types.go
      via: Import and use prompt.Prompt
      pattern: prompt\.Prompt
    - from: pkg/prompt/store/file_store.go
      to: pkg/prompt/store/store.go
      via: Implements PromptStore interface
      pattern: PromptStore interface
    - from: pkg/prompt/store/provider.go
      to: pkg/prompt/store/store.go
      via: Returns PromptStore instances
      pattern: PromptStore
---

<objective>
Core Types and Store Layer Implementation

Purpose: Establish the foundational type system and persistence abstraction for versioned prompts. This is the infrastructure that all subsequent phases will build upon.

Output:
- Core types (Prompt, PromptContext, etc.) in pkg/prompt/types.go
- PromptStore interface with InMemory and File implementations
- DI provider for store initialization
- Comprehensive test coverage with table-driven tests
- Centralized mock generation for store interface
</objective>

<execution_context>
@/home/denkhaus/.claude/get-shit-done/workflows/execute-plan.md
@/home/denkhaus/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/01-core-types-and-store-layer/01-CONTEXT.md
@PROMPT_OPTIMIZER_PLAN.md
@.planning/codebase/CONVENTIONS.md
@.planning/codebase/TESTING.md
@.planning/codebase/ARCHITECTURE.md
@pkg/prompt/manager.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Read and understand Go guidance files</name>
  <files>/home/denkhaus/dev/kb/guide.golang.*.md</files>
  <action>
    Read all Go guidance files from /home/denkhaus/dev/kb/guide.golang.*.md to understand:
    - Go programming patterns and conventions
    - Testing patterns with table-driven tests and gomock
    - DI patterns with samber/do/v2
    - Error handling patterns with structured errors
    - Mock generation patterns with centralized generate.go

    Files to read:
    - guide.golang.programming.md
    - guide.golang.testing.md
    - guide.golang.di.md
    - guide.golang.config.md
    - guide.golang.logging.md
    - guide.golang.linting.md

    This ensures all implementation follows established project patterns and conventions.
  </action>
  <verify>Guidance files are read and understood (no output needed - knowledge check)</verify>
  <done>All Go guidance patterns are understood and will be applied during implementation</done>
</task>

<task type="auto">
  <name>Task 2: Create core types in pkg/prompt/types.go</name>
  <files>pkg/prompt/types.go</files>
  <action>
    Create pkg/prompt/types.go with the following types:

    1. Import required packages:
       - time for timestamps
       - github.com/Masterminds/semver/v3 for versioning

    2. Prompt struct with fields:
       - ID string (e.g., "system", "subagent@1.0.0")
       - Name string (human-readable name)
       - Content string (prompt text, may contain template syntax)
       - Context map[string]interface{} (optional context values)
       - Tags []string (optional categorization tags)
       - CreatedAt time.Time
       - UpdatedAt time.Time
       - Version *semver.Version (SemVer for change tracking)
       - IsBuiltin bool (built-in prompts cannot be deleted)

    3. PromptContext struct with fields:
       - Values map[string]interface{} (general values)
       - SubAgent *SubAgentContext
       - Agent *AgentContext

    4. SubAgentContext struct with fields:
       - Role string
       - Description string
       - SpawnAgentTool string
       - RemoveAgentTool string
       - ResumeAgentTool string
       - AgentOutputTool string
       - ListAgentsTool string

    5. AgentContext struct with fields:
       - AgentID string
       - Task string

    6. Built-in Prompt ID constants:
       - PromptIDSystem = "system"
       - PromptIDSupervisor = "supervisor"
       - PromptIDCompacter = "compacter"
       - PromptIDSubagent = "subagent"

    Follow existing codebase conventions from pkg/shared/ for struct definitions.
  </action>
  <verify>
    go build ./pkg/prompt/ completes without errors
    grep -q "type Prompt struct" pkg/prompt/types.go
    grep -q "PromptIDSystem" pkg/prompt/types.go
  </verify>
  <done>Core types are defined with proper SemVer versioning support</done>
</task>

<task type="auto">
  <name>Task 3: Create PromptStore interface in pkg/prompt/store/store.go</name>
  <files>pkg/prompt/store/store.go</files>
  <action>
    Create pkg/prompt/store/store.go with:

    1. Package declaration: package store

    2. Import required packages:
       - context for Context
       - errors for error definitions
       - github.com/Masterminds/semver/v3
       - github.com/denkhaus/gollum/pkg/prompt

    3. PromptStore interface with methods:
       - SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*prompt.Prompt, error)
         - Creates new version with auto-incremented patch version
         - Returns new prompt with versioned ID (e.g., "subagent@1.0.1")
       - Load(ctx context.Context, id string) (*prompt.Prompt, error)
         - Returns nil if not found (not an error)
       - Delete(ctx context.Context, id string) error
         - Returns nil if not found
         - Returns error for IsBuiltin prompts
       - List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error)
       - Exists(ctx context.Context, id string) (bool, error)
       - ListTags(ctx context.Context) ([]string, error)
       - ResolveAlias(ctx context.Context, id string) (*prompt.Prompt, error)
         - Resolves "subagent" → "subagent@latest" → versioned ID
       - ListVersions(ctx context.Context, baseID string) ([]*prompt.Prompt, error)
       - SetLatestAlias(ctx context.Context, baseID, versionID string) error

    4. ListFilter struct with fields:
       - Tags []string
       - IDs []string

    5. Error variables:
       - ErrPromptNotFound
       - ErrPromptIsBuiltin
       - ErrInvalidPromptID
       - ErrCASFailed

    6. PromptStoreConfig struct with envconfig tags:
       - Type string (default: "memory")
       - FilePath string (default: "./data/prompts")
       - LangfusePublicKey string
       - LangfuseSecretKey string
       - LangfuseHost string (default: "https://cloud.langfuse.com")
       - CacheEnabled bool (default: true)

    Follow interface design patterns from existing codebase (e.g., LoggerService, AgentRegistry).
  </action>
  <verify>
    go build ./pkg/prompt/store/ completes without errors
    grep -q "type PromptStore interface" pkg/prompt/store/store.go
    grep -q "SaveNewVersion" pkg/prompt/store/store.go
  </verify>
  <done>PromptStore interface defines complete contract for prompt persistence</done>
</task>

<task type="auto">
  <name>Task 4: Implement InMemory store in pkg/prompt/store/memory_store.go</name>
  <files>pkg/prompt/store/memory_store.go</files>
  <action>
    Create pkg/prompt/store/memory_store.go with:

    1. memoryStore struct with:
       - sync.RWMutex for concurrent access
       - prompts map[string]*prompt.Prompt

    2. NewMemoryStore() PromptStore constructor

    3. Implement all PromptStore interface methods:
       - Load: RLock, lookup, return prompt (nil if not found)
       - SaveNewVersion: Lock, find latest version, increment patch, create new prompt with:
         - Auto-incremented version (1.0.0 → 1.0.1)
         - ID as "{baseID}@{newVersion}"
         - CreatedAt/UpdatedAt as time.Now()
         - Remove aliases from old version, add to new version
       - Delete: Lock, check IsBuiltin (return error if true), delete from map
       - List: RLock, iterate prompts, apply filters, return copies
       - Exists: RLock, check map
       - ListTags: RLock, collect unique tags
       - ResolveAlias: Handle shortcuts ("subagent"), resolve @latest to versioned ID
       - ListVersions: Find all prompts matching base ID pattern
       - SetLatestAlias: Remove @latest from all versions, add to specified version

    4. Helper functions:
       - copyPrompt(prompt *prompt.Prompt) *prompt.Prompt (deep copy)
       - hasAnyTag(tags []string, filter []string) bool
       - containsAny(id string, ids []string) bool
       - incrementPatchVersion(v *semver.Version) *semver.Version

    Use sync.RWMutex for thread-safe access following patterns from pkg/registry/registry.go.
  </action>
  <verify>
    go test -v ./pkg/prompt/store/ -run TestMemoryStore
    grep -q "type memoryStore struct" pkg/prompt/store/memory_store.go
    grep -q "NewMemoryStore" pkg/prompt/store/memory_store.go
  </verify>
  <done>InMemory store implements full PromptStore interface with thread-safe operations</done>
</task>

<task type="auto">
  <name>Task 5: Implement File store in pkg/prompt/store/file_store.go</name>
  <files>pkg/prompt/store/file_store.go</files>
  <action>
    Create pkg/prompt/store/file_store.go with:

    1. fileStore struct with:
       - sync.RWMutex for cache synchronization
       - dir string (storage directory)
       - cache map[string]*prompt.Prompt (optional cache)
       - enableCache bool

    2. NewFileStore(dir string, enableCache bool) PromptStore constructor:
       - Create directory with os.MkdirAll(dir, 0755) if not exists
       - Initialize cache map

    3. Implement all PromptStore interface methods:
       - Load: Check cache first, then os.ReadFile, json.Unmarshal, update cache
       - SaveNewVersion:
         - Calculate next version by listing existing versions
         - Use syscall.Flock for exclusive file locking:
           - f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
           - syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
           - defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
         - Write JSON with json.Marshal
         - Update cache
       - Delete: Check IsBuiltin, os.Remove, remove from cache
       - List: os.ReadDir, filter .json files, load each
       - Exists: Check cache, then os.Stat
       - ListTags: Load all prompts, collect unique tags
       - ResolveAlias: Same logic as memory store
       - ListVersions: Scan directory for matching {baseID}@*.json files
       - SetLatestAlias: Update file metadata

    4. Helper functions:
       - getFilePath(id string) string (returns dir/id.json)
       - Same helper functions as memory_store.go (copyPrompt, hasAnyTag, etc.)

    Use syscall.Flock for cross-process file locking. Handle os.IsNotExist for "not found" cases.
    Return nil (not error) when prompt not found for Load, Delete, Exists methods.
  </action>
  <verify>
    go test -v ./pkg/prompt/store/ -run TestFileStore
    grep -q "syscall.Flock" pkg/prompt/store/file_store.go
    grep -q "type fileStore struct" pkg/prompt/store/file_store.go
  </verify>
  <done>File store persists prompts as JSON with file locking for concurrent safety</done>
</task>

<task type="auto">
  <name>Task 6: Create DI provider in pkg/prompt/store/provider.go</name>
  <files>pkg/prompt/store/provider.go</files>
  <action>
    Create pkg/prompt/store/provider.go with:

    1. Import required packages:
       - context
       - github.com/denkhaus/gollum/pkg/config
       - github.com/denkhaus/gollum/pkg/logger
       - github.com/samber/do/v2

    2. PromptStoreProvider interface:
       - GetStore() PromptStore

    3. storeProviderImpl struct with:
       - logService logger.LoggerService
       - configService config.ConfigService
       - store PromptStore

    4. NewPromptStoreProvider(injector do.Injector) (PromptStoreProvider, error) constructor:
       - Invoke LoggerService from injector
       - Invoke ConfigService from injector
       - Get PromptStoreConfig from config service
       - Switch on cfg.Type:
         - "file": NewFileStore(cfg.FilePath, cfg.CacheEnabled), log with type and path
         - "langfuse": NewLangfuseStore (log with host) - placeholder for future
         - "memory" or "": NewMemoryStore(), log with type
         - default: log warning, use NewMemoryStore()
       - Return &storeProviderImpl with initialized fields

    5. GetStore() PromptStore method returns p.store

    Follow DI provider patterns from pkg/tools/*_provider.go files.
    Log initialization with logger.String("type", cfg.Type) for observability.
  </action>
  <verify>
    go build ./pkg/prompt/store/ completes without errors
    grep -q "PromptStoreProvider" pkg/prompt/store/provider.go
    grep -q "NewPromptStoreProvider" pkg/prompt/store/provider.go
  </verify>
  <done>DI provider creates configured PromptStore instances based on configuration</done>
</task>

<task type="auto">
  <name>Task 7: Create comprehensive tests in pkg/prompt/store/store_test.go</name>
  <files>pkg/prompt/store/store_test.go</files>
  <action>
    Create pkg/prompt/store/store_test.go with:

    1. Table-driven tests for InMemory store:
       - TestMemoryStore_SaveNewVersion: Version increment, ID format, timestamps
       - TestMemoryStore_Load: Found/not found cases, nil returns for missing
       - TestMemoryStore_Delete: Success, not found (nil), IsBuiltin protection
       - TestMemoryStore_List: No filter, tag filter, ID filter
       - TestMemoryStore_Exists: True/false cases
       - TestMemoryStore_ResolveAlias: Shortcuts, @latest, versioned IDs
       - TestMemoryStore_ListVersions: Multiple versions of same base ID
       - TestMemoryStore_SetLatestAlias: Alias moves to new version
       - TestMemoryStore_ConcurrentAccess: Parallel goroutines reading/writing

    2. Table-driven tests for File store:
       - TestFileStore_SaveNewVersion: File creation, JSON format, file locking
       - TestFileStore_Load: File reading, cache behavior, not found (nil)
       - TestFileStore_Delete: File removal, IsBuiltin protection
       - TestFileStore_List: Directory scanning, filtering
       - TestFileStore_Exists: File existence check
       - TestFileStore_FileLocking: Concurrent writes with flock
       - TestFileStore_CacheBehavior: Cache hits/misses

    3. Helper functions:
       - createTestPrompt(id, content string) *prompt.Prompt
       - setupTestStore(t *testing.T) PromptStore (uses t.TempDir() for file store)

    Use t.Run() for subtests, table-driven test patterns from pkg/tools/bash_test.go.
    Use t.TempDir() for file store tests to avoid polluting working directory.
    Use testify/assert for assertions following existing test patterns.
  </action>
  <verify>
    go test -v ./pkg/prompt/store/ -cover
    go test -v ./pkg/prompt/store/ -run TestMemoryStore_ConcurrentAccess
    go test -v ./pkg/prompt/store/ -run TestFileStore_FileLocking
  </verify>
  <done>All store operations have comprehensive test coverage with concurrent access tests</done>
</task>

<task type="auto">
  <name>Task 8: Add mock generation to pkg/mocks/generate.go</name>
  <files>pkg/mocks/generate.go</files>
  <action>
    Update pkg/mocks/generate.go to add mock generation for PromptStore:

    1. Add go:generate comment for PromptStore interface:
       //go:generate mockgen -destination=mock_store.go -package=mocks github.com/denkhaus/gollum/pkg/prompt/store PromptStore

    2. Ensure file has proper package declaration (package mocks)

    3. Run go generate ./pkg/mocks/ to generate the mock

    Following the centralized mock generation pattern from existing pkg/mocks/generate.go.
  </action>
  <verify>
    grep -q "PromptStore" pkg/mocks/generate.go
    test -f pkg/mocks/mock_store.go after running go generate
  </verify>
  <done>Centralized mock generation includes PromptStore for use in tests</done>
</task>

</tasks>

<verification>
After completing all tasks, verify:

1. Build succeeds: go build ./pkg/prompt/...
2. All tests pass: go test -v ./pkg/prompt/store/...
3. Linter passes: golangci-lint run ./pkg/prompt/...
4. Coverage report: go test -cover ./pkg/prompt/store/...
5. Verify file locking: Check syscall.Flock usage in file_store.go
6. Verify nil returns: Check Load/Delete return nil for not found cases
7. Verify SemVer: Check version increment logic in SaveNewVersion
8. Verify aliases: Check ResolveAlias handles shortcuts and @latest
</verification>

<success_criteria>
- Prompt struct with SemVer versioning exists in pkg/prompt/types.go
- PromptStore interface defines complete persistence contract
- InMemory store implements full interface with thread-safe operations
- File store persists as JSON with syscall.Flock for concurrent access
- Alias resolution converts shortcuts to versioned IDs
- Load/Delete return nil (not error) for not found cases
- DI provider creates configured store instances
- Comprehensive tests cover all store operations including concurrent access
- Centralized mock generation includes PromptStore
- All code follows existing Gollum codebase conventions
</success_criteria>

<output>
After completion, create `.planning/phases/01-core-types-and-store-layer/01-01-SUMMARY.md`
</output>
