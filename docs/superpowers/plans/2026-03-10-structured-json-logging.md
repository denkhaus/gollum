# Structured JSON Logging with Agent ID Tracking

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement structured JSON logging for session files with automatic agent ID tracking, enabling traceability of agent actions.

**Architecture:**
1. Replace console encoder with JSON encoder for file logging only (stdout remains human-readable)
2. Add helper methods (`InfoWithAgent`, `ErrorWithAgent`, etc.) that auto-prepend agent_id as a structured field
3. Update `storeInBuffer` to extract agent_id from fields for the in-memory buffer
4. Update tool logging calls to use new helper methods with structured fields

**Tech Stack:**
- Go 1.23+
- zap (uber-go/zap) for structured logging
- google/uuid for UUID handling
- testify for testing

---

## Chunk 1: Core Logger Changes

This chunk implements the JSON encoder for file logging and adds the helper methods for agent-aware logging.

### Task 1: Add JSON Encoder to File Logger

**Files:**
- Modify: `pkg/logger/file.go`

- [ ] **Step 1: Write failing test for JSON log format**

Create `pkg/logger/file_test.go`:

```go
package logger

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// TestJSONLogFormat verifies that file logs are written in JSON format
func TestJSONLogFormat(t *testing.T) {
    // Create temp directory for logs
    tmpDir := t.TempDir()
    sessionID := uuid.New()

    // Create a minimal logger service
    svc := &service{
        config: zap.NewProductionConfig(),
        configService: &mockConfigService{},
    }

    // Enable file logging
    err := svc.EnableFileLogging(tmpDir, sessionID)
    require.NoError(t, err)
    defer svc.CloseFileLogging()

    // Log with agent ID
    agentID := uuid.New()
    svc.InfoWithAgent("Test message", agentID, zap.String("test_field", "test_value"))

    // Flush to ensure write
    err = svc.Flush()
    require.NoError(t, err)

    // Read the log file
    logPath := filepath.Join(tmpDir, sessionID.String()+".log")
    content, err := os.ReadFile(logPath)
    require.NoError(t, err)

    // Verify it's valid JSON
    lines := strings.Split(strings.TrimSpace(string(content)), "\n")
    require.Greater(t, len(lines), 0, "Log file should have at least one line")

    var logEntry map[string]interface{}
    err = json.Unmarshal([]byte(lines[0]), &logEntry)
    require.NoError(t, err, "Log line should be valid JSON")

    // Verify required fields
    assert.Contains(t, logEntry, "timestamp")
    assert.Contains(t, logEntry, "level")
    assert.Contains(t, logEntry, "message")
    assert.Contains(t, logEntry, "agent_id")
    assert.Contains(t, logEntry, "test_field")
    assert.Equal(t, "Test message", logEntry["message"])
    assert.Equal(t, agentID.String(), logEntry["agent_id"])
    assert.Equal(t, "test_value", logEntry["test_field"])
}

// mockConfigService for testing
type mockConfigService struct{}

func (m *mockConfigService) IsDevMode() bool                                      { return false }
func (m *mockConfigService) GetLogLevel() string                                   { return "info" }
func (m *mockConfigService) GetConfig() interface{}                                { return nil }
func (m *mockConfigService) GetLLMConfig(name string) (interface{}, bool)         { return nil, false }
func (m *mockConfigService) GetLLMConfigs() map[string]interface{}                { return nil }
func (m *mockConfigService) GetLoggingConfig() logger.LoggingConfig {
    return logger.LoggingConfig{
        SessionLogEnabled:    false,
        SessionLogBufferSize: 1000,
        MaxSessionLogFiles:   10,
    }
}
func (m *mockConfigService) GetBashConfig() logger.BashConfig { return logger.BashConfig{} }
func (m *mockConfigService) GetAgentConfig() logger.AgentConfig { return logger.AgentConfig{} }
```

Run: `go test -v ./pkg/logger/ -run TestJSONLogFormat`
Expected: FAIL - `InfoWithAgent` method doesn't exist yet

- [ ] **Step 2: Modify file.go to use JSON encoder**

Edit `pkg/logger/file.go`:

Replace the `EnableFileLogging` function's encoder creation (around line 52):

```go
// OLD CODE (remove this):
encoder := newRawModeConsoleEncoder(s.config.EncoderConfig)
fileCore := zapcore.NewCore(encoder, fileWriteSyncer, s.atomicLevel)

// NEW CODE (add this):
// Use JSON encoder for structured logging
encoderConfig := zapcore.EncoderConfig{
    TimeKey:        "timestamp",
    LevelKey:       "level",
    NameKey:        "logger",
    CallerKey:      "caller",
    MessageKey:     "message",
    StacktraceKey:  "stacktrace",
    LineEnding:     zapcore.DefaultLineEnding,
    EncodeLevel:    zapcore.LowercaseLevelEncoder,
    EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
    EncodeDuration: zapcore.SecondsDurationEncoder,
    EncodeCaller:   zapcore.ShortCallerEncoder,
}
encoder := zapcore.NewJSONEncoder(encoderConfig)
fileCore := zapcore.NewCore(encoder, fileWriteSyncer, s.atomicLevel)
```

- [ ] **Step 3: Run test to verify JSON encoder**

Run: `go test -v ./pkg/logger/ -run TestJSONLogFormat`
Expected: FAIL - `InfoWithAgent` method doesn't exist yet (but encoder is now JSON)

- [ ] **Step 4: Commit JSON encoder change**

