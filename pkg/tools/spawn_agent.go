package tools

import (
	"context"
	"fmt"
	"strings"
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
	"go.uber.org/zap"
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
		toolRegistry    shared.ToolRegistry
		agent           shared.Agent
	}

	// SpawnAgentToolProvider creates SpawnAgentTool instances via DI
	SpawnAgentToolProvider interface {
		CreateTool(agent shared.Agent, agentFactory shared.AgentFactory) gollem.Tool
	}

	spawnAgentToolProvider struct {
		logService      logger.LoggerService
		registry        registry.AgentRegistry
		promptManager   manager.PromptManager
		executionHelper AgentExecutionHelper
		configService   config.ConfigService
		hookManager     hooks.HookManager
		toolRegistry    shared.ToolRegistry
	}
)

// NewSpawnAgentToolProvider creates a provider for SpawnAgent tools
func NewSpawnAgentToolProvider(injector do.Injector) (SpawnAgentToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)
	executionHelper := do.MustInvoke[AgentExecutionHelper](injector)
	configService := do.MustInvoke[config.ConfigService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	toolRegistry := do.MustInvoke[shared.ToolRegistry](injector)

	return &spawnAgentToolProvider{
		logService:      logService,
		registry:        agentRegistry,
		promptManager:   promptManager,
		executionHelper: executionHelper,
		configService:   configService,
		hookManager:     hookManager,
		toolRegistry:    toolRegistry,
	}, nil
}

// CreateSpawnAgentTool creates a new SpawnAgentTool for a specific sender
func (p *spawnAgentToolProvider) CreateTool(agent shared.Agent, agentFactory shared.AgentFactory) gollem.Tool {
	return &spawnAgentToolImpl{
		logService:      p.logService,
		agentFactory:    agentFactory,
		registry:        p.registry,
		promptManager:   p.promptManager,
		executionHelper: p.executionHelper,
		configService:   p.configService,
		hookManager:     p.hookManager,
		toolRegistry:    p.toolRegistry,
		agent:           agent,
	}
}

// formatToolNamesForDescription creates a comma-separated list of tool names for descriptions
func formatToolNamesForDescription(tools []shared.ToolName) string {
	if len(tools) == 0 {
		return "none"
	}

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = "'" + tool.String() + "'"
	}

	// Join with commas
	result := strings.Join(names, ", ")
	return result
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
			"allowed_tools": {
				Type:  gollem.TypeArray,
				Items: &gollem.Parameter{Type: gollem.TypeString},
				Description: fmt.Sprintf("List of tool names the agent can access. Available built-in tools: %s. MCP tools use 'server_name/tool_name' format. If omitted, agent has no tools available.",
					formatToolNamesForDescription(shared.SubAgentBuiltinTools)),
			},
		},
	}
}

// Run executes the SpawnAgent tool to create and run subagents
func (t *spawnAgentToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameSpawnAgent, args,
		func() (map[string]any, error) {
			return t.runSpawnAgent(ctx, args)
		})
}

