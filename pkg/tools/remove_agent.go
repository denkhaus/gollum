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
)

type (
	// RemoveAgentTool removes agents and all their subagents recursively
	RemoveAgentTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
		senderID    uuid.UUID
	}

	// RemoveAgentToolProvider creates RemoveAgentTool instances via DI
	RemoveAgentToolProvider interface {
		CreateTool(senderID uuid.UUID) *RemoveAgentTool
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

// CreateRemoveAgentTool creates a new RemoveAgentTool for a specific sender
func (p *removeAgentToolProvider) CreateTool(senderID uuid.UUID) *RemoveAgentTool {
	return &RemoveAgentTool{
		logService:  p.logService,
		hookManager: p.hookManager,
		registry:    p.registry,
		senderID:    senderID,
	}
}

// Spec returns the tool specification for the RemoveAgent tool
func (t *RemoveAgentTool) Spec() gollem.ToolSpec {
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
func (t *RemoveAgentTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.senderID, shared.ToolNameRemoveAgent, args,
		func() (map[string]any, error) {
			return t.runRemoveAgent(ctx, args)
		})
}

// runRemoveAgent implements the core RemoveAgent logic
func (t *RemoveAgentTool) runRemoveAgent(_ context.Context, args map[string]any) (map[string]any, error) {
	agentIDStr, ok := args["agent_id"].(string)
	if !ok {
		t.logService.Debugf("RemoveAgent: invalid agent_id type from sender %s", t.senderID)
		return map[string]any{
			"success": false,
			"error":   "agent_id is required and must be a string",
		}, nil
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		t.logService.Debugf("RemoveAgent: invalid UUID format '%s' from sender %s", agentIDStr, t.senderID)
		return map[string]any{
			"success": false,
			"error":   "agent_id must be a valid UUID",
		}, nil
	}

	t.logService.Debugf("RemoveAgent: attempting to remove agent %s by sender %s", agentID, t.senderID)

	// Prevent self-removal
	if agentID == t.senderID {
		t.logService.Infof("RemoveAgent: self-removal attempted by agent %s", t.senderID)
		return map[string]any{
			"success": false,
			"error":   "cannot remove yourself",
		}, nil
	}

	// Check if target agent exists
	targetAgent, exists := t.registry.GetAgent(agentID)
	if !exists {
		t.logService.Infof("RemoveAgent: target agent %s not found", agentID)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("agent %s not found", agentID),
		}, nil
	}

	// Get target agent config to check parent relationship
	targetConfig := targetAgent.GetConfig()

	// Check permissions: only parent can remove child agents
	hasPermission := targetConfig.ParentID != nil && *targetConfig.ParentID == t.senderID

	// Also check if sender is the root/creator agent
	// This allows agents to remove their own spawned subagents
	senderChildren := t.registry.GetChildren(t.senderID)
	for _, child := range senderChildren {
		if child.GetID() == agentID {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		t.logService.Warnf("RemoveAgent: permission denied for sender %s to remove agent %s (not parent)", t.senderID, agentID)
		return map[string]any{
			"success": false,
			"error":   "permission denied: only parent agent can remove child agents",
		}, nil
	}

	// Check for force parameter
	force := false
	if forceVal, ok := args["force"].(bool); ok {
		force = forceVal
	}

	// Get children count for reporting
	children := t.registry.GetChildren(agentID)
	childCount := len(children)

	t.logService.Debugf("RemoveAgent: agent %s has %d children, force=%v", agentID, childCount, force)

	// Warn if agent has children and force is not set
	if childCount > 0 && !force {
		t.logService.Infof("RemoveAgent: agent %s has %d children but force=false", agentID, childCount)
		return map[string]any{
			"success":      false,
			"error":        fmt.Sprintf("agent has %d active children. Use force=true to remove with all descendants", childCount),
			"child_count":  childCount,
			"has_children": true,
		}, nil
	}

	t.logService.Infof("RemoveAgent: removing agent %s with %d descendants by sender %s", agentID, childCount, t.senderID)

	// Perform recursive removal using registry's Cleanup method
	err = t.registry.Cleanup(agentID)
	if err != nil {
		t.logService.Errorf("RemoveAgent: failed to cleanup agent %s: %v", agentID, err)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to remove agent: %v", err),
		}, nil
	}

	t.logService.Infof("RemoveAgent: successfully removed agent %s and all %d descendants", agentID, childCount)

	return map[string]any{
		"success":       true,
		"removed_agent": agentID.String(),
		"removed_by":    t.senderID.String(),
		"child_count":   childCount,
		"message":       fmt.Sprintf("Agent %s and all %d descendants removed successfully", agentID.String(), childCount),
	}, nil
}