```bash
git add pkg/logger/file.go pkg/logger/file_test.go
git commit -m "feat(logger): use JSON encoder for file logging

Replace console encoder with JSON encoder for structured session logs.
Stdout logging remains human-readable with console format.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 2: Add Agent-Aware Helper Methods to LoggerService

**Files:**
- Modify: `pkg/logger/logger.go`

- [ ] **Step 1: Update LoggerService interface**

Add these methods to the `LoggerService` interface (after line 44):

```go
// InfoWithAgent logs an info message with agent ID
InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
// ErrorWithAgent logs an error message with agent ID
ErrorWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
// DebugWithAgent logs a debug message with agent ID
DebugWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
// WarnWithAgent logs a warning message with agent ID
WarnWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
```

- [ ] **Step 2: Implement helper methods in service struct**

Add implementations after the `Warnf` method (around line 198):

```go
// InfoWithAgent logs an info message with agent ID included as a structured field.
func (s *service) InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    allFields := append([]zap.Field{zap.String("agent_id", agentID.String())}, fields...)
    s.Info(msg, allFields...)
}

// ErrorWithAgent logs an error message with agent ID included as a structured field.
func (s *service) ErrorWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    allFields := append([]zap.Field{zap.String("agent_id", agentID.String())}, fields...)
    s.Error(msg, allFields...)
}

// DebugWithAgent logs a debug message with agent ID included as a structured field.
func (s *service) DebugWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    allFields := append([]zap.Field{zap.String("agent_id", agentID.String())}, fields...)
    s.Debug(msg, allFields...)
}

// WarnWithAgent logs a warning message with agent ID included as a structured field.
func (s *service) WarnWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    allFields := append([]zap.Field{zap.String("agent_id", agentID.String())}, fields...)
    s.Warn(msg, allFields...)
}
```

- [ ] **Step 3: Run test to verify helper methods work**

Run: `go test -v ./pkg/logger/ -run TestJSONLogFormat`
Expected: PASS - Helper methods now exist and JSON encoder produces valid output

- [ ] **Step 4: Commit helper methods**

```bash
git add pkg/logger/logger.go
git commit -m "feat(logger): add agent-aware logging helper methods

Add InfoWithAgent, ErrorWithAgent, DebugWithAgent, and WarnWithAgent
methods that automatically include agent_id as a structured field.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 3: Update storeInBuffer to Extract Agent ID

**Files:**
- Modify: `pkg/logger/logger.go`

- [ ] **Step 1: Update storeInBuffer to extract agent_id**

Replace the existing `storeInBuffer` method (around line 200) with:

```go
// storeInBuffer stores a log entry in the session buffer.
func (s *service) storeInBuffer(level string, msg string, fields []zap.Field) {
    entry := LogEntry{
        Timestamp: time.Now(),
        Level:     level,
        Message:   msg,
        Fields:    zapFieldsToMap(fields),
    }

    // Extract agent_id from fields if present
    for _, field := range fields {
        if field.Key == "agent_id" {
            if agentStr, ok := field.Interface.(string); ok {
                entry.AgentID, _ = uuid.Parse(agentStr)
            }
            break
        }
    }

    s.logBuffer.add(entry)
}
```

- [ ] **Step 2: Write test for agent ID extraction**

Add to `pkg/logger/buffer_test.go`:

```go
// TestLogEntry_AgentIDExtraction tests that agent_id is extracted from zap fields
func TestLogEntry_AgentIDExtraction(t *testing.T) {
    t.Run("Agent ID extracted from fields", func(t *testing.T) {
        buf := newLogBuffer(10, true)
        svc := &service{logBuffer: buf}

        agentID := uuid.New()
        svc.storeInBuffer("info", "test message", []zap.Field{
            zap.String("agent_id", agentID.String()),
            zap.String("other_field", "value"),
        })

        entries := buf.getEntries(LogFilter{})
        require.Equal(t, 1, len(entries))
        assert.Equal(t, agentID, entries[0].AgentID)
    })

    t.Run("Nil agent ID when not in fields", func(t *testing.T) {
        buf := newLogBuffer(10, true)
        svc := &service{logBuffer: buf}

        svc.storeInBuffer("info", "test message", []zap.Field{
            zap.String("other_field", "value"),
        })

        entries := buf.getEntries(LogFilter{})
        require.Equal(t, 1, len(entries))
        assert.Equal(t, uuid.Nil, entries[0].AgentID)
    })

    t.Run("Invalid UUID handled gracefully", func(t *testing.T) {
        buf := newLogBuffer(10, true)
        svc := &service{logBuffer: buf}

        svc.storeInBuffer("info", "test message", []zap.Field{
            zap.String("agent_id", "not-a-uuid"),
        })

        entries := buf.getEntries(LogFilter{})
        require.Equal(t, 1, len(entries))
        assert.Equal(t, uuid.Nil, entries[0].AgentID)
    })
}
```

- [ ] **Step 3: Run test to verify agent ID extraction**

Run: `go test -v ./pkg/logger/ -run TestLogEntry_AgentIDExtraction`
Expected: PASS

- [ ] **Step 4: Commit storeInBuffer update**

```bash
git add pkg/logger/logger.go pkg/logger/buffer_test.go
git commit -m "feat(logger): extract agent_id from fields for buffer

Update storeInBuffer to extract agent_id from zap fields so the
in-memory log buffer also tracks which agent generated each log entry.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Tool Logging Updates

This chunk updates tool implementations to use the new agent-aware logging methods with structured fields.

### Task 4: Update Bash Tool Logging

**Files:**
- Modify: `pkg/tools/bash.go`

- [ ] **Step 1: Update bash tool to use InfoWithAgent**

Replace line 134:
```go
// OLD:
t.logService.Infof("Executing bash command: %s", command)

