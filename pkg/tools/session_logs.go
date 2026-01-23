// Package tools provides the SessionLogsTool for querying session logs.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

const (
	modeHead  = "head"
	modeTail  = "tail"
	modeSince = "since"
	modeAll   = "all"
)

type (
	// SessionLogsTool allows agents to query session logs with filtering.
	SessionLogsTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
	}

	// SessionLogsToolProvider creates SessionLogsTool instances via DI.
	SessionLogsToolProvider interface {
		CreateTool(agentID uuid.UUID) *SessionLogsTool
	}

	sessionLogsToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}
)

// NewSessionLogsToolProvider creates a provider for SessionLogs tools.
func NewSessionLogsToolProvider(injector do.Injector) (SessionLogsToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	return &sessionLogsToolProvider{
		logService:  logService,
		hookManager: hookManager,
	}, nil
}

// CreateSessionLogsTool creates a new SessionLogsTool with agent ID.
func (p *sessionLogsToolProvider) CreateTool(agentID uuid.UUID) *SessionLogsTool {
	return &SessionLogsTool{
		logService:  p.logService,
		hookManager: p.hookManager,
		agentID:     agentID,
	}
}

// Run executes the SessionLogs tool to retrieve filtered log entries.
func (t *SessionLogsTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameSessionLogs, args,
		func() (map[string]any, error) {
			return t.runSessionLogs(ctx, args)
		})
}

// runSessionLogs implements the core SessionLogs logic
func (t *SessionLogsTool) runSessionLogs(_ context.Context, args map[string]any) (map[string]any, error) {
	// Get mode parameter (required)
	mode, exists := args["mode"].(string)
	if !exists || mode == "" {
		// Default to tail mode
		mode = modeTail
	}

	// Validate mode
	if !isValidMode(mode) {
		return nil, fmt.Errorf("invalid mode: %s (must be one of: %s, %s, %s, %s)", mode, modeHead, modeTail, modeSince, modeAll)
	}

	// Parse optional parameters
	count := 100 // default
	if c, exists := args["count"].(float64); exists {
		count = int(c)
		if count <= 0 {
			return nil, fmt.Errorf("count must be positive, got: %d", count)
		}
	}

	// Build log filter
	filter := logger.LogFilter{}

	// Set level filter if provided
	if level, exists := args["level"].(string); exists && level != "" {
		filter.Level = level
	}

	// Set agent ID filter if provided
	if agentIDStr, exists := args["agent_id"].(string); exists && agentIDStr != "" {
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid agent_id UUID format: %w", err)
		}
		filter.AgentID = agentID
	}

	// Handle "since" mode
	if mode == modeSince {
		sinceStr, exists := args["since"].(string)
		if !exists || sinceStr == "" {
			return nil, fmt.Errorf("since parameter is required for 'since' mode")
		}

		sinceTime, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid since datetime format (expected RFC3339): %w", err)
		}
		filter.Since = sinceTime
	}

	// Apply mode-specific logic
	switch mode {
	case modeHead:
		// Head: newest first (reverse), limited by count
		filter.Reverse = true
		filter.Count = count
	case modeTail:
		// Tail: oldest first (chronological), limited by count
		filter.Count = count
	case modeSince:
		// Since: chronological order, no count limit (use count as soft limit if huge)
		if count > 0 {
			filter.Count = count
		}
	case modeAll:
		// All: chronological order, optional count limit
		if count > 0 {
			filter.Count = count
		}
	}

	// Retrieve logs from logger service
	entries := t.logService.GetLogs(filter)

	// Convert entries to JSON-serializable format
	jsonEntries := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		jsonEntry := map[string]interface{}{
			"timestamp": entry.Timestamp.Format(time.RFC3339),
			"level":     entry.Level,
			"message":   entry.Message,
		}

		// Add optional fields
		if len(entry.Fields) > 0 {
			jsonEntry["fields"] = entry.Fields
		}
		if entry.AgentID != uuid.Nil {
			jsonEntry["agent_id"] = entry.AgentID.String()
		}

		jsonEntries = append(jsonEntries, jsonEntry)
	}

	// Build response
	result := map[string]any{
		"entries":        jsonEntries,
		"total_matching": len(jsonEntries),
		"returned":       len(jsonEntries),
		"filters_applied": map[string]interface{}{
			"mode":  mode,
			"count": count,
		},
	}

	if filter.Level != "" {
		result["filters_applied"].(map[string]interface{})["level"] = filter.Level
	}
	if filter.AgentID != uuid.Nil {
		result["filters_applied"].(map[string]interface{})["agent_id"] = filter.AgentID.String()
	}
	if !filter.Since.IsZero() {
		result["filters_applied"].(map[string]interface{})["since"] = filter.Since.Format(time.RFC3339)
	}

	return result, nil
}

// Spec returns the tool specification for the SessionLogs tool.
func (t *SessionLogsTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name: shared.ToolNameSessionLogs,
		Description: "Queries session logs with intelligent filtering. " +
			"Supports multiple modes: '" + modeTail + "' (last N entries, default), '" + modeHead + "' (first N entries, newest first), " +
			"'" + modeSince + "' (all entries after a datetime), and '" + modeAll + "' (all entries with optional count limit). " +
			"Optional filters: level (debug/info/warn/error), agent_id (specific agent UUID).",
		Parameters: map[string]*gollem.Parameter{
			"mode": {
				Type:        gollem.TypeString,
				Description: "Query mode: '" + modeTail + "' (last N, chronological), '" + modeHead + "' (first N, newest first), '" + modeSince + "' (after datetime), '" + modeAll + "' (all entries). Defaults to '" + modeTail + "'.",
			},
			"count": {
				Type:        gollem.TypeInteger,
				Description: "Maximum number of entries to return (default: 100). Applied to 'head', 'tail', and 'all' modes.",
			},
			"since": {
				Type:        gollem.TypeString,
				Description: "ISO 8601 datetime (RFC3339) for 'since' mode (e.g., '2025-12-30T23:00:00Z'). Required for 'since' mode.",
			},
			"level": {
				Type:        gollem.TypeString,
				Description: "Filter by log level: 'debug', 'info', 'warn', 'error' (case-insensitive).",
			},
			"agent_id": {
				Type:        gollem.TypeString,
				Description: "Filter by specific agent UUID (e.g., '123e4567-e89b-12d3-a456-426614174000').",
			},
		},
	}
}

// isValidMode checks if the mode is valid.
func isValidMode(mode string) bool {
	switch mode {
	case modeHead, modeTail, modeSince, modeAll:
		return true
	default:
		return false
	}
}
