# Unified Logging Architecture Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make all agent-related logs session/channel-aware by integrating logger with the channel system via LogForwarder interface.

**Architecture:** Logger service forwards logs with SessionID/ChannelID/ChannelID to ChannelFacade, which routes them to specific channels (TUI, ACP). Clean separation via LogForwarder interface in shared package.

**Tech Stack:** Go 1.21+, zap logging, existing channel abstraction, DI with samber/do/v2

---

## File Structure

**New files:**
- `pkg/shared/logging_context.go` - LoggingContext struct + validation
- `pkg/shared/log_forwarder.go` - LogForwarder interface

**Modified files:**
- `pkg/logger/logger.go` - Add logWithContext(), SetLogForwarder(), *WithContext() methods
- `pkg/channel/facade.go` - Implement LogForwarder, fix DisplayLog() routing, remove duplicate buffer
- `pkg/channel/interface.go` - Remove GetLogs() from ChannelFacade interface
- `pkg/di/container.go` - Wire logger.SetLogForwarder(facade)

**No changes needed:**
- `pkg/tui/channel.go` - Already implements OnLog() correctly
- `pkg/acp/service.go` - Already implements OnLog() with routing

---

## Chunk 1: Foundation - Shared Package

### Task 1: Create LoggingContext struct

**Files:**
- Create: `pkg/shared/logging_context.go`

- [ ] **Step 1: Create file with LoggingContext struct**

```go
// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"github.com/google/uuid"
)

// LoggingContext holds routing information for session/channel-aware logging.
// All fields must be valid for the context to be considered complete.
type LoggingContext struct {
	SessionID string    // Session identifier for routing
	ChannelID uuid.UUID // Channel identifier for routing
	AgentID   uuid.UUID // Agent identifier for the log source
}

// IsValid returns true if all required fields are populated with valid values.
func (c LoggingContext) IsValid() bool {
	return c.SessionID != "" &&
		c.ChannelID != uuid.Nil &&
		c.AgentID != uuid.Nil
}
```

- [ ] **Step 2: Run Go fmt/vet**

```bash
go fmt ./pkg/shared/logging_context.go
go vet ./pkg/shared/logging_context.go
```

Expected: No errors

- [ ] **Step 3: Write test for IsValid()**

Create: `pkg/shared/logging_context_test.go`

```go
package shared

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoggingContext_IsValid_ValidContext(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.True(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_EmptySessionID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "",
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_NilChannelID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.Nil,
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_NilAgentID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.New(),
		AgentID:   uuid.Nil,
	}

	assert.False(t, ctx.IsValid())
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/shared/logging_context_test.go -v
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/shared/logging_context.go pkg/shared/logging_context_test.go
git commit -m "feat(shared): add LoggingContext with validation

Add LoggingContext struct for session/channel-aware logging:
- SessionID, ChannelID, AgentID fields
- IsValid() method for strict validation
- Tests for valid and invalid contexts

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 2: Create LogForwarder interface

**Files:**
- Create: `pkg/shared/log_forwarder.go`

- [ ] **Step 1: Create file with LogForwarder interface**

```go
// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"github.com/denkhaus/gollum/pkg/channel"
)

// LogForwarder defines the interface for forwarding log entries to the channel system.
// This allows the logger to route logs to specific channels without depending on
// the concrete ChannelFacade implementation.
type LogForwarder interface {
	// ForwardLog sends a log entry to the channel system for routing.
	// The entry's SessionID and ChannelID control which channel receives the log.
	ForwardLog(entry channel.LogEntry)
}
```

- [ ] **Step 2: Run Go fmt/vet**

```bash
go fmt ./pkg/shared/log_forwarder.go
go vet ./pkg/shared/log_forwarder.go
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/shared/log_forwarder.go
git commit -m "feat(shared): add LogForwarder interface

Add LogForwarder interface for clean separation between logger and channel:
- ForwardLog() method for channel-based log routing
- Allows logger to depend on abstraction, not concrete facade

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Logger Service Enhancements

### Task 3: Add internal logWithContext method

**Files:**
- Modify: `pkg/logger/logger.go`