// NEW:
t.logService.InfoWithAgent("Executing bash command", t.agentID, zap.String("command", command))
```

- [ ] **Step 2: Update timeout warning (line 157)**

```go
// OLD:
t.logService.Warnf("Command timed out after %.2fs: %s", timeout.Seconds(), command)

// NEW:
t.logService.WarnWithAgent("Command timed out", t.agentID,
    zap.Float64("timeout_seconds", timeout.Seconds()),
    zap.String("command", command))
```

- [ ] **Step 3: Update command failed error (line 162)**

```go
// OLD:
t.logService.Errorf("Command failed: %s - %v", command, err)

// NEW:
t.logService.ErrorWithAgent("Command failed", t.agentID,
    zap.String("command", command),
    zap.Error(err))
```

- [ ] **Step 4: Update command succeeded log (line 170)**

```go
// OLD:
t.logService.Infof("Command succeeded in %.2fs: %s", duration.Seconds(), command)

// NEW:
t.logService.InfoWithAgent("Command succeeded", t.agentID,
    zap.Float64("duration_seconds", duration.Seconds()),
    zap.String("command", command))
```

- [ ] **Step 5: Update file changes detection warning (line 185)**

```go
// OLD:
t.logService.Warn("Failed to detect file changes", zap.Error(detectErr))

// NEW:
t.logService.WarnWithAgent("Failed to detect file changes", t.agentID, zap.Error(detectErr))
```

- [ ] **Step 6: Update bash file modification log (line 189)**

```go
// OLD:
t.logService.Info("Bash command modified files",
    zap.Int("count", len(changes)),
    zap.String("command", command))

// NEW:
t.logService.InfoWithAgent("Bash command modified files", t.agentID,
    zap.Int("count", len(changes)),
    zap.String("command", command))
```

- [ ] **Step 7: Update diff content warnings (lines 200, 217)**

```go
// OLD (line 200):
t.logService.Warnf("Failed to get current content for diff: %s - %v", change.Path, err)

// NEW:
t.logService.WarnWithAgent("Failed to get current content for diff", t.agentID,
    zap.String("path", change.Path),
    zap.Error(err))

// OLD (line 217):
t.logService.Warnf("Failed to generate diff for %s: %v", change.Path, err)

// NEW:
t.logService.WarnWithAgent("Failed to generate diff", t.agentID,
    zap.String("path", change.Path),
    zap.Error(err))
```

- [ ] **Step 8: Run bash tool tests**

Run: `go test -v ./pkg/tools/ -run Bash`
Expected: PASS

- [ ] **Step 9: Commit bash tool logging updates**

```bash
git add pkg/tools/bash.go
git commit -m "feat(tools): use structured logging with agent ID in bash tool

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods that include agent_id as a structured field for better
traceability in session logs.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 5: Update Spawn Agent Tool Logging

**Files:**
- Modify: `pkg/tools/spawn_agent.go`

- [ ] **Step 1: Update spawn_agent.go Infof calls to InfoWithAgent**

Find and replace these logging calls:

Line 161:
```go
// OLD:
t.logService.Infof("Spawning subagent: role=%s description=%s background=%v share_context=%v", role, description, runInBackground, shareContext)

// NEW:
t.logService.InfoWithAgent("Spawning subagent", t.agentID,
    zap.String("role", role),
    zap.String("description", description),
    zap.Bool("background", runInBackground),
    zap.Bool("share_context", shareContext))
```

Line 166:
```go
// OLD:
t.logService.Errorf("Failed to get subagent prompt: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to get subagent prompt", t.agentID, zap.Error(err))
```

Line 175:
```go
// OLD:
t.logService.Debugf("Inheriting LLM config from parent agent %s", t.senderID)

// NEW:
t.logService.DebugWithAgent("Inheriting LLM config from parent", t.agentID,
    zap.String("parent_agent_id", t.senderID.String()))
```

Line 181:
```go
// OLD:
t.logService.Debugf("Using default LLM config (no parent agent found)")

// NEW:
t.logService.DebugWithAgent("Using default LLM config", t.agentID)
```

Line 188:
```go
// OLD:
t.logService.Warnf("Failed to get message history from parent agent: %v", err)

// NEW:
t.logService.WarnWithAgent("Failed to get message history from parent", t.agentID, zap.Error(err))
```

Line 210:
```go
// OLD:
t.logService.Errorf("Failed to create subagent: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to create subagent", t.agentID, zap.Error(err))
```

Line 214:
```go
// OLD:
t.logService.Infof("Created subagent %s (role=%s)", subagent.GetID(), role)

// NEW:
t.logService.InfoWithAgent("Created subagent", t.agentID,
    zap.String("subagent_id", subagent.GetID().String()),
    zap.String("role", role))
```

Line 224:
```go
// OLD:
t.logService.Errorf("Failed to store agent result: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to store agent result", t.agentID, zap.Error(err))
```

Line 239:
```go
// OLD:
t.logService.Errorf("Failed to register background agent: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to register background agent", t.agentID, zap.Error(err))
```

Line 243:
```go
// OLD:
t.logService.Infof("Starting background agent %s", subagent.GetID())

// NEW:
t.logService.InfoWithAgent("Starting background agent", t.agentID,
    zap.String("subagent_id", subagent.GetID().String()))
```

Line 250:
```go
// OLD:
t.logService.Errorf("Failed to register synchronous agent: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to register synchronous agent", t.agentID, zap.Error(err))
```

Line 254:
```go
// OLD:
t.logService.Infof("Executing synchronous agent %s", subagent.GetID())

// NEW:
t.logService.InfoWithAgent("Executing synchronous agent", t.agentID,
    zap.String("subagent_id", subagent.GetID().String()))
```

Line 257:
```go
// OLD:
t.logService.Errorf("Agent execution failed: %v", err)

// NEW:
t.logService.ErrorWithAgent("Agent execution failed", t.agentID, zap.Error(err))
```

