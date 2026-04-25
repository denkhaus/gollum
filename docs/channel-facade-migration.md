# Channel Facade Refactor - Migration Guide

## Summary

The `ChannelFacade.SubmitInput` method now delegates to a new `InputHandler` interface that encapsulates session and supervisor management logic.

## What Changed

### Before (Business logic in facade)
```go
type ChannelFacade interface {
    SubmitInput(ctx, channelID, sessionID, input) (*InputResult, error)
    // SubmitInput contained: command handling, session creation, supervisor creation
}
```

### After (Delegates to InputHandler)
```go
// New interface for business logic
type InputHandler interface {
    HandleInput(ctx, channelID, sessionID, input) (*InputResult, error)
    CancelInput(sessionID) error
}

// ChannelFacade delegates to InputHandler
type ChannelFacade interface {
    SubmitInput(...) (*InputResult, error)  // Now delegates to InputHandler
    CancelInput(...) error                   // Now delegates to InputHandler
    // ... other methods unchanged
}
```

## Impact

### For Channel Consumers (ACP, TUI, etc.)

**No changes required!** The `ChannelFacade` interface maintains backward compatibility.

```go
// This still works exactly as before
facade.SubmitInput(ctx, channelID, sessionID, input)
facade.CancelInput(sessionID)
```

### For Testing

You can now mock `InputHandler` directly instead of the entire facade:

```go
// Before: Mock entire facade
mockFacade := channel.NewMockChannelFacade(ctrl)
mockFacade.EXPECT().SubmitInput(...)

// After: Mock just the input handler (more focused)
mockHandler := channel.NewMockInputHandler(ctrl)
mockHandler.EXPECT().HandleInput(...)
```

### For Extensibility

You can provide custom input handling by implementing `InputHandler`:

```go
// Custom input handler with different session management
type CustomInputHandler struct {
    // ... custom fields
}

func (h *CustomInputHandler) HandleInput(ctx, channelID, sessionID, input) (*InputResult, error) {
    // Custom session/supervisor logic
}

// Inject custom handler (requires constructor modification)
facade := NewChannelFacadeWithHandler(customHandler)
```

## Benefits

1. **Better SOC**: Business logic separated from infrastructure
2. **Easier testing**: Mock InputHandler instead of entire facade
3. **Extensibility**: Swap input handling strategies without touching facade
4. **No breaking changes**: Existing code continues to work

## Questions?

See `pkg/channel` package documentation for log routing behavior and implementation guidelines.