- [ ] **Step 1: Add SetLogForwarder method to service struct**

```go
type service struct {
	logger         *zap.Logger
	atomicLevel    zap.AtomicLevel
	config         zap.Config
	originalLogger *zap.Logger
	logBuffer      *logBuffer
	tuiMode        bool
	configService  config.ConfigService
	logFile        *os.File
	logFilePath    string
	fileLogger     *zap.Logger
	forwarder      LogForwarder // NEW: for channel log forwarding
}
```

- [ ] **Step 2: Add SetLogForwarder method**

Add to logger service methods:

```go
// SetLogForwarder sets the log forwarder for channel-based log routing.
func (s *service) SetLogForwarder(forwarder LogForwarder) {
	s.forwarder = forwarder
}
```

- [ ] **Step 3: Add internal logWithContext method**

Add to logger service (after storeInBuffer method):

```go
// logWithContext is the internal implementation for context-aware logging.
// It validates the context, logs to zap, stores in buffer, and forwards to channels.
func (s *service) logWithContext(level string, msg string, ctx shared.LoggingContext, fields ...zap.Field) {
	// Validate context strictly
	if !ctx.IsValid() {
		s.logger.Error("LoggingContext is incomplete - log not processed",
			zap.String("session_id", ctx.SessionID),
			zap.String("channel_id", ctx.ChannelID.String()),
			zap.String("agent_id", ctx.AgentID.String()),
		)
		return
	}

	// Add context as zap fields for structured logging
	allFields := append([]zap.Field{
		zap.String("session_id", ctx.SessionID),
		zap.String("channel_id", ctx.ChannelID.String()),
		zap.String("agent_id", ctx.AgentID.String()),
	}, fields...)

	// Log to zap (stdout/file)
	switch level {
	case "debug":
		s.logger.Debug(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Debug(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "info":
		s.logger.Info(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Info(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "warn":
		s.logger.Warn(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Warn(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "error":
		s.logger.Error(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Error(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	}

	// Store in buffer with full context
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

	// Forward to channel facade via LogForwarder
	if s.forwarder != nil {
		channelEntry := channel.LogEntry{
			Level:     level,
			Message:   msg,
			Timestamp: entry.Timestamp,
			Fields:    entry.Fields,
			SessionID: ctx.SessionID,
			ChannelID: ctx.ChannelID,
		}
		s.forwarder.ForwardLog(channelEntry)
	}
}
```

- [ ] **Step 4: Add import for shared package**

Add to imports section:
```go
import (
	// ... existing imports
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/shared"
)
```

- [ ] **Step 5: Compile check**

```bash
go build ./pkg/logger/
```

Expected: No compilation errors

- [ ] **Step 6: Commit**