Line 261:
```go
// OLD:
t.logService.Infof("Agent %s completed successfully", subagent.GetID())

// NEW:
t.logService.InfoWithAgent("Agent completed successfully", t.agentID,
    zap.String("subagent_id", subagent.GetID().String()))
```

- [ ] **Step 2: Run spawn_agent tests**

Run: `go test -v ./pkg/tools/ -run SpawnAgent`
Expected: PASS

- [ ] **Step 3: Commit spawn_agent logging updates**

```bash
git add pkg/tools/spawn_agent.go
git commit -m "feat(tools): use structured logging with agent ID in spawn_agent

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods for better traceability of agent lifecycle events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 6: Update Remove Agent Tool Logging

**Files:**
- Modify: `pkg/tools/remove_agent.go`

- [ ] **Step 1: Update remove_agent.go logging calls**

Line 90:
```go
// OLD:
t.logService.Debugf("RemoveAgent: invalid agent_id type from sender %s", t.senderID)

// NEW:
t.logService.DebugWithAgent("RemoveAgent: invalid agent_id type", t.agentID,
    zap.String("sender_id", t.senderID.String()))
```

Line 99:
```go
// OLD:
t.logService.Debugf("RemoveAgent: invalid UUID format '%s' from sender %s", agentIDStr, t.senderID)

// NEW:
t.logService.DebugWithAgent("RemoveAgent: invalid UUID format", t.agentID,
    zap.String("agent_id_str", agentIDStr),
    zap.String("sender_id", t.senderID.String()))
```

Line 106:
```go
// OLD:
t.logService.Debugf("RemoveAgent: attempting to remove agent %s by sender %s", agentID, t.senderID)

// NEW:
t.logService.DebugWithAgent("RemoveAgent: attempting to remove agent", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.String("sender_id", t.senderID.String()))
```

Line 110:
```go
// OLD:
t.logService.Infof("RemoveAgent: self-removal attempted by agent %s", t.senderID)

// NEW:
t.logService.InfoWithAgent("RemoveAgent: self-removal attempted", t.agentID,
    zap.String("sender_id", t.senderID.String()))
```

Line 120:
```go
// OLD:
t.logService.Infof("RemoveAgent: target agent %s not found", agentID)

// NEW:
t.logService.InfoWithAgent("RemoveAgent: target agent not found", t.agentID,
    zap.String("target_agent_id", agentID.String()))
```

Line 144:
```go
// OLD:
t.logService.Warnf("RemoveAgent: permission denied for sender %s to remove agent %s (not parent)", t.senderID, agentID)

// NEW:
t.logService.WarnWithAgent("RemoveAgent: permission denied", t.agentID,
    zap.String("sender_id", t.senderID.String()),
    zap.String("target_agent_id", agentID.String()),
    zap.String("reason", "not_parent"))
```

Line 161:
```go
// OLD:
t.logService.Debugf("RemoveAgent: agent %s has %d children, force=%v", agentID, childCount, force)

// NEW:
t.logService.DebugWithAgent("RemoveAgent: agent has children", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Int("child_count", childCount),
    zap.Bool("force", force))
```

Line 165:
```go
// OLD:
t.logService.Infof("RemoveAgent: agent %s has %d children but force=false", agentID, childCount)

// NEW:
t.logService.InfoWithAgent("RemoveAgent: agent has children but force=false", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Int("child_count", childCount))
```

Line 174:
```go
// OLD:
t.logService.Infof("RemoveAgent: removing agent %s with %d descendants by sender %s", agentID, childCount, t.senderID)

// NEW:
t.logService.InfoWithAgent("RemoveAgent: removing agent with descendants", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Int("descendant_count", childCount),
    zap.String("sender_id", t.senderID.String()))
```

Line 179:
```go
// OLD:
t.logService.Errorf("RemoveAgent: failed to cleanup agent %s: %v", agentID, err)

// NEW:
t.logService.ErrorWithAgent("RemoveAgent: failed to cleanup agent", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Error(err))
```

Line 186:
```go
// OLD:
t.logService.Infof("RemoveAgent: successfully removed agent %s and all %d descendants", agentID, childCount)

// NEW:
t.logService.InfoWithAgent("RemoveAgent: successfully removed agent", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Int("descendant_count", childCount))
```

- [ ] **Step 2: Run remove_agent tests**

Run: `go test -v ./pkg/tools/ -run RemoveAgent`
Expected: PASS

- [ ] **Step 3: Commit remove_agent logging updates**

```bash
git add pkg/tools/remove_agent.go
git commit -m "feat(tools): use structured logging with agent ID in remove_agent

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods for better traceability of agent removal events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 7: Update Invoke Skill Tool Logging

**Files:**
- Modify: `pkg/tools/invoke_skill.go`

- [ ] **Step 1: Update invoke_skill.go logging calls**

Line 143:
```go
// OLD:
t.logService.Errorf("Skill not found: %s: %v", skillName, err)

// NEW:
t.logService.ErrorWithAgent("Skill not found", t.agentID,
    zap.String("skill_name", skillName),
    zap.Error(err))
```

Line 197:
```go
// OLD:
t.logService.Warnf("Skill invocation blocked by hook: %v", beforeResult.Error)

// NEW:
t.logService.WarnWithAgent("Skill invocation blocked by hook", t.agentID,
    zap.String("error", beforeResult.Error))
```

Line 201:
```go
// OLD:
t.logService.Infof("Invoking skill: name=%s context_mode=%s model=%s", skill.Name, contextMode, modelName)

// NEW:
t.logService.InfoWithAgent("Invoking skill", t.agentID,
    zap.String("skill_name", skill.Name),
    zap.String("context_mode", string(contextMode)),
    zap.String("model", modelName))
```

