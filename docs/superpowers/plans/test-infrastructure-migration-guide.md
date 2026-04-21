# Test Infrastructure Migration Guide

This guide helps teams migrate existing tests to use the new shared test infrastructure.

## Quick Start

### Before (Old Pattern)
```go
func setupTestInjector() do.Injector {
	ctrl := gomock.NewController(&testing.T{})
	injector := do.New()
	
	do.Provide(injector, config.NewService)
	do.Provide(injector, logger.NewService)
	
	mockEventBus := events.NewMockBus(ctrl)
	mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("test-subscription-id", nil).AnyTimes()
	do.ProvideValue[events.Bus](injector, mockEventBus)
	// ... 20+ more lines ...
	
	return injector
}
```

### After (New Pattern)
```go
func setupTestInjector() do.Injector {
	return testutil.NewTestInjector(t)
}
```

## Benefits

- **70% reduction** in test setup code
- **Consistent mock behavior** across all tests
- **Easier maintenance** - changes in one place
- **Faster test writing** - get to the actual test faster

## Error Handling Migration

### Before
```go
if err := os.ReadFile(path); err != nil {
	return err
}
```

### After
```go
if err := os.ReadFile(path); err != nil {
	return shared.WrapInternal(err, "failed to read config file")
}
```

## Rollout Plan

1. **Week 1**: Migrate 3-5 test files as pilot
2. **Week 2**: Review feedback, adjust helpers if needed
3. **Week 3-4**: Migrate remaining test files
4. **Week 5**: Clean up old patterns

## Getting Help

- See `pkg/testutil/injector_test.go` for examples
- Check `pkg/channel/facade_test.go` for migrated example
- See `pkg/prompt/store/file_store.go` for error wrapping example
- Ask in `#dev-experience` channel