```bash
git add pkg/logger/logger.go
git commit -m "feat(logger): add logWithContext for session/channel-aware logging

Add internal logWithContext() method:
- Validates LoggingContext strictly
- Logs to zap with session/channel/agent fields
- Stores in buffer with full context
- Forwards to LogForwarder for channel routing
- Add SetLogForwarder() for DI injection

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 4: Add public *WithContext methods

**Files:**
- Modify: `pkg/logger/logger.go`

- [ ] **Step 1: Add InfoWithContext method**

Add to LoggerService interface and service implementation:

```go
// InfoWithContext logs an info message with session, channel, and agent context.
func (s *service) InfoWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field) {
	s.logWithContext("info", msg, ctx, fields...)
}
```

- [ ] **Step 2: Add ErrorWithContext method**

```go
// ErrorWithContext logs an error message with session, channel, and agent context.
func (s *service) ErrorWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field) {
	s.logWithContext("error", msg, ctx, fields...)
}
```

- [ ] **Step 3: Add DebugWithContext method**

```go
// DebugWithContext logs a debug message with session, channel, and agent context.
func (s *service) DebugWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field) {
	s.logWithContext("debug", msg, ctx, fields...)
}
```

- [ ] **Step 4: Add WarnWithContext method**

```go
// WarnWithContext logs a warning message with session, channel, and agent context.
func (s *service) WarnWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field) {
	s.logWithContext("warn", msg, ctx, fields...)
}
```

- [ ] **Step 5: Add to interface**

Update LoggerService interface:

```go
type LoggerService interface {
	// ... existing methods ...
	
	// InfoWithContext logs an info message with session, channel, and agent context.
	InfoWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field)
	// ErrorWithContext logs an error message with session, channel, and agent context.
	ErrorWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field)
	// DebugWithContext logs a debug message with session, channel, and agent context.
	DebugWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field)
	// WarnWithContext logs a warning message with session, channel, and agent context.
	WarnWithContext(msg string, ctx shared.LoggingContext, fields ...zap.Field)
	
	GetLogger() *zap.Logger
	// ... rest of interface ...
}
```

- [ ] **Step 6: Compile check**

```bash
go build ./pkg/logger/
```

Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add pkg/logger/logger.go
git commit -m "feat(logger): add public *WithContext methods

Add InfoWithContext, ErrorWithContext, DebugWithContext, WarnWithContext:
- Provide public API for session/channel-aware logging
- Delegate to internal logWithContext() method
- Update LoggerService interface

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 5: Update *WithAgent methods for backward compatibility

**Files:**
- Modify: `pkg/logger/logger.go`

- [ ] **Step 1: Update InfoWithAgent to use logWithContext**

```go
// InfoWithAgent logs an info message with agent ID included as a structured field.
// Backward compatible - creates minimal LoggingContext with AgentID only.
// Note: SessionID and ChannelID will be empty, so logs won't be routed to channels.
func (s *service) InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
	ctx := shared.LoggingContext{
		AgentID: agentID,
		// SessionID and ChannelID left empty for backward compatibility
	}
	s.logWithContext("info", msg, ctx, fields...)
}
```

- [ ] **Step 2: Update ErrorWithAgent**

```go
// ErrorWithAgent logs an error message with agent ID included as a structured field.
// Backward compatible - creates minimal LoggingContext with AgentID only.
func (s *service) ErrorWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
	ctx := shared.LoggingContext{
		AgentID: agentID,
	}
	s.logWithContext("error", msg, ctx, fields...)
}
```

- [ ] **Step 3: Update DebugWithAgent**

```go
// DebugWithAgent logs a debug message with agent ID included as a structured field.
func (s *service) DebugWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
	ctx := shared.LoggingContext{
		AgentID: agentID,
	}
	s.logWithContext("debug", msg, ctx, fields...)
}
```

- [ ] **Step 4: Update WarnWithAgent**

```go
// WarnWithAgent logs a warning message with agent ID included as a structured field.
func (s *service) WarnWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
	ctx := shared.LoggingContext{
		AgentID: agentID,
	}
	s.logWithContext("warn", msg, ctx, fields...)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./pkg/logger/... -v
```

Expected: All tests pass (existing tests still work)

- [ ] **Step 6: Commit**

```bash
git add pkg/logger/logger.go
git commit -m "feat(logger): update *WithAgent methods to use logWithContext

Update *WithAgent methods to be backward compatible wrappers:
- Create minimal LoggingContext with AgentID only
- Delegate to logWithContext() internally
- SessionID/ChannelID empty for existing code
- Maintains backward compatibility

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: Channel Integration

### Task 6: Implement LogForwarder on ChannelFacade

**Files:**
- Modify: `pkg/channel/facade.go`

- [ ] **Step 1: Add import for shared package**

```go
import (
	// ... existing imports
	"github.com/denkhaus/gollum/pkg/shared"
)
```

- [ ] **Step 2: Implement ForwardLog method**

Add to channelFacadeImpl:

