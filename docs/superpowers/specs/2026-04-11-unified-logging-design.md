# Unified Logging Architecture Design

**Date:** 2026-04-11
**Status:** Approved
**Author:** Claude Code
**Related:** ACP Improvements, Channel Abstraction

## Executive Summary

This document describes a unified logging architecture that makes all agent-related logs session and channel-aware. Logs are routed through the channel system to appropriate destinations (ACP clients, TUI) based on their session and channel context.

**Current Problem:**
- Logger has no awareness of sessions or channels
- DisplayLog() in ChannelFacade broadcasts to all channels (wrong routing)
- Duplicate log buffers exist in logger and ChannelFacade
- TUI tightly coupled to logger's GetLogs() instead of being a proper channel

**Solution:**
- Introduce LoggingContext with SessionID/ChannelID/AgentID
- Create LogForwarder interface for clean separation
- Fix DisplayLog() routing to target specific channels
- Keep single buffer in logger package

## Architecture

### Components

#### 1. LoggingContext (shared package)

```go
type LoggingContext struct {
    SessionID string
    ChannelID uuid.UUID
    AgentID   uuid.UUID
}

func (c LoggingContext) IsValid() bool {
    return c.SessionID != "" &&
           c.ChannelID != uuid.Nil &&
           c.AgentID != uuid.Nil
}
```

- Holds routing information for log entries
- Strict validation: all fields must be valid or logging fails
- Used throughout logging call chain

#### 2. LogForwarder (shared package)

```go
type LogForwarder interface {
    ForwardLog(entry channel.LogEntry)
}
```

- Interface for channel-based log routing
- Implemented by ChannelFacade
- Allows logger to depend on abstraction, not concrete facade

#### 3. Logger Service Enhancements

**New API with context:**
```go
func (s *service) InfoWithContext(msg string, ctx LoggingContext, fields ...zap.Field)
func (s *service) ErrorWithContext(msg string, ctx LoggingContext, fields ...zap.Field)
func (s *service) DebugWithContext(msg string, ctx LoggingContext, fields ...zap.Field)
func (s *service) WarnWithContext(msg string, ctx LoggingContext, fields ...zap.Field)
```

**Backward compatible:**
```go
func (s *service) InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    ctx := LoggingContext{AgentID: agentID}
    s.InfoWithContext(msg, ctx, fields...)
}
```

**Internal implementation:**
```go
func (s *service) logWithContext(level string, msg string, ctx LoggingContext, fields ...zap.Field) {
    // Validate context
    if !ctx.IsValid() {
        s.logger.Error("LoggingContext is incomplete - log not processed", ...)
        return
    }

    // Add context as zap fields
    allFields := append([]zap.Field{
        zap.String("session_id", ctx.SessionID),
        zap.String("channel_id", ctx.ChannelID.String()),
        zap.String("agent_id", ctx.AgentID.String()),
    }, fields...)

    // Log to zap
    // ...

    // Store in buffer
    entry := LogEntry{
        Timestamp: time.Now(),
        Level:     level,
        Message:   msg,
        Fields:    zapFieldsToMap(allFields),
        AgentID:   ctx.AgentID,
        SessionID: ctx.SessionID,
        ChannelID: ctx.ChannelID,
    }
    s.logBuffer.add(entry)

    // Forward to channel facade
    if s.forwarder != nil {
        s.forwarder.ForwardLog(entry)
    }
}
```

#### 4. ChannelFacade Fixes

**Remove duplicate buffer:**
- Delete `p.logs`, `p.maxLogs`
- Remove `GetLogs()` method

**Fix DisplayLog() routing:**
```go
func (p *channelFacadeImpl) DisplayLog(entry LogEntry) {
    p.mu.RLock()
    channel, exists := p.channels[entry.ChannelID]
    p.mu.RUnlock()

    if !exists {
        p.logger.Warn("channel not found for log entry",
            zap.String("channel_id", entry.ChannelID.String()),
            zap.String("session_id", entry.SessionID),
        )
        return
    }

    // Forward to specific channel only
    channel.OnLog(entry)
}
```

**Implement LogForwarder:**
```go
func (p *channelFacadeImpl) ForwardLog(entry channel.LogEntry) {
    p.DisplayLog(entry)
}
```

#### 5. TUI Channel (No Changes Needed)

TUIChannel already implements `channel.Channel` with working `OnLog()`:
- Converts LogEntry to MessageTypeSystemInfo
- Formats with `[LEVEL] prefix`
- Sends to messageChan for display

**How it works:**
```
Logger.ForwardLog(entry)
  → facade.DisplayLog(entry)
  → routes to entry.ChannelID
  → TUIChannel.OnLog(entry) if matches
  → TUI displays as system message
```

## Data Flow