Line 219:
```go
// OLD:
t.logService.Warnf("Failed to get message history from parent agent: %v", err)

// NEW:
t.logService.WarnWithAgent("Failed to get message history from parent", t.agentID, zap.Error(err))
```

Line 242:
```go
// OLD:
t.logService.Debugf("Skill has tool restrictions: tools=%v scope=%s", skill.Tools, skill.ToolScope)

// NEW:
t.logService.DebugWithAgent("Skill has tool restrictions", t.agentID,
    zap.Any("tools", skill.Tools),
    zap.String("scope", string(skill.ToolScope)))
```

Line 248:
```go
// OLD:
t.logService.Errorf("Failed to create skill subagent: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to create skill subagent", t.agentID, zap.Error(err))
```

Line 255:
```go
// OLD:
t.logService.Infof("Created skill subagent %s (skill=%s)", subagent.GetID(), skill.Name)

// NEW:
t.logService.InfoWithAgent("Created skill subagent", t.agentID,
    zap.String("subagent_id", subagent.GetID().String()),
    zap.String("skill_name", skill.Name))
```

Line 259:
```go
// OLD:
t.logService.Errorf("Failed to register skill subagent: %v", err)

// NEW:
t.logService.ErrorWithAgent("Failed to register skill subagent", t.agentID, zap.Error(err))
```

Line 269:
```go
// OLD:
t.logService.Errorf("Skill execution failed: %v", err)

// NEW:
t.logService.ErrorWithAgent("Skill execution failed", t.agentID, zap.Error(err))
```

Line 276:
```go
// OLD:
t.logService.Infof("Skill %s completed successfully", skill.Name)

// NEW:
t.logService.InfoWithAgent("Skill completed successfully", t.agentID,
    zap.String("skill_name", skill.Name))
```

- [ ] **Step 2: Run invoke_skill tests**

Run: `go test -v ./pkg/tools/ -run InvokeSkill`
Expected: PASS

- [ ] **Step 3: Commit invoke_skill logging updates**

```bash
git add pkg/tools/invoke_skill.go
git commit -m "feat(tools): use structured logging with agent ID in invoke_skill

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods for better traceability of skill invocation events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 8: Update Resume Agent Tool Logging

**Files:**
- Modify: `pkg/tools/resume_agent.go`

- [ ] **Step 1: Update resume_agent.go logging calls**

Line 128:
```go
// OLD:
t.logService.Warnf("Permission denied: agent %s attempted to resume agent %s (not direct parent)", t.senderID, agentID)

// NEW:
t.logService.WarnWithAgent("Permission denied: not direct parent", t.agentID,
    zap.String("sender_id", t.senderID.String()),
    zap.String("target_agent_id", agentID.String()))
```

Line 135:
```go
// OLD:
t.logService.Infof("Resuming agent %s from sender %s (background=%v)", agentID, t.senderID, runInBackground)

// NEW:
t.logService.InfoWithAgent("Resuming agent", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.String("sender_id", t.senderID.String()),
    zap.Bool("background", runInBackground))
```

Line 148:
```go
// OLD:
t.logService.Infof("Found agent %s (role=%s)", agentID, agentConfig.Role)

// NEW:
t.logService.InfoWithAgent("Found agent to resume", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.String("role", agentConfig.Role))
```

Line 152:
```go
// OLD:
t.logService.Warnf("Failed to delete previous agent result for %s: %v", agentID, err)

// NEW:
t.logService.WarnWithAgent("Failed to delete previous agent result", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Error(err))
```

Line 164:
```go
// OLD:
t.logService.Warnf("Failed to store cancel function for agent %s: %v", agentID, err)

// NEW:
t.logService.WarnWithAgent("Failed to store cancel function", t.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Error(err))
```

Line 168:
```go
// OLD:
t.logService.Infof("Starting background execution for agent %s", agentID)

// NEW:
t.logService.InfoWithAgent("Starting background execution for agent", t.agentID,
    zap.String("target_agent_id", agentID.String()))
```

Line 174:
```go
// OLD:
t.logService.Infof("Executing agent %s synchronously", agentID)

// NEW:
t.logService.InfoWithAgent("Executing agent synchronously", t.agentID,
    zap.String("target_agent_id", agentID.String()))
```

Line 177:
```go
// OLD:
t.logService.Errorf("Agent execution failed: %v", err)

// NEW:
t.logService.ErrorWithAgent("Agent execution failed", t.agentID, zap.Error(err))
```

Line 181:
```go
// OLD:
t.logService.Infof("Agent %s completed successfully", agentID)

// NEW:
t.logService.InfoWithAgent("Agent completed successfully", t.agentID,
    zap.String("target_agent_id", agentID.String()))
```

- [ ] **Step 2: Run resume_agent tests**

Run: `go test -v ./pkg/tools/ -run ResumeAgent`
Expected: PASS

- [ ] **Step 3: Commit resume_agent logging updates**

```bash
git add pkg/tools/resume_agent.go
git commit -m "feat(tools): use structured logging with agent ID in resume_agent

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods for better traceability of agent resume events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 9: Update List Agents Tool Logging

**Files:**
- Modify: `pkg/tools/list_agents.go`

- [ ] **Step 1: Update list_agents.go Infof calls to InfoWithAgent**

Line 125:
```go
// OLD:
t.logService.Infof("Agent %s listing related agents (recursive=%v, tree=%v)", t.senderID, recursive, tree)

// NEW:
t.logService.InfoWithAgent("Listing related agents", t.agentID,
    zap.String("sender_id", t.senderID.String()),
    zap.Bool("recursive", recursive),
    zap.Bool("tree", tree))
```