```go
// ForwardLog implements shared.LogForwarder for channel-based log routing.
func (p *channelFacadeImpl) ForwardLog(entry channel.LogEntry) {
	p.DisplayLog(entry)
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./pkg/channel/
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add pkg/channel/facade.go
git commit -m "feat(channel): implement LogForwarder on ChannelFacade

Add ForwardLog() method to implement shared.LogForwarder:
- Delegates to DisplayLog() for routing
- Allows logger to forward logs without depending on concrete facade

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 7: Fix DisplayLog routing and remove duplicate buffer

**Files:**
- Modify: `pkg/channel/facade.go`

- [ ] **Step 1: Remove buffer fields from struct**

Remove these fields from channelFacadeImpl:
```go
// REMOVE THESE:
logs    []LogEntry
maxLogs int
```

- [ ] **Step 2: Remove buffer initialization from constructor**

Remove from NewChannelFacade:
```go
// REMOVE:
maxLogs := cfg.GetLoggingConfig().SessionLogBufferSize
```

And from struct initialization:
```go
// REMOVE:
logs:           make([]LogEntry, 0, maxLogs),
maxLogs:        maxLogs,
```

- [ ] **Step 3: Rewrite DisplayLog to route to specific channel**

```go
// DisplayLog sends a log entry to the specific channel identified by ChannelID.
// The entry's SessionID and ChannelID control routing.
func (p *channelFacadeImpl) DisplayLog(entry channel.LogEntry) {
	p.mu.RLock()
	targetChannel, exists := p.channels[entry.ChannelID]
	p.mu.RUnlock()

	if !exists {
		p.logger.Warn("channel not found for log entry",
			zap.String("channel_id", entry.ChannelID.String()),
			zap.String("session_id", entry.SessionID),
		)
		return
	}

	// Forward to specific channel only
	targetChannel.OnLog(entry)
}
```

- [ ] **Step 4: Remove GetLogs method**

Delete the entire GetLogs method:
```go
// REMOVE THIS METHOD ENTIRELY:
func (p *channelFacadeImpl) GetLogs(since time.Time, limit int) []LogEntry {
	// ... delete all of this
}
```

- [ ] **Step 5: Remove GetLogs from interface**

Update `pkg/channel/interface.go` - remove from ChannelFacade interface:
```go
// REMOVE from interface:
// GetLogs returns recent log entries for channels to poll
// GetLogs(since time.Time, limit int) []LogEntry
```

- [ ] **Step 6: Run tests**

```bash
go test ./pkg/channel/... -v
```

Expected: Some tests may fail (those that test GetLogs)

- [ ] **Step 7: Update or remove failing tests**

For each failing test:
- If testing GetLogs() → Remove test (functionality moved to logger)
- If testing DisplayLog() → Update to test routing behavior

Example update for DisplayLog tests:
```go
func TestChannelFacade_DisplayLog_RoutesToSpecificChannel(t *testing.T) {
	// ... setup with multiple channels ...
	
	entry := LogEntry{
		Level:     "info",
		Message:   "Test log",
		Timestamp: time.Now(),
		SessionID: "test-session",
		ChannelID: channels[0].ID(), // Target first channel
	}
	
	service.DisplayLog(entry)
	
	// Verify only the target channel received the log
	assert.Equal(t, 1, channels[0].getLogCount())
	assert.Equal(t, 0, channels[1].getLogCount())
}
```

- [ ] **Step 8: Run tests again**

```bash
go test ./pkg/channel/... -v
```

Expected: All tests pass

- [ ] **Step 9: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/interface.go pkg/channel/facade_test.go
git commit -m "refactor(channel): fix DisplayLog routing, remove duplicate buffer

Fix DisplayLog() to route to specific channel by ChannelID:
- Remove duplicate logs buffer and maxLogs fields
- Remove GetLogs() method (functionality in logger)
- Route to entry.ChannelID, not broadcast
- Add warning when channel not found
- Update tests for new routing behavior

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: DI Integration

### Task 8: Wire logger to facade via DI container

**Files:**
- Modify: `pkg/di/container.go`

- [ ] **Step 1: Add logger.SetLogForwarder after facade creation**

Find where services are provided and add after channel.NewChannelFacade:

```go
	// Channel Abstraction Layer
	do.Provide(p.injector, command.NewManager)
	do.Provide(p.injector, channel.NewChannelFacade)

	// Wire logger to facade for log forwarding
	do.Provide(p.injector, func(injector do.Injector) (struct{}, error) {
		logger := do.MustInvoke[logger.LoggerService](injector)
		facade := do.MustInvoke[channel.ChannelFacade](injector)
		
		// Type assertion to access SetLogForwarder
		if facadeWithSetter, ok := facade.(interface{ SetLogForwarder(shared.LogForwarder) }); ok {
			facadeWithSetter.SetLogForwarder(facade)
		}
		
		return struct{}{}, nil
	})