// runSpawnAgent implements the core SpawnAgent logic
func (t *spawnAgentToolImpl) runSpawnAgent(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Validate required parameters
	role, errResp := args.MustGetString(shared.ParamRole)
	if errResp != nil {
		return t.executionHelper.ErrorResponse("role is required and must be a non-empty string"), nil
	}

	description, errResp := args.MustGetString(shared.ParamDescription)
	if errResp != nil {
		return t.executionHelper.ErrorResponse("description is required and must be a non-empty string"), nil
	}

	prompt, errResp := args.MustGetString(shared.ParamPrompt)
	if errResp != nil {
		return t.executionHelper.ErrorResponse("prompt is required and must be a non-empty string"), nil
	}

	// Check run_in_background parameter (defaults to false)
	runInBackground := args.GetBool(shared.ParamRunInBackground, false)

	// Check share_context parameter (defaults to false)
	shareContext := args.GetBool(shared.ParamShareContext, false)

	// Parse allowed_tools parameter (optional)
	allowedTools := args.GetStringSlice(shared.ParamAllowedTools)

	// Validate built-in tool names (MCP tools with "/" are validated later in factory)
	for _, toolName := range allowedTools {
		if !strings.Contains(toolName, "/") {
			// Check against tool registry (spawn_agent is allowed when explicitly specified)
			if !t.toolRegistry.IsValidTool(shared.ToolName(toolName)) {
				t.logService.WarnWithContext("Invalid built-in tool name in allowed_tools",
					t.agent.ToSessionContext(),
					zap.String("tool_name", toolName))
				return t.executionHelper.ErrorResponse(fmt.Sprintf("invalid built-in tool name: %s", toolName)), nil
			}
		}
	}

	t.logService.InfoWithContext("Spawning subagent",
		t.agent.ToSessionContext(),
		zap.String("role", role),
		zap.String("description", description),
		zap.Bool("background", runInBackground),
		zap.Bool("share_context", shareContext))

	// Get specialized subagent prompt from PromptManager (includes role, description, and tool names)
	systemPrompt, err := t.promptManager.GetSubagentTaskPrompt(role, description)
	if err != nil {
		t.logService.ErrorWithContext("Failed to get subagent prompt",
			t.agent.ToSessionContext(), zap.Error(err))
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to get subagent prompt: %v", err)), nil
	}

	t.logService.DebugWithContext("Inheriting LLM config from parent", t.agent.ToSessionContext(),
		zap.String("parent_agent_id", t.agent.GetID().String()))

	// Get message history if share_context is enabled
	var history *gollem.History
	if shareContext {
		history, err = t.agent.GetMessageHistory(ctx)
		if err != nil {
			t.logService.WarnWithContext("Failed to get message history from parent", t.agent.ToSessionContext(), zap.Error(err))
			// Continue without history - non-fatal error
		}
	}

	// Create subagent configuration
	agentID := uuid.New()
	parentID := t.agent.GetID()
	parentConfig := t.agent.GetConfig()

	clientConfig, err := t.configService.ClientConfig(shared.AgentTypeSubAgent)
	if err != nil {
		return nil, fmt.Errorf("invalid llm client config: %w", err)
	}

	subagentConfig := &shared.AgentConfig{
		Type:            shared.AgentTypeSubAgent,
		AllowCompaction: false, // Don't allow compaction in Sub-agents
		ParentID:        &parentID,
		SessionContext: shared.SessionContext{
			SessionID: parentConfig.SessionContext.SessionID,
			ChannelID: parentConfig.SessionContext.ChannelID,
			AgentID:   agentID,
		},
		SystemPrompt:    systemPrompt,
		Role:            role,
		Description:     description,
		LLMClientConfig: clientConfig,
		OutputMode:      shared.OutputModeSummary, // Sub-agents use summary mode
		History:         history,                  // Include parent message history for context awareness
		AllowedTools:    allowedTools,             // Explicitly allowed tools
	}

	// Create the subagent using the factory (which now adds default tools)
	subagent, err := t.agentFactory.CreateAgent(ctx, subagentConfig)
	if err != nil {
		t.logService.ErrorWithContext("Failed to create subagent",
			t.agent.ToSessionContext(), zap.Error(err))
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to create subagent: %v", err)), nil
	}

	t.logService.InfoWithContext("Created subagent",
		t.agent.ToSessionContext(),
		zap.String("subagent_id", subagent.GetID().String()),
		zap.String("role", role))

	// Create initial agent result
	agentResult := shared.AgentResult{
		AgentID:   subagent.GetID(),
		Status:    shared.AgentStatusRunning,
		Output:    map[string]any{},
		StartedAt: time.Now().Unix(),
	}
	if err := t.registry.StoreAgentResult(agentResult); err != nil {
		t.logService.ErrorWithContext("Failed to store agent result",
			t.agent.ToSessionContext(), zap.Error(err))
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