Line 157:
```go
// OLD:
t.logService.Debugf("Found parent agent: %s (role: %s, description: %s)", parentConfig.ID, parentConfig.Role, parentConfig.Description)

// NEW:
t.logService.DebugWithAgent("Found parent agent", t.agentID,
    zap.String("parent_id", parentConfig.ID.String()),
    zap.String("role", parentConfig.Role),
    zap.String("description", parentConfig.Description))
```

Line 176:
```go
// OLD:
t.logService.Debugf("Found descendant: %s (depth: %d, role: %s)", desc.config.ID, desc.depth, desc.config.Role)

// NEW:
t.logService.DebugWithAgent("Found descendant", t.agentID,
    zap.String("descendant_id", desc.config.ID.String()),
    zap.Int("depth", desc.depth),
    zap.String("role", desc.config.Role))
```

Line 192:
```go
// OLD:
t.logService.Debugf("Found subagent: %s (role: %s, description: %s)", config.ID, config.Role, config.Description)

// NEW:
t.logService.DebugWithAgent("Found subagent", t.agentID,
    zap.String("subagent_id", config.ID.String()),
    zap.String("role", config.Role),
    zap.String("description", config.Description))
```

Line 205:
```go
// OLD:
t.logService.Infof("Agent %s found 1 parent and %d descendants", t.senderID, descendantCount)

// NEW:
t.logService.InfoWithAgent("Found parent and descendants", t.agentID,
    zap.String("sender_id", t.senderID.String()),
    zap.Int("descendant_count", descendantCount))
```

Line 208:
```go
// OLD:
t.logService.Infof("Agent %s found 1 parent (no descendants)", t.senderID)

// NEW:
t.logService.InfoWithAgent("Found parent only", t.agentID,
    zap.String("sender_id", t.senderID.String()))
```

Line 215:
```go
// OLD:
t.logService.Infof("Agent %s found %d descendants (no parent)", t.senderID, descendantCount)

// NEW:
t.logService.InfoWithAgent("Found descendants only", t.agentID,
    zap.String("sender_id", t.senderID.String()),
    zap.Int("descendant_count", descendantCount))
```

Line 218:
```go
// OLD:
t.logService.Infof("Agent %s has no related agents", t.senderID)

// NEW:
t.logService.InfoWithAgent("No related agents found", t.agentID,
    zap.String("sender_id", t.senderID.String()))
```

- [ ] **Step 2: Run list_agents tests**

Run: `go test -v ./pkg/tools/ -run ListAgents`
Expected: PASS

- [ ] **Step 3: Commit list_agents logging updates**

```bash
git add pkg/tools/list_agents.go
git commit -m "feat(tools): use structured logging with agent ID in list_agents

Replace formatted logging calls with InfoWithAgent methods for
better traceability of agent listing events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 10: Update Glob Tool Logging

**Files:**
- Modify: `pkg/tools/glob.go`

- [ ] **Step 1: Update glob.go logging calls (remove [Agent %s] prefix, use structured fields)**

Note: glob.go already includes agent ID in messages using `[Agent %s]` format. We'll convert to structured fields.

Line 89:
```go
// OLD:
t.logService.Errorf("[Agent %s] Glob pattern validation failed: pattern is required and must be non-empty", t.agentID)

// NEW:
t.logService.ErrorWithAgent("Glob pattern validation failed", t.agentID,
    zap.String("reason", "pattern_is_required"))
```

Line 105:
```go
// OLD:
t.logService.Errorf("[Agent %s] Failed to resolve absolute path for '%s': %v", t.agentID, searchPath, err)

// NEW:
t.logService.ErrorWithAgent("Failed to resolve absolute path", t.agentID,
    zap.String("search_path", searchPath),
    zap.Error(err))
```

Line 112:
```go
// OLD:
t.logService.Infof("[Agent %s] Starting glob search: pattern='%s' path='%s'", t.agentID, pattern, searchPath)

// NEW:
t.logService.InfoWithAgent("Starting glob search", t.agentID,
    zap.String("pattern", pattern),
    zap.String("search_path", searchPath))
```

Line 116:
```go
// OLD:
t.logService.Debugf("[Agent %s] Full glob pattern: %s", t.agentID, fullPattern)

// NEW:
t.logService.DebugWithAgent("Full glob pattern", t.agentID,
    zap.String("pattern", fullPattern))
```

Line 121:
```go
// OLD:
t.logService.Errorf("[Agent %s] Invalid glob pattern '%s': %v", t.agentID, pattern, err)

// NEW:
t.logService.ErrorWithAgent("Invalid glob pattern", t.agentID,
    zap.String("pattern", pattern),
    zap.Error(err))
```

Line 131:
```go
// OLD:
t.logService.Debugf("[Agent %s] Pattern contains **, using recursive glob search", t.agentID)

// NEW:
t.logService.DebugWithAgent("Pattern contains **, using recursive glob search", t.agentID,
    zap.String("pattern", pattern))
```

Line 134:
```go
// OLD:
t.logService.Errorf("[Agent %s] Recursive glob failed for pattern '%s': %v", t.agentID, pattern, err)

// NEW:
t.logService.ErrorWithAgent("Recursive glob failed", t.agentID,
    zap.String("pattern", pattern),
    zap.Error(err))
```

Line 142:
```go
// OLD:
t.logService.Infof("[Agent %s] Glob search completed: pattern='%s' matches=%d", t.agentID, pattern, len(matches))

// NEW:
t.logService.InfoWithAgent("Glob search completed", t.agentID,
    zap.String("pattern", pattern),
    zap.Int("match_count", len(matches)))
