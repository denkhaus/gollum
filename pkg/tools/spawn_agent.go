package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// spawnAgentToolImpl creates new subagents with immediate task execution
	spawnAgentToolImpl struct {
		logService      logger.LoggerService
		agentFactory    shared.AgentFactory
		registry        registry.AgentRegistry
		promptManager   manager.PromptManager
		executionHelper AgentExecutionHelper
		configService   config.ConfigService
		hookManager     hooks.HookManager
		senderID        uuid.UUID
	}

	// SpawnAgentToolProvider creates SpawnAgentTool instances via DI
	SpawnAgentToolProvider interface {
		CreateTool(senderID uuid.UUID, agentFactory shared.AgentFactory) gollem.Tool
	}

	spawnAgentToolProvider struct {
		logService      logger.LoggerService
		registry        registry.AgentRegistry
		promptManager   manager.PromptManager
		executionHelper AgentExecutionHelper
		configService   config.ConfigService
		hookManager     hooks.HookManager
	}
)

// NewSpawnAgentToolProvider creates a provider for SpawnAgent tools
func NewSpawnAgentToolProvider(injector do.Injector) (SpawnAgentToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)
	executionHelper := do.MustInvoke[AgentExecutionHelper](injector)
	configService := do.MustInvoke[config.ConfigService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)

	return &spawnAgentToolProvider{
		logService:      logService,
		registry:        registry,
		promptManager:   promptManager,
		executionHelper: executionHelper,
		configService:   configService,
		hookManager:     hookManager,
	}, nil
}

// CreateSpawnAgentTool creates a new SpawnAgentTool for a specific sender
func (p *spawnAgentToolProvider) CreateTool(senderID uuid.UUID, agentFactory shared.AgentFactory) gollem.Tool {
	return &spawnAgentToolImpl{
		logService:      p.logService,
		agentFactory:    agentFactory,
		registry:        p.registry,
		promptManager:   p.promptManager,
		executionHelper: p.executionHelper,
		configService:   p.configService,
		hookManager:     p.hookManager,
		senderID:        senderID,
	}
}

// Spec returns the tool specification for SpawnAgentTool
func (t *spawnAgentToolImpl) Spec() gollem.ToolSpec {
	maxSubAgents := t.configService.GetAgentLimits().MaxSubAgentsPerParent

	return gollem.ToolSpec{
		Name: shared.ToolNameSpawnAgent.String(),
		// Description references ToolNameResumeAgent for cross-tool discoverability
		Description: fmt.Sprintf(`Creates a new subagent with a task prompt and executes it immediately.
			The agent preserves context and remains accessible for follow-up interactions via the %s tool.
			Remove the agent when done using the %s tool. Maximum concurrent subagents: %d.`,
			shared.ToolNameResumeAgent,
			shared.ToolNameRemoveAgent,
			maxSubAgents,
		),
		Parameters: map[string]*gollem.Parameter{
			"role": {
				Type:        gollem.TypeString,
				Description: "Agent identity for UI/Logging (e.g., 'Code Reviewer', 'Git Specialist', 'Data Analyst')",
			},
			"description": {
				Type:        gollem.TypeString,
				Description: "Short task description (3-5 words) for identification (e.g., 'Refactoring registry.go')",
			},
			"prompt": {
				Type:        gollem.TypeString,
				Description: "Detailed task instructions for the agent to execute",
			},
			"run_in_background": {
				Type:        gollem.TypeBoolean,
				Description: fmt.Sprintf("If true, executes asynchronously. Use %s tool to retrieve results. If false or omitted, waits for completion and returns result directly.", shared.ToolNameAgentOutput),
			},
			"share_context": {
				Type:        gollem.TypeBoolean,
				Description: "If true, includes parent agent's message history in subagent configuration for context awareness. Default is false.",
			},
		},
	}
}

// Run executes the SpawnAgent tool to create and run subagents
func (t *spawnAgentToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.senderID, shared.ToolNameSpawnAgent, args,
		func() (map[string]any, error) {
			return t.runSpawnAgent(ctx, args)
		})
}