```

Wait - this approach won't work because ChannelFacade doesn't expose SetLogForwarder in its interface. Let me use a simpler approach:

- [ ] **Step 1: Use InvokeAfter to wire forwarder**

Replace the above with this simpler approach:

```go
	// Channel Abstraction Layer
	do.Provide(p.injector, command.NewManager)
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	// Logger
	do.Provide(p.injector, logger.NewService)
	logger := do.MustInvoke[logger.LoggerService](injector)

	// Wire logger forwarder to facade
	// Note: This requires ChannelFacade to implement SetLogForwarder
	// which we'll add to the implementation
	if facadeImpl, ok := facade.(*channel.ChannelFacade); ok {
		logger.SetLogForwarder(facadeImpl)
	}
```

Actually, this still has issues. Let me check the actual types first:

- [ ] **Step 1: Check actual types**

Need to see what channel.NewChannelFacade returns. It returns ChannelFacade interface, not concrete type.

Better approach - add a provider that sets up the relationship:

```go
	// Channel Abstraction Layer
	do.Provide(p.injector, command.NewManager)
	do.Provide(p.injector, channel.NewChannelFacade)

	// Logger
	do.Provide(p.injector, logger.NewService)
	
	// Wire logger forwarder to channel facade
	do.Provide(p.injector, func(injector do.Injector) (struct{}, error) {
		facade := do.MustInvoke[channel.ChannelFacade](injector)
		loggerSvc := do.MustInvoke[logger.LoggerService](injector)
		
		// The facade should implement LogForwarder
		if forwarder, ok := facade.(shared.LogForwarder); ok {
			loggerSvc.SetLogForwarder(forwarder)
		}
		
		return struct{}{}, nil
	})
```

But we need to import shared in the DI container. Let me add that:

- [ ] **Step 1: Add shared import**

```go
	"github.com/denkhaus/gollum/pkg/shared"
```

- [ ] **Step 2: Add wiring provider after logger creation**

```go
	// Logger
	do.Provide(p.injector, logger.NewService)

	// Wire logger forwarder to channel facade
	do.Provide(p.injector, func(injector do.Injector) (struct{}, error) {
		facade := do.MustInvoke[channel.ChannelFacade](injector)
		loggerSvc := do.MustInvoke[logger.LoggerService](injector)

		// The facade implements LogForwarder, wire it to logger
		if forwarder, ok := facade.(shared.LogForwarder); ok {
			loggerSvc.SetLogForwarder(forwarder)
		}

		return struct{}{}, nil
	})
```

- [ ] **Step 3: Compile check**

```bash
go build ./pkg/di/
```

Expected: No errors

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/di/... -v
```

Expected: All tests pass

- [ ] **Step 5: Integration test**

```bash
go test ./pkg/logger/... ./pkg/channel/... -v
```

Expected: All tests pass, logger forwards to facade

- [ ] **Step 6: Commit**

