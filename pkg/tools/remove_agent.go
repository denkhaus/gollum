package tools

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// removeAgentToolImpl removes agents and all their subagents recursively
	removeAgentToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
		agent       shared.Agent
	}

	// RemoveAgentToolProvider creates RemoveAgentTool instances via DI
	RemoveAgentToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	removeAgentToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
	}
)

// NewRemoveAgentToolProvider creates a provider for RemoveAgent tools
func NewRemoveAgentToolProvider(injector do.Injector) (RemoveAgentToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &removeAgentToolProvider{
		logService:  logService,
		hookManager: hookManager,
		registry:    registry,
	}, nil
}

// CreateRemoveAgentTool creates a new RemoveAgentTool for a specific agent
func (p *removeAgentToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &removeAgentToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		registry:    p.registry,
		agent:       agent,
	}
}

// Spec returns the tool specification for the RemoveAgent tool
func (t *removeAgentToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameRemoveAgent.String(),
		Description: "Removes an agent and all its subagents recursively. Only the agent creator or parent can remove an agent. Cannot remove yourself.",
		Parameters: map[string]*gollem.Parameter{
			"agent_id": {
				Type:        gollem.TypeString,
				Description: "UUID of the agent to remove",
			},
			"force": {
				Type:        gollem.TypeBoolean,
				Description: "Force removal even if agent has active children (optional, defaults to false)",
			},
		},
	}
}

// Run executes the RemoveAgent tool to remove agents
func (t *removeAgentToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agent.GetID(), shared.ToolNameRemoveAgent, args,
		func() (map[string]any, error) {
			return t.runRemoveAgent(ctx, args)
		})
}

// runRemoveAgent implements the core RemoveAgent logic
func (t *removeAgentToolImpl) runRemoveAgent(_ context.Context, args ToolRequestParams) (map[string]any, error) {
	agentIDStr, errResp := args.MustGetString(shared.ParamAgentID)
	if errResp != nil {
		return errResp, nil
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		t.logService.DebugWithContext("RemoveAgent: invalid UUID format", t.agent.ToLoggingContext(),
			zap.String("agent_id_str", agentIDStr),
			zap.String("sender_id", t.agent.GetID().String()))
		return map[string]any{
			"success": false,
			"error":   "agent_id must be a valid UUID",
		}, nil
	}

	t.logService.DebugWithContext("RemoveAgent: attempting to remove agent", t.agent.ToLoggingContext(),
		zap.String("target_agent_id", agentID.String()),
		zap.String("sender_id", t.agent.GetID().String()))

	// Prevent self-removal
	if agentID == t.agent.GetID() {
		t.logService.InfoWithContext("RemoveAgent: self-removal attempted", t.agent.ToLoggingContext(),
			zap.String("sender_id", t.agent.GetID().String()))
		return map[string]any{
			"success": false,
			"error":   "cannot remove yourself",
		}, nil
	}

	// Check if target agent exists
	targetAgent, exists := t.registry.GetAgent(agentID)
	if !exists {
		t.logService.InfoWithContext("RemoveAgent: target agent not found", t.agent.ToLoggingContext(),
			zap.String("target_agent_id", agentID.String()))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("agent %s not found", agentID),
		}, nil
	}

	// Get target agent config to check parent relationship
	targetConfig := targetAgent.GetConfig()

	// Check permissions: only parent can remove child agents
	hasPermission := targetConfig.ParentID != nil && *targetConfig.ParentID == t.agent.GetID()

	// Also check if sender is the root/creator agent
	// This allows agents to remove their own spawned subagents
	senderChildren := t.registry.GetChildren(t.agent.GetID())
	for _, child := range senderChildren {
		if child.GetID() == agentID {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		t.logService.WarnWithContext("RemoveAgent: permission denied", t.agent.ToLoggingContext(),
			zap.String("sender_id", t.agent.GetID().String()),
			zap.String("target_agent_id", agentID.String()),
			zap.String("reason", "not_parent"))
		return map[string]any{
			"success": false,
			"error":   "permission denied: only parent agent can remove child agents",
		}, nil
	}

	// Check for force parameter
	force := args.GetBool(shared.ParamForce, false)

	// Get children count for reporting
	children := t.registry.GetChildren(agentID)
	childCount := len(children)

	t.logService.DebugWithContext("RemoveAgent: agent has children", t.agent.ToLoggingContext(),
		zap.String("target_agent_id", agentID.String()),
		zap.Int("child_count", childCount),
		zap.Bool("force", force))

	// Warn if agent has children and force is not set
	if childCount > 0 && !force {
		t.logService.InfoWithContext("RemoveAgent: agent has children but force=false", t.agent.ToLoggingContext(),
			zap.String("target_agent_id", agentID.String()),
			zap.Int("child_count", childCount))
		return map[string]any{
			"success":      false,
			"error":        fmt.Sprintf("agent has %d active children. Use force=true to remove with all descendants", childCount),
			"child_count":  childCount,
			"has_children": true,
		}, nil
	}

	t.logService.InfoWithContext("RemoveAgent: removing agent with descendants", t.agent.ToLoggingContext(),
		zap.String("target_agent_id", agentID.String()),
		zap.Int("descendant_count", childCount),
		zap.String("sender_id", t.agent.GetID().String()))

	// Perform recursive removal using registry's Cleanup method
	err = t.registry.Cleanup(agentID)
	if err != nil {
		t.logService.ErrorWithContext("RemoveAgent: failed to cleanup agent", t.agent.ToLoggingContext(),
			zap.String("target_agent_id", agentID.String()),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to remove agent: %v", err),
		}, nil
	}

	t.logService.InfoWithContext("RemoveAgent: successfully removed agent", t.agent.ToLoggingContext(),
		zap.String("target_agent_id", agentID.String()),
		zap.Int("descendant_count", childCount))

	return map[string]any{
		"success":       true,
		"removed_agent": agentID.String(),
		"removed_by":    t.agent.GetID().String(),
		"child_count":   childCount,
		"message":       fmt.Sprintf("Agent %s and all %d descendants removed successfully", agentID.String(), childCount),
	}, nil
}