```
Agent Execution
  ↓
Logger.InfoWithContext(msg, LoggingContext{
    SessionID: "session-123",
    ChannelID: tui-channel-uuid,
    AgentID: agent-uuid,
  })
  ↓
┌─────────────────────────────────────┐
│ Logger Service                      │
│  1. Validate LoggingContext         │
│  2. Log to zap (stdout/file)        │
│  3. Store in logBuffer              │
│  4. forwarder.ForwardLog(entry)     │
└─────────────────────────────────────┘
  ↓
┌─────────────────────────────────────┐
│ ChannelFacade (LogForwarder)        │
│  1. DisplayLog(entry)               │
│  2. Route to entry.ChannelID        │
└─────────────────────────────────────┘
  ↓
┌─────────────────────────────────────┐
│ Specific Channel (TUI/ACP)          │
│  1. OnLog(entry)                    │
│  2. Process/log/forward             │
└─────────────────────────────────────┘
```

## Error Handling

**Strict validation:**
- LoggingContext must be complete or logging fails immediately
- Empty SessionID or Nil UUID = invalid
- No partial contexts allowed

**ChannelFacade:**
- Channel not found → Log warning, return (fail fast)
- Invalid LogEntry → Should never happen if logger validates

**Logger degradation:**
- Works fine if forwarder is nil (just no channel routing)
- Allows startup before ChannelFacade is registered
- Useful for testing without full DI setup

## Implementation Order

### Phase 1: Foundation (No Breaking Changes)
1. Create `pkg/shared/logging_context.go`
   - LoggingContext struct
   - IsValid() method

2. Create `pkg/shared/log_forwarder.go`
   - LogForwarder interface

3. Add to logger service:
   - Internal `logWithContext()` method
   - `SetLogForwarder(forwarder LogForwarder)` method

### Phase 2: Logger Enhancements
1. Implement new `*WithContext()` methods
   - InfoWithContext, ErrorWithContext, etc.

2. Update existing `*WithAgent()` methods
   - Use `logWithContext()` internally
   - Create minimal LoggingContext for backward compatibility

3. Add LoggingContext validation
   - Fail fast on invalid contexts

### Phase 3: Channel Integration
1. ChannelFacade implements LogForwarder
   - ForwardLog() calls DisplayLog()

2. Fix DisplayLog() routing
   - Remove duplicate buffer (logs, maxLogs, GetLogs)
   - Route to specific channel by ChannelID

3. Wire up in DI container
   - Inject facade into logger via SetLogForwarder()

### Phase 4: Cleanup
1. Update any remaining code that uses old patterns
2. Update documentation
3. Add examples in code comments

## File Structure

**New files:**
```
pkg/shared/
  logging_context.go  // LoggingContext + IsValid()
  log_forwarder.go     // LogForwarder interface
```

**Modified files:**
```
pkg/logger/
  logger.go           // Add logWithContext(), SetLogForwarder(), *WithContext()

pkg/channel/
  facade.go            // Implement LogForwarder, fix DisplayLog()
  interface.go         // Remove GetLogs() from ChannelFacade

pkg/di/
  container.go         // Wire logger.SetLogForwarder(facade)
```

**No changes needed:**
```
pkg/tui/channel.go    // Already implements OnLog() correctly
pkg/acp/service.go    // Already implements OnLog() with routing
```

## Dependencies

```
pkg/logger → pkg/shared (LoggingContext, LogForwarder)
pkg/channel → pkg/shared (LogForwarder interface)
pkg/acp → pkg/channel (uses DisplayLog)
pkg/tui → pkg/channel (uses channel.Channel)
```

**No circular dependencies** - all imports flow toward shared interfaces.

## Testing Strategy

### Unit Tests

**Logger tests:**
- Mock LogForwarder to verify ForwardLog() called with correct entry
- Test LoggingContext validation (valid/invalid)
- Test with nil forwarder (graceful degradation)
- Test backward compatibility of *WithAgent() methods

**ChannelFacade tests:**
- Test DisplayLog() routing to correct channel
- Test error handling when channel not found
- Test with multiple channels (verify no cross-talk)

### Integration Tests

- Logger → LogForwarder → ChannelFacade → TUIChannel
- Verify logs flow through entire chain with correct SessionID/ChannelID
- Test ACP channel routing (specific session vs broadcast)

### Test Helpers

- `NewTestLoggingContext(sessionID, channelID, agentID)` helper
- Mock LogForwarder implementation
- Channel test fixtures

**Follow:** Go Testing Guide at `/home/denkhaus/dev/kb/guides/guide.golang.testing.md`

## Migration Notes

**For existing code using *WithAgent():**
- No changes needed - backward compatible
- Logs will have empty SessionID/ChannelID (no channel routing)

**For new agent code:**
- Use `*WithContext()` methods with full LoggingContext
- Provides session/channel-aware routing

**TUI:**
- Already works via OnLog()
- Can eventually remove direct logger buffer access

## Success Criteria

1. ✅ All agent-related logs are session/channel-aware
2. ✅ DisplayLog() routes to specific channel, not broadcast
3. ✅ No duplicate buffers (single source of truth in logger)
4. ✅ TUI receives logs via OnLog(), not direct buffer access
5. ✅ ACP receives logs for correct session
6. ✅ Clean separation with LogForwarder interface
7. ✅ Strict validation of LoggingContext
8. ✅ Backward compatible with existing code

## Open Questions

None identified.