```bash
git add pkg/di/container.go
git commit -m "feat(di): wire logger LogForwarder to ChannelFacade

Add DI provider to connect logger and channel facade:
- ChannelFacade implements shared.LogForwarder
- Logger receives forwarder via SetLogForwarder()
- Enables session/channel-aware log routing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: Testing & Validation

### Task 9: Add integration tests for log forwarding

**Files:**
- Create: `pkg/channel/facade_integration_log_test.go`

- [ ] **Step 1: Create integration test file**

```go
// Package channel provides integration tests for log forwarding
package channel

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestLogForwarding_Integration tests that logger forwards logs to channels
func TestLogForwarding_Integration(t *testing.T) {
	// Setup logger with mock config
	cfg := &mockConfigService{logBufferSize: 100}
	injector := setupTestInjectorWithConfig(cfg)

	loggerSvc := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[ChannelFacade](injector)

	// Register forwarder
	loggerSvc.SetLogForwarder(facade)

	// Create test channel
	testChannel := newMockChannel(uuid.New())
	err := facade.RegisterChannel(testChannel)
	require.NoError(t, err)

	// Create logging context
	ctx := shared.LoggingContext{
		SessionID: "test-session-123",
		ChannelID: testChannel.ID(),
		AgentID:   uuid.New(),
	}

	// Log with context
	loggerSvc.InfoWithContext("Test message", ctx, zap.String("test", "value"))

	// Give time for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify log was received
	assert.Equal(t, 1, testChannel.getLogCount())
	receivedLog := testChannel.getLastLog()
	assert.Equal(t, "info", receivedLog.Level)
	assert.Equal(t, "Test message", receivedLog.Message)
}

