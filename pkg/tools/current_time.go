package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

const (
	defaultTimezone = "UTC"
)

type (
	// currentTimeToolImpl returns the current time
	currentTimeToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
	}
	// CurrentTimeToolProvider creates CurrentTimeTool instances via DI
	CurrentTimeToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
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

// CreateCurrentTimeTool creates a new CurrentTimeTool with agent reference
func (p *currentTimeToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &currentTimeToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		agent:       agent,
	}
}

// Run executes the CurrentTime tool to return the current time
func (t *currentTimeToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameCurrentTime, args,
		func() (map[string]any, error) {
			return t.runCurrentTime(ctx, args)
		})
}

// runCurrentTime implements the core CurrentTime logic
func (t *currentTimeToolImpl) runCurrentTime(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Get timezone from args, default to UTC
	timezone := args.GetString(shared.ParamTimezone, defaultTimezone)
	if timezone == "" {
		timezone = defaultTimezone
	}

	// Load location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Use enriched logging if flow/step context is available
		if fc := hooks.GetFlowStepContext(ctx); fc != nil {
			t.logService.WarnWithFlowStep(
				fmt.Sprintf("Invalid timezone '%s', using UTC: %v", timezone, err),
				fc.FlowName, fc.StateName, fc.StepType)
		} else {
			t.logService.Warnf("Invalid timezone '%s', using UTC: %v", timezone, err)
		}
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

	// Use enriched logging if flow/step context is available
	timeStr, _ := result["time"].(string)
	if fc := hooks.GetFlowStepContext(ctx); fc != nil {
		t.logService.DebugWithFlowStep("current_time() called", fc.FlowName, fc.StateName, fc.StepType, zap.String("timezone", timezone), zap.String("time", timeStr))
	} else {
		t.logService.DebugWithContext("current_time() called", t.agent.ToLoggingContext(), zap.String("timezone", timezone), zap.String("time", timeStr))
	}

	return result, nil
}

// Spec returns the tool specification for the CurrentTime tool
func (t *currentTimeToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameCurrentTime.String(),
		Description: "Returns the current date and time, optionally in a specific timezone",
		Parameters: map[string]*gollem.Parameter{
			"timezone": {
				Type:        gollem.TypeString,
				Description: "Timezone string (e.g., 'UTC', 'Europe/Berlin', 'America/New_York'). Defaults to UTC if not provided or invalid.",
			},
		},
	}
}
