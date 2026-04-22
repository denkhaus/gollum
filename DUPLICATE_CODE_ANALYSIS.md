# Duplicate Code Analysis Report

## Executive Summary
This report identifies duplicate code patterns across the gollum codebase and recommends opportunities for creating reusable shared utilities in `pkg/shared/`.

## Current State
- **Total packages analyzed**: 33 packages in `pkg/`
- **Current shared utilities**: 28 files in `pkg/shared/`
- **Test files**: 100+ test files across packages

## Critical Duplicate Patterns Found

### 1. Test Setup & Mock Injection Patterns ⚠️ HIGH IMPACT

**Pattern**: Repeated DI container setup with similar mock configurations

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go` (lines 217-290)
- `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/manager_lifecycle_test.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/service_test.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/audit_test.go`

**Duplicate Code**:
```go
// Found in multiple test files
func setupTestInjector() do.Injector {
    injector := do.New()
    do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})
    do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
    do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
    // ... repeated across 10+ test files
}
```

**Recommendation**: Create `pkg/shared/test_injector.go`
- **Effort**: Medium (2-3 hours)
- **Benefit**: High (reduces test setup code by ~70%)
- **Impact**: 15+ test files

**Functions to create**:
- `SetupTestInjector(opts ...TestInjectorOption) do.Injector`
- `SetupMockLogger(ctrl *gomock.Controller) *MockLoggerService`
- `SetupMockConfig(cfg *config.ConfigService) *MockConfigService`
- `SetupMockSessionManager(ctrl *gomock.Controller) *MockSessionManager`

---

### 2. Mock Implementation Duplication ⚠️ HIGH IMPACT

**Pattern**: Identical mock structs implemented in multiple test files

**Locations**:
- `mockChannel` - `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go`
- `mockCommandManager` - `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go`
- `mockAgentFactory` - `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go`
- `mockAgent` - `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go`
- Similar mocks in extensions, hooks, flows packages

**Recommendation**: Consolidate to `pkg/shared/test_mocks.go`
- **Effort**: Low (1-2 hours)
- **Benefit**: High (eliminates duplicate mock definitions)
- **Impact**: 20+ test files

**Already exists** but underutilized:
- `pkg/shared/agent_mock.go` - Good foundation
- `pkg/shared/factory_mock.go` - Good foundation

**Missing mocks to add**:
- `TestChannel` (generic channel mock)
- `TestCommandManager` (generic command manager mock)
- `TestAgent` (generic agent mock with configurable behavior)

---

### 3. Logger Mock Setup Patterns ⚠️ MEDIUM IMPACT

**Pattern**: Repetitive gomock logger expectations setup

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/channel/facade_test.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/audit_test.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/tui/model_handlers_test.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/manager_llm_hooks_test.go`

**Duplicate Code**:
```go
mockLogger := logger.NewMockLoggerService(ctrl)
mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
```

**Recommendation**: Create helper in `pkg/shared/test_helpers.go`
```go
func SetupPermissiveMockLogger(ctrl *gomock.Controller) *logger.MockLoggerService {
    mockLogger := logger.NewMockLoggerService(ctrl)
    mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
    mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
    mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
    mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
    return mockLogger
}
```

- **Effort**: Low (30 minutes)
- **Benefit**: Medium (reduces boilerplate in 15+ tests)
- **Impact**: 15+ test files

---

### 4. Error Handling Patterns ⚠️ MEDIUM IMPACT

**Pattern**: Similar error wrapping and creation patterns

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/yaegi_func_runner.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/yaegi.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/cli/root.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/tui/options.go`

**Duplicate Patterns**:
```go
// Pattern 1: Error wrapping with context
return fmt.Errorf("operation failed: %w", err)

// Pattern 2: Type-specific errors
return fmt.Errorf("invalid type (got %T, expected *ExpectedType)", value)

// Pattern 3: Not found errors
return fmt.Errorf("%s not found: %s", entity, identifier)
```

**Recommendation**: Create `pkg/shared/errors.go`
- **Effort**: Low (1 hour)
- **Benefit**: Medium (consistent error handling)
- **Impact**: 30+ files

**Functions to create**:
- `WrapError(op string, err error) error`
- `TypeError(value interface{}, expectedType string) error`
- `NotFoundError(entity, id string) error`
- `ValidationError(field, reason string) error`

---

### 5. Validation Logic Duplication ⚠️ LOW-MEDIUM IMPACT

**Pattern**: Repeated nil checks and validation logic

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/typed_registry.go` (lines with "if r.names == nil")
- `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/executor_hooks_wrapper.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/tui/channel.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/extensions/validation.go`

**Duplicate Patterns**:
```go
// Pattern 1: Nil slice initialization
if s == nil {
    s = make([]Type, 0)
}

// Pattern 2: Map initialization
if m == nil {
    m = make(map[Key]Value)
}

// Pattern 3: String validation
if strings.TrimSpace(s) == "" {
    return errors.New("cannot be empty")
}
```

