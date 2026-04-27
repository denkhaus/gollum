// Package tools provides tool implementations for agent operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// agentOutputToolImpl retrieves results from background agents
	agentOutputToolImpl struct {
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
		agent       shared.Agent
	}

	// AgentOutputToolProvider creates AgentOutputTool instances via DI
	AgentOutputToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	agentOutputToolProvider struct {
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
	}
)

// NewAgentOutputToolProvider creates a provider for AgentOutput tools
func NewAgentOutputToolProvider(injector do.Injector) (AgentOutputToolProvider, error) {
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &agentOutputToolProvider{
		hookManager: hookManager,
		registry:    registry,
	}, nil
}

// CreateAgentOutputTool creates a new AgentOutputTool for a specific sender
func (p *agentOutputToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &agentOutputToolImpl{
		hookManager: p.hookManager,
		registry:    p.registry,
		agent:       agent,
	}
}

// Spec returns the tool specification for the AgentOutput tool
func (t *agentOutputToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameAgentOutput.String(),
		Description: "Retrieves results from agents running in background (async execution mode).",
		Parameters: map[string]*gollem.Parameter{
			"agent_id": {
				Type:        gollem.TypeString,
				Description: fmt.Sprintf("The agent ID (UUID) of the background agent to retrieve results from. Agent must have been started with %s tool using run_in_background=true.", shared.ToolNameSpawnAgent),
			},
			"block": {
				Type:        gollem.TypeBoolean,
				Description: "If true (default), waits for agent completion. If false, returns current status immediately without blocking.",
			},
			"timeout": {
				Type:        gollem.TypeInteger,
				Description: fmt.Sprintf("Maximum time to wait in milliseconds (default: %d, max: %d). Only applies when block=true.", DefaultAgentOutputTimeout, MaxAgentOutputTimeout),
			},
		},
	}
}

// Run executes the AgentOutput tool to retrieve results from background agents
func (t *agentOutputToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameAgentOutput, args,
		func() (map[string]any, error) {
			return t.runAgentOutput(ctx, args)
		})
}

// runAgentOutput implements the core AgentOutput logic
func (t *agentOutputToolImpl) runAgentOutput(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Validate agent_id
	agentIDStr, errResp := args.MustGetString(shared.ParamAgentID)
	if errResp != nil {
		return errorResponseAgentOutput("agent_id is required and must be a non-empty string"), nil
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return errorResponseAgentOutput("agent_id must be a valid UUID"), nil
	}

	// Parse block parameter (defaults to true)
	block := args.GetBool(shared.ParamBlock, true)

	// Parse timeout parameter (defaults to DefaultAgentOutputTimeout)
	timeoutInt := args.GetInt(shared.ParamTimeout, DefaultAgentOutputTimeout)
	// Clamp timeout between MinAgentOutputTimeout and MaxAgentOutputTimeout
	timeoutInt = shared.Clamp(timeoutInt, MinAgentOutputTimeout, MaxAgentOutputTimeout)
	timeout := time.Duration(timeoutInt) * time.Millisecond

	// PERMISSION CHECK: Verify caller is DIRECT parent of target agent
	// Separation of concerns: each agent can only access outputs from their direct children
	if !t.registry.IsDirectParent(t.agent.GetID(), agentID) {
		return errorResponseAgentOutput(
			fmt.Sprintf("permission denied: you can only get output from your direct subagents (not grandchildren or other agents). Agent %s is not your direct child.", agentID.String()),
		), nil
	}

	// Get current agent result
	currentResult, exists := t.registry.GetAgentResult(agentID)
	if !exists {
		return errorResponseAgentOutput(fmt.Sprintf("agent %s not found", agentID.String())), nil
	}

	// Non-blocking mode: return current status immediately
	if !block {
		return successResponseAgentOutputNonBlocking(currentResult), nil
	}

	// Blocking mode: wait for completion or timeout
	finalResult, err := t.registry.WaitForAgent(ctx, agentID, timeout)
	if err != nil {
		// WaitForAgent returns error on timeout, but includes current result
		return successResponseAgentOutputTimeout(finalResult, err.Error()), nil
	}

	return successResponseAgentOutputBlocking(finalResult), nil
}

func successResponseAgentOutputNonBlocking(result *shared.AgentResult) map[string]any {
	response := map[string]any{
		"success":    true,
		"agent_id":   result.AgentID.String(),
		"status":     string(result.Status),
		"started_at": result.StartedAt,
	}

	if result.Status == shared.AgentStatusCompleted || result.Status == shared.AgentStatusFailed {
		if result.CompletedAt != nil {
			response["completed_at"] = *result.CompletedAt
		}
		if result.Status == shared.AgentStatusCompleted && result.Output != nil {
			response["output"] = result.Output
		}
		if result.Status == shared.AgentStatusFailed && result.Error != "" {
			response["error"] = result.Error
		}
	}

	return response
}

func successResponseAgentOutputBlocking(result *shared.AgentResult) map[string]any {
	response := map[string]any{
		"success":    true,
		"agent_id":   result.AgentID.String(),
		"status":     string(result.Status),
		"started_at": result.StartedAt,
	}

	if result.CompletedAt != nil {
		response["completed_at"] = *result.CompletedAt
	}

	if result.Status == shared.AgentStatusCompleted && result.Output != nil {
		response["output"] = result.Output
	}

	if result.Status == shared.AgentStatusFailed && result.Error != "" {
		response["error"] = result.Error
	}

	return response
}

func successResponseAgentOutputTimeout(result *shared.AgentResult, timeoutErr string) map[string]any {
	response := map[string]any{
		"success":    false,
		"agent_id":   result.AgentID.String(),
		"status":     string(result.Status),
		"started_at": result.StartedAt,
		"error":      timeoutErr,
	}

	if result.CompletedAt != nil {
		response["completed_at"] = *result.CompletedAt
	}

	if result.Output != nil {
		response["output"] = result.Output
	}

	if result.Error != "" {
		response["agent_error"] = result.Error
	}

	return response
}

func errorResponseAgentOutput(errMsg string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   errMsg,
	}
}
