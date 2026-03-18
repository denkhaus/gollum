# Gollum Codebase Quality Analysis

**Date:** 2026-03-18
**Scope:** Code quality, DRY, SOC, YAGNI, guide compliance

## Executive Summary

The codebase demonstrates good adherence to the established guides with a few areas for improvement. No critical violations were found, but several enhancement opportunities exist around code reuse and consolidation.

---

## 1. Guide Compliance Analysis

### 1.1 Configuration Guide ✅ COMPLIANT

**Status:** Fully compliant

- All environment variables handled through central `pkg/config/service.go`
- No direct `os.Getenv()` calls found outside config package
- Proper sub-config structure with `AnthropicConfig`, `GeminiConfig`, `FilesConfig`, etc.
- Getter methods provide clean access to configuration values
- Defaults are provided in the config structs

**Evidence:**
```go
// pkg/config/service.go - Proper structure
type serviceImpl struct {
    Anthropic       AnthropicConfig       `envconfig:"ANTHROPIC"`
    Gemini          GeminiConfig          `envconfig:"GEMINI"`
    // ... other sub-configs
}
```

### 1.2 Dependency Injection Guide ✅ COMPLIANT

**Status:** Fully compliant

- All services use `do.MustInvoke` for dependency retrieval
- Provider pattern is consistently used across packages
- Services are registered via `NewService` constructors
- No direct struct instantiation in business logic

**Evidence:**
```go
// Consistent pattern across all providers
func NewReadFileToolProvider(injector do.Injector) (ReadFileToolProvider, error) {
    logService := do.MustInvoke[logger.LoggerService](injector)
    fsm := do.MustInvoke[state.FileStateManager](injector)
    // ...
}
```

### 1.3 Logging Guide ✅ COMPLIANT

**Status:** Fully compliant

- `LoggerService` interface used throughout
- Structured logging with `zap.Field`
- Contextual logging with agent IDs
- No direct `log.` package usage found

### 1.4 Testing Guide - NOT REVIEWED

*(Testing patterns not analyzed in this review)*

### 1.5 Programming Guide ✅ MOSTLY COMPLIANT

**Status:** Minor observations

- Error handling uses `errs.Wrap` consistently
- Context propagation is proper
- File splitting is reasonable (largest files are ~1000 lines)

---

## 2. Duplicate Code Patterns (DRY Violations)

### 2.1 Tool Provider Pattern - HIGH DUPLICATION

**Location:** `pkg/tools/*.go`

**Issue:** Every tool file repeats the same provider pattern:

```go
// Repeated in grep.go, read_file.go, write_file.go, edit.go, etc.
type xxxToolProvider struct {
    logService  logger.LoggerService
    fsm         state.FileStateManager      // some have this
    hookManager hooks.HookManager
    diffProvider diff.Provider              // some have this
}

func NewXxxToolProvider(injector do.Injector) (XxxToolProvider, error) {
    logService := do.MustInvoke[logger.LoggerService](injector)
    fsm := do.MustInvoke[state.FileStateManager](injector)
    hookManager := do.MustInvoke[hooks.HookManager](injector)
    // ...
}
```

**Recommendation:** Create a shared `ToolProviderBase` in `pkg/tools/base.go`:

```go
type ToolProviderBase struct {
    LogService   logger.LoggerService
    FSM          state.FileStateManager
    HookManager  hooks.HookManager
    DiffProvider diff.Provider
}

func NewToolProviderBase(injector do.Injector) ToolProviderBase {
    return ToolProviderBase{
        LogService:   do.MustInvoke[logger.LoggerService](injector),
        FSM:          do.MustInvoke[state.FileStateManager](injector),
        HookManager:  do.MustInvoke[hooks.HookManager](injector),
        DiffProvider: do.MustInvoke[diff.Provider](injector),
    }
}
```

### 2.2 Argument Extraction Pattern - MEDIUM DUPLICATION

**Location:** `pkg/tools/*.go`

**Issue:** Repeated pattern for extracting and validating arguments:

```go
// Appears in multiple tool files
path, ok := args["file_path"].(string)
if !ok || path == "" {
    t.logService.Error("... failed: file_path is required...",
        zap.String("agent_id", t.agentID.String()))
    return map[string]any{
        "success": false,
        "error":   "file_path is required and must be a non-empty string",
    }, nil
}

// Convert relative path to absolute
path, err := filepath.Abs(path)
if err != nil {
    // ...
}
```

**Recommendation:** Create helper functions in `pkg/tools/args.go`:

```go
func ExtractFilePath(args map[string]any, log logger.LoggerService, agentID uuid.UUID) (string, error)
func ExtractRequiredString(args map[string]any, key string) (string, error)
func ExtractOptionalBool(args map[string]any, key string, def bool) bool
```

### 2.3 Error Response Pattern - MEDIUM DUPLICATION