**Recommendation**: Create `pkg/shared/validation.go`
- **Effort**: Low (1 hour)
- **Benefit**: Low-Medium (consistency gains)
- **Impact**: 20+ files

**Functions to create**:
- `EnsureSlice[T any](s []T) []T`
- `EnsureMap[K comparable, V any](m map[K]V) map[K]V`
- `RequireNonEmpty(s string) error`
- `RequireNotEmpty[T comparable](v T) error`

---

### 6. String Conversion & Formatting ⚠️ LOW IMPACT

**Pattern**: Repeated string conversion and formatting logic

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/tui/model_formatting.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/hooks/types.go`
- `/home/denkhaus/dev/gomodules/gollum/pkg/mcp/config/interpolate.go`

**Already exists**:
- `pkg/shared/conversions.go` - Good start with basic conversions

**Missing utilities**:
- Safe string conversions from various types
- Truncation with ellipsis
- Multi-line string formatting
- UUID string formatting helpers

**Recommendation**: Extend `pkg/shared/conversions.go`
- **Effort**: Low (30 minutes)
- **Benefit**: Low (minor consistency improvement)
- **Impact**: 10+ files

---

### 7. Type Assertion Patterns ⚠️ LOW IMPACT

**Pattern**: Repeated type assertion patterns from `interface{}`

**Locations**:
- `/home/denkhaus/dev/gomodules/gollum/pkg/shared/shared.go` (ToolResult methods)
- Various packages handling configuration and agent data

**Already exists**:
- Good patterns in `ToolResult` type with `GetString()`, `GetInt()`, `GetBool()`

**Recommendation**: Extract to generic utilities
- **Effort**: Low (30 minutes)
- **Benefit**: Low (code reuse opportunity)
- **Impact**: 5-10 files

---

## Priority Recommendations

### Immediate Actions (High ROI)

1. **Create Test Infrastructure Package** ⭐⭐⭐
   - `pkg/shared/test_injector.go` - Reusable DI setup
   - `pkg/shared/test_mocks.go` - Consolidated mock implementations
   - `pkg/shared/test_helpers.go` - Common test utilities
   - **Estimated effort**: 4-6 hours
   - **Impact**: 30+ test files, 70% reduction in test setup code

2. **Standardize Error Handling** ⭐⭐
   - `pkg/shared/errors.go` - Common error creation helpers
   - **Estimated effort**: 1-2 hours
   - **Impact**: 30+ files, improved error consistency

### Short-term Actions (Medium ROI)

3. **Validation Utilities** ⭐
   - `pkg/shared/validation.go` - Common validation patterns
   - **Estimated effort**: 1 hour
   - **Impact**: 20+ files

4. **Extend Conversion Utilities** ⭐
   - Enhance `pkg/shared/conversions.go`
   - **Estimated effort**: 30 minutes
   - **Impact**: 10+ files

## Estimated Total Effort vs. Benefit

| Priority | Effort | Files Affected | Code Reduction | Benefit |
|----------|--------|----------------|----------------|---------|
| High | 6-8 hours | 30+ | ~2000 lines | High |
| Medium | 2 hours | 30+ | ~500 lines | Medium |
| Low | 1 hour | 15+ | ~200 lines | Low |

## Implementation Strategy

### Phase 1: Test Infrastructure (Week 1)
1. Create `pkg/shared/test_injector.go`
2. Create `pkg/shared/test_mocks.go`
3. Create `pkg/shared/test_helpers.go`
4. Migrate 3-5 test files as proof of concept
5. Document usage patterns

### Phase 2: Core Utilities (Week 2)
1. Create `pkg/shared/errors.go`
2. Create `pkg/shared/validation.go`
3. Extend `pkg/shared/conversions.go`
4. Update 5-10 source files

### Phase 3: Migration (Week 3-4)
1. Migrate remaining test files
2. Update source files to use new utilities
3. Remove duplicate code
4. Update documentation

## Files to Create

### High Priority
1. `/home/denkhaus/dev/gomodules/gollum/pkg/shared/test_injector.go`
2. `/home/denkhaus/dev/gomodules/gollum/pkg/shared/test_mocks.go`
3. `/home/denkhaus/dev/gomodules/gollum/pkg/shared/test_helpers.go`
4. `/home/denkhaus/dev/gomodules/gollum/pkg/shared/errors.go`

### Medium Priority
5. `/home/denkhaus/dev/gomodules/gollum/pkg/shared/validation.go`

## Existing Good Practices to Maintain

The codebase already has some good patterns in `pkg/shared/`:
- ✅ `agent_mock.go` - Well-structured mock
- ✅ `factory_mock.go` - Factory pattern mock
- ✅ `conversions.go` - Type conversion utilities
- ✅ `shared.go` - Common types and constants

These should be extended rather than replaced.

## Conclusion

The gollum codebase has significant opportunities for code deduplication, particularly in test infrastructure. The most impactful changes involve consolidating test setup patterns and mock implementations, which would reduce code duplication by approximately 2,500 lines across 30+ files.

The recommended approach is to start with test infrastructure (highest ROI), then move to core utilities, and finally migrate existing code incrementally.