// runSpawnAgent implements the core SpawnAgent logic
func (t *spawnAgentToolImpl) runSpawnAgent(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Validate required parameters
	role, ok := args["role"].(string)
	if !ok || role == "" {
		return t.executionHelper.ErrorResponse("role is required and must be a non-empty string"), nil
	}

	description, ok := args["description"].(string)
	if !ok || description == "" {
		return t.executionHelper.ErrorResponse("description is required and must be a non-empty string"), nil
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

	// Check share_context parameter (defaults to false)
	shareContext := false
	if scVal, exists := args["share_context"]; exists {
		if scBool, ok := scVal.(bool); ok {
			shareContext = scBool
		}
	}

	t.logService.Infof("Spawning subagent: role=%s description=%s background=%v share_context=%v", role, description, runInBackground, shareContext)

	// Get specialized subagent prompt from PromptManager (includes role, description, and tool names)
	systemPrompt, err := t.promptManager.GetSubagentTaskPrompt(role, description)
	if err != nil {
		t.logService.Errorf("Failed to get subagent prompt: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to get subagent prompt: %v", err)), nil
	}

	// Get parent agent to inherit LLM client config and optionally message history
	parentAgent, hasParent := t.registry.GetAgent(t.senderID)
	var llmClientConfig *shared.LLMClientConfig
	if hasParent {
		llmClientConfig = parentAgent.GetConfig().LLMClientConfig
		t.logService.Debugf("Inheriting LLM config from parent agent %s", t.senderID)
	} else {
		// Default fallback
		llmClientConfig = &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		}
		t.logService.Debugf("Using default LLM config (no parent agent found)")
	}

	var history *gollem.History
	if shareContext && hasParent {
		history, err = parentAgent.GetMessageHistory(ctx)
		if err != nil {
			t.logService.Warnf("Failed to get message history from parent agent: %v", err)
			// Continue without history - non-fatal error
		}
	}

	// Create subagent configuration
	taskID := uuid.New()
	subagentConfig := &shared.AgentConfig{
		AllowCompaction:  false, // Don't allow compaction in Sub-agents
		ID:               taskID,
		ParentID:         &t.senderID,
		SystemPrompt:     systemPrompt,
		Role:             role,
		Description:      description,
		LLMClientConfig:  llmClientConfig,
		OutputMode:       shared.OutputModeSummary, // Sub-agents use summary mode
		History:          history,                  // Include parent message history for context awareness
	}

	// Create the subagent using the factory (which now adds default tools)
	subagent, err := t.agentFactory.CreateAgent(ctx, subagentConfig)
	if err != nil {
		t.logService.Errorf("Failed to create subagent: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to create subagent: %v", err)), nil
	}

	t.logService.Infof("Created subagent %s (role=%s)", subagent.GetID(), role)

	// Create initial agent result
	agentResult := shared.AgentResult{
		AgentID:   subagent.GetID(),
		Status:    shared.AgentStatusRunning,
		Output:    map[string]any{},
		StartedAt: time.Now().Unix(),
	}
	if err := t.registry.StoreAgentResult(agentResult); err != nil {
		t.logService.Errorf("Failed to store agent result: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to store agent result: %v", err)), nil
	}

	// Execute based on mode
	if runInBackground {
		// Asynchronous execution - create independent context for background agent
		// This ensures the background agent continues running even after the parent request completes
		// The background agent is not tied to the parent request lifecycle
		bgCtx := context.Background()

		// Register agent with cancel function (using bgCtx, not parent ctx)
		// Note: The cancel function won't be tied to parent context anymore
		// Background agents should be explicitly cancelled via RemoveAgentTool
		if err := t.registry.Register(subagent, subagentConfig, nil); err != nil {
			t.logService.Errorf("Failed to register background agent: %v", err)
			return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to register agent: %v", err)), nil
		}

		t.logService.Infof("Starting background agent %s", subagent.GetID())
		go t.executionHelper.ExecuteInBackground(bgCtx, subagent, prompt)
		return t.executionHelper.SuccessResponseAsync(subagent.GetID(), role, description), nil
	}

	// Synchronous execution - register without cancel function
	if err := t.registry.Register(subagent, subagentConfig); err != nil {
		t.logService.Errorf("Failed to register synchronous agent: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to register agent: %v", err)), nil
	}

	t.logService.Infof("Executing synchronous agent %s", subagent.GetID())
	response, err := t.executionHelper.ExecuteSynchronously(ctx, subagent, prompt)
	if err != nil {
		t.logService.Errorf("Agent execution failed: %v", err)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("execution failed: %v", err)), nil
	}

	t.logService.Infof("Agent %s completed successfully", subagent.GetID())
	return response, nil
}
