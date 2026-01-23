package tools

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// CurrentTimeTool returns the current time
	CurrentTimeTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
	}
	// CurrentTimeToolProvider creates CurrentTimeTool instances via DI
	CurrentTimeToolProvider interface {
		CreateTool(agentID uuid.UUID) *CurrentTimeTool
	}

	currentTimeToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}
)

// NewCurrentTimeToolProvider creates a provider for CurrentTime tools
func NewCurrentTimeToolProvider(injector do.Injector) (CurrentTimeToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	return &currentTimeToolProvider{
		logService:  logService,
		hookManager: hookManager,
	}, nil
}

// CreateCurrentTimeTool creates a new CurrentTimeTool with agent ID
func (p *currentTimeToolProvider) CreateTool(agentID uuid.UUID) *CurrentTimeTool {
	return &CurrentTimeTool{
		logService:  p.logService,
		hookManager: p.hookManager,
		agentID:     agentID,
	}
}

// Run executes the CurrentTime tool to return the current time
func (t *CurrentTimeTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameCurrentTime, args,
		func() (map[string]any, error) {
			return t.runCurrentTime(ctx, args)
		})
}

// runCurrentTime implements the core CurrentTime logic
func (t *CurrentTimeTool) runCurrentTime(_ context.Context, args map[string]any) (map[string]any, error) {
	// Get timezone from args, default to UTC
	timezone := "UTC"
	if tz, exists := args["timezone"].(string); exists && tz != "" {
		timezone = tz
	}

	// Load location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		t.logService.Warnf("Invalid timezone '%s', using UTC: %v", timezone, err)
		loc = time.UTC
	}

	now := time.Now().In(loc)

	result := map[string]any{
		"time":           now.Format("2006-01-02 15:04:05"),
		"timezone":       timezone,
		"unix_timestamp": now.Unix(),
		"day_of_week":    now.Weekday().String(),
		"is_dst":         now.IsDST(),
		"rfc3339":        now.Format(time.RFC3339),
	}

	t.logService.Debugf("Current time (%s): %s", timezone, result["time"])

	return result, nil
}

// Spec returns the tool specification for the CurrentTime tool
func (t *CurrentTimeTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameCurrentTime,
		Description: "Returns the current date and time, optionally in a specific timezone",
		Parameters: map[string]*gollem.Parameter{
			"timezone": {
				Type:        gollem.TypeString,
				Description: "Timezone string (e.g., 'UTC', 'Europe/Berlin', 'America/New_York'). Defaults to UTC if not provided or invalid.",
			},
		},
		Required: []string{},
	}
}