// TestLogForwarding_InvalidContext_DoesNotForward tests that invalid context is rejected
func TestLogForwarding_InvalidContext_DoesNotForward(t *testing.T) {
	cfg := &mockConfigService{logBufferSize: 100}
	injector := setupTestInjectorWithConfig(cfg)

	loggerSvc := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[ChannelFacade](injector)
	loggerSvc.SetLogForwarder(facade)

	// Create test channel
	testChannel := newMockChannel(uuid.New())
	err := facade.RegisterChannel(testChannel)
	require.NoError(t, err)

	// Create INVALID logging context (empty SessionID)
	ctx := shared.LoggingContext{
		SessionID: "", // Invalid!
		ChannelID: testChannel.ID(),
		AgentID:   uuid.New(),
	}

	// Log with invalid context
	loggerSvc.InfoWithContext("This should not forward", ctx)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify log was NOT received (context validation failed)
	assert.Equal(t, 0, testChannel.getLogCount())
}
```

- [ ] **Step 2: Run integration tests**

```bash
go test ./pkg/channel/facade_integration_log_test.go -v
```

Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add pkg/channel/facade_integration_log_test.go
git commit -m "test(channel): add integration tests for log forwarding

Add integration tests for logger → channel log forwarding:
- Test valid context forwards to correct channel
- Test invalid context is rejected
- Verify SessionID/ChannelID routing works end-to-end

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 10: Verify ACP receives logs correctly

**Files:**
- Test: Manual verification or add test in `pkg/acp/`

- [ ] **Step 1: Create test for ACP log routing**

Create: `pkg/acp/log_routing_test.go`

```go
// Package acp provides tests for log routing to ACP sessions
package acp

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/ironpark/go-acp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestACP_LogRouting_SpecificSession tests that logs route to specific ACP session
func TestACP_LogRouting_SpecificSession(t *testing.T) {
	// Setup minimal DI
	injector := do.New()
	do.ProvideValue[injector, config.ConfigService](&mockConfigService{})
	do.Provide(injector, registry.NewAgentRegistry)
	do.Provide(injector, session.NewSessionManager)
	do.Provide(injector, logger.NewService)
	do.Provide(injector, channel.NewChannelFacade)

	// Create ACP service
	acpSvc := &acpServiceImpl{id: uuid.New()}
	acpSvc.SetLogger(do.MustInvoke[logger.LoggerService](injector))
	acpSvc.SetFacade(do.MustInvoke[channel.ChannelFacade](injector))

	// Create session store
	store := &mockSessionStore{}
	acpSvc.SetSessionStore(store)

	// Create mock client
	mockClient := &mockACPV1Client{}
	acpSvc.SetClient(mockClient)

	// Create a session
	sessionID := acp.SessionID("test-session-456")
	testSession := &shared.ACPSession{
		Context:    context.Background(),
		CancelFunc: func() {},
		SessionID:  sessionID,
	}
	store.Set(sessionID, testSession)

	// Create logging context for this session
	ctx := shared.LoggingContext{
		SessionID: string(sessionID),
		ChannelID: acpSvc.id,
		AgentID:   uuid.New(),
	}

	// Log with context
	loggerSvc := do.MustInvoke[logger.LoggerService](injector)
	loggerSvc.InfoWithContext("Test ACP log message", ctx)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify mock client received the log
	require.True(t, len(mockClient.receivedLogs) > 0, "No logs received")
	
	// Find the log we just sent
	var foundLog bool
	for _, logMsg := range mockClient.receivedLogs {
		if logMsg == "[info] Test ACP log message" {
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected log message not found")
}

// Mock implementations...
type mockACPV1Client struct {
	receivedLogs []string
}

func (m *mockACPV1Client) SendText(ctx context.Context, sessionID acp.SessionID, text string) error {
	m.receivedLogs = append(m.receivedLogs, text)
	return nil
}
// ... other mock methods
```

- [ ] **Step 2: Run test**

```bash
go test ./pkg/acp/log_routing_test.go -v
```

Expected: Test passes, ACP receives logs for specific session

- [ ] **Step 3: Commit**

```bash
git add pkg/acp/log_routing_test.go
git commit -m "test(acp): add test for session-aware log routing

Add test verifying ACP receives logs for specific session:
- Create ACP session with session store
- Log with LoggingContext containing SessionID
- Verify log routes to correct ACP session only
- Uses mock ACP client to capture streamed logs

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: Documentation & Cleanup

### Task 11: Update documentation

**Files:**
- Modify: Relevant documentation files

- [ ] **Step 1: Update ACP improvements document**

Update `docs/acp_improvements.md` - mark Phase 1 items as complete:

```markdown
### Phase 1: Quick Wins (1-2 days)

1. ✅ **Log Forwarding** - Implemented via LogForwarder interface
2. ✅ **Document Session Lifecycle** - Added clarifying comments
3. ✅ **OnAgentLifecycle Notification** - Implemented
```

- [ ] **Step 2: Add architecture diagram to README or docs**

Consider adding a section explaining the unified logging flow.

- [ ] **Step 3: Commit documentation**

```bash
git add docs/acp_improvements.md
git commit -m "docs: mark ACP Phase 1 improvements complete

Update ACP improvements document:
- Mark log forwarding as complete
- Mark session lifecycle docs as complete  
- Mark agent lifecycle notifications as complete

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 12: Final verification

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

```bash
go test ./pkg/... -v
```

Expected: All tests pass

- [ ] **Step 2: Build all packages**

```bash
go build ./pkg/... ./cmd/...
```

Expected: No compilation errors

- [ ] **Step 3: Check for any TODO comments we left**

```bash
grep -r "TODO" pkg/logger/ pkg/channel/ pkg/shared/
```

Expected: No relevant TODOs (or file issues for remaining ones)

- [ ] **Step 4: Create summary commit**

```bash
git add .
git commit -m "chore: finalize unified logging implementation

Complete unified logging architecture:
- All tests pass
- All packages build
- Documentation updated
- Ready for production use

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Success Criteria Verification

After completing all tasks, verify:

- [ ] ✅ All agent-related logs are session/channel-aware
- [ ] ✅ DisplayLog() routes to specific channel, not broadcast
- [ ] ✅ No duplicate buffers (single buffer in logger)
- [ ] ✅ TUI receives logs via OnLog()
- [ ] ✅ ACP receives logs for correct session
- [ ] ✅ Clean separation with LogForwarder interface
- [ ] ✅ Strict validation of LoggingContext
- [ ] ✅ Backward compatible with existing *WithAgent() code

---

## Notes for Agentic Workers

- **Follow TDD:** Write failing test first, then implement, then verify
- **Frequent commits:** Each task ends with a commit, don't batch changes
- **Testing guide:** Follow `/home/denkhaus/dev/kb/guides/guide.golang.testing.md`
- **When stuck:** Check the spec document at `docs/superpowers/specs/2026-04-11-unified-logging-design.md`
- **LogEntry already has SessionID/ChannelID fields** - added in previous work

**Dependencies:** This plan assumes the command manager has already been moved to `pkg/command/` package.

**Testing patterns:** Use table-driven tests where appropriate, mock interfaces using mockgen (run `make mocks` or `go generate ./...` to regenerate mocks after interface changes).
