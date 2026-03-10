# Structured JSON Logging with Agent ID Tracking

**Date:** 2026-03-10
**Status:** Approved
**Author:** Claude (via brainstorming)

## Problem Statement

The current logging system uses plain text (tab-separated) format for session logs. This makes it difficult to:

1. **Trace actions to specific agents** - When a subagent executes a tool (e.g., bash command), the log entry doesn't indicate which agent performed the action
2. **Query logs programmatically** - Plain text logs require regex parsing
3. **Correlate events** - No structured fields for filtering or analysis

### Example Issue

From session log `10ac5328-7a57-4d9b-8b31-b0c396a96ecb.log`:
```
2026-03-10T16:00:28.804763001+01:00	INFO	Created skill subagent 9e51c80a-7a11-40f1-93b8-4c0133bd6383
2026-03-10T16:00:30.710629364+01:00	INFO	Executing bash command: ls -la ...
```

The bash command execution log doesn't show that agent `9e51c80a-7a11-40f1-93b8-4c0133bd6383` executed it.

## Solution

Replace the console encoder with JSON encoder for file logs, and add helper methods that automatically include agent IDs in structured log fields.

## Design

### 1. JSON Log Format

Minimal JSON structure with agent ID:

```json
{
  "timestamp": "2026-03-10T16:00:30.710629364+01:00",
  "level": "INFO",
  "message": "Executing bash command",
  "agent_id": "9e51c80a-7a11-40f1-93b8-4c0133bd6383",
  "command": "ls -la ~/.config/superpowers/skills"
}
```

### 2. Logger Helper Methods

New methods on `LoggerService` interface:

```go
// InfoWithAgent logs an info message with agent ID
InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
ErrorWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
DebugWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
WarnWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)
```

Implementation automatically prepends the agent_id field:

```go
func (s *service) InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {
    allFields := append([]zap.Field{zap.String("agent_id", agentID.String())}, fields...)
    s.Info(msg, allFields...)
}
```

### 3. Tool Logging Pattern

Before:
```go
t.logService.Infof("Executing bash command: %s", command)
```

After:
```go
t.logService.InfoWithAgent("Executing bash command", t.agentID, zap.String("command", command))
```

### 4. Buffer Integration

Update `storeInBuffer` to extract agent_id from fields:

```go
func (s *service) storeInBuffer(level string, msg string, fields []zap.Field) {
    entry := LogEntry{
        Timestamp: time.Now(),
        Level:     level,
        Message:   msg,
        Fields:    zapFieldsToMap(fields),
        Sequence:  s.logBuffer.nextSeq,
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

## Implementation Plan

See implementation plan to be created via writing-plans skill.

## Files Modified

- `pkg/logger/file.go` - Change encoder to JSON
- `pkg/logger/logger.go` - Add helper methods, update storeInBuffer
- `pkg/tools/bash.go` - Update logging calls
- `pkg/tools/spawn_agent.go` - Update logging calls
- `pkg/tools/send_message.go` - Update logging calls (if applicable)
- `pkg/tools/remove_agent.go` - Update logging calls (if applicable)

## Testing

- Unit tests for new helper methods
- Test that agent_id is correctly extracted and stored in buffer
- Integration test to verify JSON log file format

## Trade-offs

**Pro:**
- Structured logs are queryable and parseable
- Agent actions are fully traceable
- Tools explicitly declare their agent ID when logging

**Con:**
- Log files are less human-readable (need JSON pretty-printer)
- Slightly more verbose logging calls in tools
- JSON encoding has minor performance cost (mitigated by minimal fields)