```

Line 144:
```go
// OLD:
t.logService.Debugf("[Agent %s] Found matches: %v", t.agentID, matches)

// NEW:
t.logService.DebugWithAgent("Found matches", t.agentID,
    zap.Any("matches", matches))
```

Line 146:
```go
// OLD:
t.logService.Debugf("[Agent %s] No matches found for pattern '%s'", t.agentID, pattern)

// NEW:
t.logService.DebugWithAgent("No matches found", t.agentID,
    zap.String("pattern", pattern))
```

Lines 172, 183, 190, 196, 207, 213 (recursiveGlob helper):
```go
// OLD (line 172):
logService.Debugf("[Agent %s] Starting recursive glob: root='%s' pattern='%s'", agentID, root, pattern)

// NEW:
logService.DebugWithAgent("Starting recursive glob", agentID,
    zap.String("root", root),
    zap.String("pattern", pattern))

// OLD (line 183):
logService.Debugf("[Agent %s] Skipping path during walk: %s (error: %v)", agentID, path, err)

// NEW:
logService.DebugWithAgent("Skipping path during walk", agentID,
    zap.String("path", path),
    zap.Error(err))

// OLD (line 190):
logService.Debugf("[Agent %s] Pattern match failed for %s: %v", agentID, path, err)

// NEW:
logService.DebugWithAgent("Pattern match failed", agentID,
    zap.String("path", path),
    zap.Error(err))

// OLD (line 196):
logService.Debugf("[Agent %s] Recursive glob matched: %s", agentID, path)

// NEW:
logService.DebugWithAgent("Recursive glob matched", agentID,
    zap.String("path", path))

// OLD (line 207):
logService.Errorf("[Agent %s] Directory walk failed: %v", agentID, err)

// NEW:
logService.ErrorWithAgent("Directory walk failed", agentID, zap.Error(err))

// OLD (line 213):
logService.Debugf("[Agent %s] Recursive glob completed: found %d matches", agentID, len(matches))

// NEW:
logService.DebugWithAgent("Recursive glob completed", agentID,
    zap.Int("match_count", len(matches)))
```

- [ ] **Step 2: Run glob tests**

Run: `go test -v ./pkg/tools/ -run Glob`
Expected: PASS

- [ ] **Step 3: Commit glob logging updates**

```bash
git add pkg/tools/glob.go
git commit -m "feat(tools): use structured logging with agent ID in glob tool

Replace [Agent %s] message prefixes with InfoWithAgent/ErrorWithAgent
methods for proper structured logging with agent_id field.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 11: Update Agent Execution Helper Logging

**Files:**
- Modify: `pkg/tools/agent_execution_helper.go`

- [ ] **Step 1: Update agent_execution_helper.go logging calls**

Line 57:
```go
// OLD:
h.logService.Debugf("Starting synchronous execution for agent %s", agentID)

// NEW:
h.logService.DebugWithAgent("Starting synchronous execution", h.agentID,
    zap.String("target_agent_id", agentID.String()))
```

Line 63:
```go
// OLD:
h.logService.Errorf("Agent execution failed after %v: %v", duration, err)

// NEW:
h.logService.ErrorWithAgent("Agent execution failed", h.agentID,
    zap.Duration("duration", duration),
    zap.Error(err))
```

Line 73:
```go
// OLD:
h.logService.Warnf("Failed to store error agent result: %v", storeErr)

// NEW:
h.logService.WarnWithAgent("Failed to store error agent result", h.agentID, zap.Error(storeErr))
```

Line 89:
```go
// OLD:
h.logService.Debugf("Agent execution completed in %v", duration)

// NEW:
h.logService.DebugWithAgent("Agent execution completed", h.agentID,
    zap.Duration("duration", duration))
```

Line 99:
```go
// OLD:
h.logService.Warnf("Failed to store completion agent result: %v", storeErr)

// NEW:
h.logService.WarnWithAgent("Failed to store completion agent result", h.agentID, zap.Error(storeErr))
```

Line 110:
```go
// OLD:
h.logService.Debugf("Starting background execution for agent %s", agentID)

// NEW:
h.logService.DebugWithAgent("Starting background execution", h.agentID,
    zap.String("target_agent_id", agentID.String()))
```

Line 118:
```go
// OLD:
h.logService.Infof("Background agent %s was cancelled after %v", agentID, duration)

// NEW:
h.logService.InfoWithAgent("Background agent was cancelled", h.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Duration("duration", duration))
```

Line 129:
```go
// OLD:
h.logService.Errorf("Background agent %s failed after %v: %v", agentID, duration, err)

// NEW:
h.logService.ErrorWithAgent("Background agent failed", h.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Duration("duration", duration),
    zap.Error(err))
```

Line 147:
```go
// OLD:
h.logService.Infof("Background agent %s completed in %v", agentID, duration)

// NEW:
h.logService.InfoWithAgent("Background agent completed", h.agentID,
    zap.String("target_agent_id", agentID.String()),
    zap.Duration("duration", duration))
```

- [ ] **Step 2: Run background agent tests**

Run: `go test -v ./pkg/tools/ -run "BackgroundAgent"`
Expected: PASS

- [ ] **Step 3: Commit agent_execution_helper logging updates**

