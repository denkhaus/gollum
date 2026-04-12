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
	modeHead = "head"
	modeTail = "tail"
	modeAll  = "all"
)

type (
	// sessionLogsToolImpl allows agents to query session logs with filtering.
	sessionLogsToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
	}

	// SessionLogsToolProvider creates SessionLogsTool instances via DI.
	SessionLogsToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
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

// CreateSessionLogsTool creates a new SessionLogsTool with agent.
func (p *sessionLogsToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &sessionLogsToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		agent:       agent,
	}
}

// Run executes the SessionLogs tool to retrieve filtered log entries.
func (t *sessionLogsToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameSessionLogs, args,
		func() (map[string]any, error) {
			return t.runSessionLogs(ctx, args)
		})
}

// runSessionLogs implements the core SessionLogs logic
func (t *sessionLogsToolImpl) runSessionLogs(_ context.Context, args ToolRequestParams) (map[string]any, error) {
	// Get mode parameter (default to tail mode)
	mode := args.GetString(shared.ParamMode, modeTail)
	if mode == "" {
		mode = modeTail
	}

	// Validate mode
	if !isValidMode(mode) {
		return nil, fmt.Errorf("invalid mode: %s (must be one of: %s, %s, %s)", mode, modeHead, modeTail, modeAll)
	}

	// Parse optional parameters
	count := args.GetInt(shared.ParamCount, DefaultSessionLogCount)
	if count <= 0 {
		count = DefaultSessionLogCount
	}

	// Build log filter
	filter := logger.LogFilter{}

	// Set level filter if provided
	if level := args.GetString(shared.ParamLevel, ""); level != "" {
		filter.Level = level
	}

	// Set agent ID filter if provided
	if agentIDStr := args.GetString(shared.ParamAgentID, ""); agentIDStr != "" {
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid agent_id UUID format: %w", err)
		}
		filter.AgentID = agentID
	}

	// Set since_seq filter if provided
	sinceSeqInt := args.GetInt64(shared.ParamSinceSeq, 0)
	if sinceSeqInt > 0 {
		filter.SinceSeq = &sinceSeqInt
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
	if filter.SinceSeq != nil {
		result["filters_applied"].(map[string]interface{})["since_seq"] = *filter.SinceSeq
	}

	return result, nil
}

// Spec returns the tool specification for the SessionLogs tool.
func (t *sessionLogsToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name: shared.ToolNameSessionLogs.String(),
		Description: "Queries session logs with intelligent filtering. " +
			"Supports multiple modes: '" + modeTail + "' (last N entries, default), '" + modeHead + "' (first N entries, newest first) and " +
			"'" + modeAll + "' (all entries with optional count limit). " +
			"Optional filters: level (debug/info/warn/error), agent_id (specific agent UUID).",
		Parameters: map[string]*gollem.Parameter{
			"mode": {
				Type:        gollem.TypeString,
				Description: "Query mode: '" + modeTail + "' (last N, chronological), '" + modeHead + "' (first N, newest first), '" + modeAll + "' (all entries). Defaults to '" + modeTail + "'.",
			},
			"count": {
				Type:        gollem.TypeInteger,
				Description: fmt.Sprintf("Maximum number of entries to return (default: %d). Applied to 'head', 'tail', and 'all' modes.", DefaultSessionLogCount),
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
	case modeHead, modeTail, modeAll:
		return true
	default:
		return false
	}
}
