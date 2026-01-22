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
	// ResumeAgentTool resumes and executes an existing registered agent
	ResumeAgentTool struct {
		logService      logger.LoggerService
		hookManager     hooks.HookManager
		registry        registry.AgentRegistry
		executionHelper AgentExecutionHelper
		senderID        uuid.UUID
	}

	// ResumeAgentToolProvider creates ResumeAgentTool instances via DI
	ResumeAgentToolProvider interface {
		CreateTool(senderID uuid.UUID) *ResumeAgentTool
	}

	resumeAgentToolProvider struct {
		logService      logger.LoggerService
		hookManager     hooks.HookManager
		registry        registry.AgentRegistry
		executionHelper AgentExecutionHelper
	}
)

// NewResumeAgentToolProvider creates a provider for ResumeAgent tools
func NewResumeAgentToolProvider(injector do.Injector) (ResumeAgentToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)
	executionHelper := do.MustInvoke[AgentExecutionHelper](injector)

	return &resumeAgentToolProvider{
		logService:      logService,
		hookManager:     hookManager,
		registry:        registry,
		executionHelper: executionHelper,
	}, nil
}

// CreateTool creates a new ResumeAgentTool for a specific sender
func (p *resumeAgentToolProvider) CreateTool(senderID uuid.UUID) *ResumeAgentTool {
	return &ResumeAgentTool{
		logService:      p.logService,
		hookManager:     p.hookManager,
		registry:        p.registry,
		executionHelper: p.executionHelper,
		senderID:        senderID,
	}
}

// Spec returns the tool specification for ResumeAgentTool
func (t *ResumeAgentTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name: shared.ToolNameResumeAgent,
		Description: fmt.Sprintf(`Sends a new prompt to an existing agent created with the %s tool.
			The agent preserves its conversation context across multiple prompts. Remove the agent when done using the %s tool.`,
			shared.ToolNameSpawnAgent,
			shared.ToolNameRemoveAgent,
		),
		Parameters: map[string]*gollem.Parameter{
			"agent_id": {
				Type:        gollem.TypeString,
				Description: "ID of the existing agent to resume (UUID string)",
			},
			"prompt": {
				Type:        gollem.TypeString,
				Description: "New task prompt for the agent to execute",
			},
			"run_in_background": {
				Type:        gollem.TypeBoolean,
				Description: fmt.Sprintf("If true, executes asynchronously. Use %s tool to retrieve results. If false or omitted, waits for completion and returns result directly.", shared.ToolNameAgentOutput),
			},
		},
		Required: []string{"agent_id", "prompt"},
	}
}

// Run executes the ResumeAgent tool to resume and run existing agents
func (t *ResumeAgentTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.senderID, shared.ToolNameResumeAgent, args,
		func() (map[string]any, error) {
			return t.runResumeAgent(ctx, args)
		})
}

// runResumeAgent implements the core ResumeAgent logic
func (t *ResumeAgentTool) runResumeAgent(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Validate required parameters
	agentIDStr, ok := args["agent_id"].(string)
	if !ok || agentIDStr == "" {
		return t.executionHelper.ErrorResponse("agent_id is required and must be a non-empty string"), nil
	}

	// Parse agent ID
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return t.executionHelper.ErrorResponse(fmt.Sprintf("invalid agent_id format: %v", err)), nil
	}

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return t.executionHelper.ErrorResponse("prompt is required and must be a non-empty string"), nil
	}

	// Check run_in_background parameter (defaults to false)
	runInBackground := false
	if bgVal, exists := args["run_in_background"]; exists {
		if bgBool, ok := bgVal.(bool); ok {
			runInBackground = bgBool
		}
	}

	// PERMISSION CHECK: Verify caller is DIRECT parent of target agent
	if !t.registry.IsDirectParent(t.senderID, agentID) {
		t.logService.Warnf("Permission denied: agent %s attempted to resume agent %s (not direct parent)", t.senderID, agentID)

		return t.executionHelper.ErrorResponse(
			fmt.Sprintf("permission denied: you can only resume your direct subagents (not grandchildren or other agents). Agent %s is not your direct child.", agentID),
		), nil
	}

	t.logService.Infof("Resuming agent %s from sender %s (background=%v)", agentID, t.senderID, runInBackground)

	// Get the agent from registry
	agent, exists := t.registry.GetAgent(agentID)
	if !exists {
		return t.executionHelper.ErrorResponse(
			fmt.Sprintf("agent %s not found in registry", agentID),
		), nil
	}

	// Get agent config from the agent itself
	agentConfig := agent.GetConfig()

	t.logService.Infof("Found agent %s (role=%s)", agentID, agentConfig.Role)

	// Delete any previous agent result to prevent returning stale results
	if err := t.registry.DeleteAgentResult(agentID); err != nil {
		t.logService.Warnf("Failed to delete previous agent result for %s: %v", agentID, err)
		// Continue anyway - this is not a critical error
	}

	// Execute based on mode
	if runInBackground {
		// Asynchronous execution - create cancellable context from request context
		// This allows the background agent to be cancelled if the request is cancelled
		bgCtx, cancel := context.WithCancel(ctx)

		// Store cancel function in registry for this agent
		if err := t.registry.SetCancelFunc(agentID, cancel); err != nil {
			t.logService.Warnf("Failed to store cancel function for agent %s: %v", agentID, err)
			// Continue anyway - the agent will still run, just won't be cancellable
		}

		t.logService.Infof("Starting background execution for agent %s", agentID)
		go t.executionHelper.ExecuteInBackground(bgCtx, agent, prompt)
		return t.executionHelper.SuccessResponseResumeAsync(agentID, agentConfig.Role, agentConfig.Description), nil
	}

	// Synchronous execution
	t.logService.Infof("Executing agent %s synchronously", agentID)
	response, err := t.executionHelper.ExecuteSynchronously(ctx, agent, prompt)
	if err != nil {
		t.logService.Errorf("Agent execution failed: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("execution failed: %v", err)), nil
	}

	t.logService.Infof("Agent %s completed successfully", agentID)
	return response, nil
}