```bash
git add pkg/tools/agent_execution_helper.go
git commit -m "feat(tools): use structured logging with agent ID in agent_execution_helper

Replace formatted logging calls with InfoWithAgent/ErrorWithAgent
methods for better traceability of agent execution events.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 12: Update Remaining Tools (Optional Lower Priority)

These tools have fewer log calls. Update if time permits:

**Files:**
- `pkg/tools/edit.go` - Has many logging calls, update with structured fields
- `pkg/tools/read_file.go` - Has logging calls, update with structured fields
- `pkg/tools/write_file.go` - Has logging calls, update with structured fields
- `pkg/tools/grep.go` - Has logging calls, update with structured fields
- `pkg/tools/change_directory.go` - Has few logging calls
- `pkg/tools/current_time.go` - Has few logging calls
- `pkg/tools/flow_tools.go` - Has debug logging calls

**Pattern for each:**
- Replace `t.logService.Infof/Errorf/Warnf/Debugf` with `InfoWithAgent/ErrorWithAgent/etc`
- Move variable data from format string to `zap.String`, `zap.Int`, etc.
- Always include `t.agentID` as first parameter

---

## Chunk 3: Integration Testing

### Task 13: Integration Test for End-to-End JSON Logging

**Files:**
- Create: `pkg/logger/integration_test.go`

- [ ] **Step 1: Write integration test**

```go
package logger

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"
)

// TestJSONLoggingIntegration tests end-to-end JSON logging with agent ID
func TestJSONLoggingIntegration(t *testing.T) {
    tmpDir := t.TempDir()
    sessionID := uuid.New()

    // Create logger service
    svc := &service{
        config:         zap.NewProductionConfig(),
        configService:  &mockConfigService{},
        logBuffer:      newLogBuffer(1000, true),
    }

    // Enable file logging
    err := svc.EnableFileLogging(tmpDir, sessionID)
    require.NoError(t, err)
    defer svc.CloseFileLogging()

    // Create an agent ID
    agentID := uuid.New()

    // Log various levels with agent ID
    svc.InfoWithAgent("Agent started", agentID)
    svc.DebugWithAgent("Debug message", agentID, zap.String("detail", "testing"))
    svc.WarnWithAgent("Warning message", agentID, zap.Int("count", 42))
    svc.ErrorWithAgent("Error message", agentID, zap.Error(assert.AnError))

    // Also test without agent ID (should use Nil)
    svc.Info("System message without agent")

    // Flush
    err = svc.Flush()
    require.NoError(t, err)

    // Read and verify log file
    logPath := filepath.Join(tmpDir, sessionID.String()+".log")
    content, err := os.ReadFile(logPath)
    require.NoError(t, err)

    lines := strings.Split(strings.TrimSpace(string(content)), "\n")
    require.Equal(t, 5, len(lines), "Should have 5 log lines")

    // Verify each line is valid JSON
    for i, line := range lines {
        var entry map[string]interface{}
        err = json.Unmarshal([]byte(line), &entry)
        require.NoError(t, err, "Line %d should be valid JSON: %s", i, line)

        // Verify common fields
        assert.Contains(t, entry, "timestamp")
        assert.Contains(t, entry, "level")
        assert.Contains(t, entry, "message")
        assert.Contains(t, entry, "agent_id")
    }

    // Verify specific entries
    var infoEntry map[string]interface{}
    json.Unmarshal([]byte(lines[0]), &infoEntry)
    assert.Equal(t, "info", infoEntry["level"])
    assert.Equal(t, "Agent started", infoEntry["message"])
    assert.Equal(t, agentID.String(), infoEntry["agent_id"])

    var debugEntry map[string]interface{}
    json.Unmarshal([]byte(lines[1]), &debugEntry)
    assert.Equal(t, "debug", debugEntry["level"])
    assert.Equal(t, "Debug message", debugEntry["message"])
    assert.Equal(t, "testing", debugEntry["detail"])

    var errorEntry map[string]interface{}
    json.Unmarshal([]byte(lines[3]), &errorEntry)
    assert.Equal(t, "error", errorEntry["level"])
    assert.Equal(t, "Error message", errorEntry["message"])

    // System message should have Nil agent ID
    var systemEntry map[string]interface{}
    json.Unmarshal([]byte(lines[4]), &systemEntry)
    assert.Equal(t, "info", systemEntry["level"])
    assert.Equal(t, uuid.Nil.String(), systemEntry["agent_id"])
}
```

- [ ] **Step 2: Run integration test**

Run: `go test -v ./pkg/logger/ -run TestJSONLoggingIntegration`
Expected: PASS

- [ ] **Step 3: Commit integration test**

```bash
git add pkg/logger/integration_test.go
git commit -m "test(logger): add integration test for JSON logging

End-to-end test verifying JSON log format with agent ID tracking
across all log levels and structured fields.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Final Verification

### Task 14: Run Full Test Suite

- [ ] **Step 1: Run all logger tests**

Run: `go test -v ./pkg/logger/`
Expected: All PASS

- [ ] **Step 2: Run all tools tests**

Run: `go test -v ./pkg/tools/`
Expected: All PASS

- [ ] **Step 3: Run full project tests**

Run: `go test -v ./...`
Expected: All PASS

- [ ] **Step 4: Manual verification**

1. Run the gollum application
2. Execute a bash command through an agent
3. Check the session log file in `.gollum/logs/`
4. Verify the log contains valid JSON with agent_id field

Example command to check log:
```bash
cat .gollum/logs/*.log | jq '.'
```

Expected: JSON output with agent_id field present

---

## Completion

When all tasks are complete, the logging system will:

1. ✅ Write session logs in JSON format
2. ✅ Include agent_id in all agent-generated log entries
3. ✅ Maintain human-readable stdout/console logs
4. ✅ Populate in-memory buffer with agent ID for TUI
5. ✅ Enable querying logs by agent ID

**Example JSON log entry:**
```json
{"timestamp":"2026-03-10T16:00:30.710629364+01:00","level":"INFO","message":"Executing bash command","agent_id":"9e51c80a-7a11-40f1-93b8-4c0133bd6383","command":"ls -la"}
```
