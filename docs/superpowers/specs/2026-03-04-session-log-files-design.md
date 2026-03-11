# Session Log Files Design

## Overview

Enhance the application to persist session logs to `.gollum/logs/<session_uuid>.log` files in the startup workspace, with automatic cleanup of old session logs when the configured maximum is exceeded.

## Requirements

- ApplicationService creates `.gollum/` directory on startup (future home for other gollum data)
- LoggerService manages `.gollum/logs/` subdirectory and file logging
- Configurable maximum number of session log files to retain
- Oldest session logs deleted when count exceeds maximum
- Session UUID is the supervisor agent's UUID

## Configuration

### LoggingConfig Extension (`pkg/config/service.go`)

```go
type LoggingConfig struct {
    SessionLogBufferSize int `envconfig:"SESSION_LOG_BUFFER_SIZE" default:"1000"`
    SessionLogEnabled   bool `envconfig:"SESSION_LOG_ENABLED" default:"true"`
    MaxSessionLogFiles   int  `envconfig:"MAX_SESSION_LOG_FILES" default:"10"` // NEW
}
```

**Validation:**
- Min: 0 (unlimited, no cleanup)
- Max: 100
- Value 0 means no automatic cleanup

### Environment Variable

```bash
GOLLUM_LOGGING_MAX_SESSION_LOG_FILES=10
```

## LoggerService Changes (`pkg/logger/logger.go`)

### Interface Additions

```go
type LoggerService interface {
    // ... existing methods ...

    // EnableFileLogging enables file logging to .gollum/logs/<sessionID>.log
    // Creates logs directory, cleans old logs, opens file for writing
    EnableFileLogging(gollumDir string, sessionID uuid.UUID) error

    // CloseFileLogging closes the current log file if open
    CloseFileLogging() error
}
```

### Implementation Details

- `EnableFileLogging`:
  - Reads `MaxSessionLogFiles` from config
  - Creates `.gollum/logs/` directory if needed
  - Scans existing `*.log` files, deletes oldest if over limit
  - Opens `.gollum/logs/<sessionID>.log` for writing
  - Adds file writer to zap logger (in addition to buffer)

- `CloseFileLogging`:
  - Flushes any buffered writes
  - Closes the log file

## ApplicationService Changes (`pkg/app/service.go`)

### Constants

```go
const (
    gollumDirName = ".gollum"
)
```

### Struct Addition

```go
type applicationServiceImpl struct {
    // ... existing fields ...
    gollumDir string
}
```

### New Method

```go
// ensureGollumDirectory creates .gollum directory if it doesn't exist
func (p *applicationServiceImpl) ensureGollumDirectory() error {
    p.gollumDir = filepath.Join(p.startupDirectory, gollumDirName)
    return os.MkdirAll(p.gollumDir, 0755)
}
```

### Run() Flow Update

```go
func (p *applicationServiceImpl) Run(ctx context.Context, startupDirectory string) error {
    p.startupDirectory = startupDirectory

    // Create .gollum directory
    if err := p.ensureGollumDirectory(); err != nil {
        return fmt.Errorf("failed to create .gollum directory: %w", err)
    }

    // Prime FileStateManager
    if err := p.primeFileStateManager(ctx); err != nil {
        return err
    }

    // Create and register Supervisor agent
    agent, _, err := p.createSupervisorAgent(ctx)
    if err != nil {
        return err
    }

    // Enable file logging (LoggerService handles logs/ subdir and cleanup)
    if err := p.logService.EnableFileLogging(p.gollumDir, agent.GetID()); err != nil {
        return fmt.Errorf("failed to enable file logging: %w", err)
    }
    defer p.logService.CloseFileLogging()

    // Run interactive loop
    return p.runInteractiveLoop(ctx, agent)
}
```

## Directory Structure

```
<workspace>/
└── .gollum/
    └── logs/
        ├── 550e8400-e29b-41d4-a716-446655440000.log
        ├── 550e8400-e29b-41d4-a716-446655440001.log
        └── ...
```

## Cleanup Algorithm

1. Scan `.gollum/logs/` for `*.log` files
2. Get file info (name, modification time)
3. Sort by modification time (oldest first)
4. If count > MaxSessionLogFiles: delete oldest files until count <= max
5. If MaxSessionLogFiles == 0: skip cleanup (unlimited)

## Files to Modify

1. `pkg/config/service.go` - Add `MaxSessionLogFiles` to LoggingConfig
2. `pkg/logger/logger.go` - Add `EnableFileLogging`, `CloseFileLogging`, file handle management
3. `pkg/app/service.go` - Add `.gollum/` directory creation, wire up file logging

## Files to Update (Mocks)

1. `pkg/mocks/mock_logger_service.go` - Regenerate after interface changes

## Testing Considerations

- Test cleanup logic with various file counts
- Test that current session log is not deleted
- Test file creation in `.gollum/logs/`
- Test MaxSessionLogFiles = 0 (unlimited) case