**Issue:** Consistent error response structure is built manually everywhere:

```go
return map[string]any{
    string(shared.KeySuccess): false,
    string(shared.KeyError):   "error message",
}, nil
```

**Recommendation:** Create helper in `pkg/tools/response.go`:

```go
func ErrorResponse(format string, args ...any) map[string]any {
    return map[string]any{
        string(shared.KeySuccess): false,
        string(shared.KeyError):   fmt.Sprintf(format, args...),
    }
}

func SuccessResponse(data map[string]any) map[string]any {
    data[string(shared.KeySuccess)] = true
    return data
}
```

### 2.4 Glob Pattern Matching in Grep - MEDIUM DUPLICATION

**Location:** `pkg/tools/grep.go` lines 323-331, 404-412, 480-488

**Issue:** Same glob filter logic repeated 3 times in `grepContent`, `grepFiles`, and `grepCount`:

```go
if globPattern != "" {
    matched, err := filepath.Match(globPattern, filepath.Base(path))
    if err != nil {
        return err
    }
    if !matched {
        return nil
    }
}
```

**Recommendation:** Extract to helper function:

```go
func matchesGlobPattern(pattern, path string) (bool, error) {
    if pattern == "" {
        return true, nil
    }
    return filepath.Match(pattern, filepath.Base(path))
}
```

---

## 3. Separation of Concerns (SOC) Issues

### 3.1 Large Files Needing Splitting

| File | Lines | Recommendation |
|------|-------|----------------|
| `pkg/tui/model.go` | 1037 | Consider splitting into `model_core.go`, `model_state.go`, `model_handlers.go` |
| `pkg/flows/executor/executor.go` | 1048 | Already well-organized, minor candidate |
| `pkg/builtin/langfuse_hook.go` | 1052 | Large but cohesive - consider extracting span builders |
| `pkg/prompt/store/file_store.go` | 734 | Consider extracting file operations |
| `pkg/hooks/manager.go` | 652 | Consider extracting trigger logic per hook type |

### 3.2 Mixed Responsibilities

**Issue:** `pkg/tools/grep.go` handles both file walking AND searching logic.

**Recommendation:** Consider extracting:
- `fileWalker` - handles directory traversal with glob filtering
- `contentSearcher` - handles regex matching and context extraction

---

## 4. YAGNI Considerations

### 4.1 Potentially Unused Code

No significant unused code detected. The codebase appears lean.

### 4.2 Over-Engineering

None detected. Abstractions are appropriate for the current scale.

---

## 5. Shared Package Opportunities

### 5.1 Proposed: `pkg/tools/common`

Should contain:
- `ToolProviderBase` - shared provider dependencies
- `ErrorResponse()`, `SuccessResponse()` - response builders
- `ExtractFilePath()`, `ExtractRequiredString()` - argument helpers
- `matchesGlobPattern()` - glob matching helper

### 5.2 Proposed: `pkg/util/filepathx`

Should contain:
- `ResolveAbsolutePath(path string) (string, error)` - used in 5+ locations
- `IsFileStaleCheck()` - staleness check pattern

---

## 6. Code Quality Observations

### 6.1 Good Patterns Found

1. **Consistent interface definitions** - All services have clear interfaces
2. **Proper error wrapping** - Uses `errs.Wrap` with context
3. **Hook wrapping pattern** - Consistent use of `WithToolHooks`, `WithFileHooks`
4. **Dependency injection** - Clean DI pattern throughout
5. **Structured logging** - Consistent zap usage with agent context

### 6.2 Minor Issues

1. **Inconsistent key usage** - Some places use `shared.KeySuccess`, others use raw `"success"`
   - Example: `grep.go:138` uses `"success"` while `edit.go:112` uses `string(shared.KeySuccess)`

2. **Magic numbers** - Default values scattered (e.g., `limit := 200` in read_file.go:115)
   - Should be constants or config values

---

## 7. Action Items Summary

### High Priority
1. Create `pkg/tools/common` package with shared helpers
2. Standardize response key usage (always use `shared.Key*`)

### Medium Priority
3. Extract glob matching helper in grep.go
4. Consider splitting `pkg/tui/model.go` (1037 lines)

### Low Priority
5. Extract file walker logic from grep.go
6. Move default values to constants

---

## 8. Files Reviewed

- `pkg/config/service.go` - Configuration service
- `pkg/logger/logger.go` - Logging service
- `pkg/shared/agent.go` - Shared types
- `pkg/tools/grep.go` - Grep tool implementation
- `pkg/tools/read_file.go` - File reading tool
- `pkg/tools/write_file.go` - File writing tool
- `pkg/tools/edit.go` - File editing tool
- `pkg/registry/registry.go` - Agent registry
- `pkg/hooks/manager.go` - Hook manager interface
