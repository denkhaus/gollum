package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	mcp "github.com/denkhaus/gollum/pkg/mcp"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/middleware/compacter"
	"github.com/m-mizutani/gollem/strategy/simple"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// defaultAgentFactory implements the AgentFactory interface
type defaultAgentFactory struct {
	logService       logger.LoggerService
	configService    config.ConfigService
	clientProvider   llm.ClientProvider
	registry         registry.AgentRegistry
	mcpRegistry      mcpregistry.MCPRegistry
	promptManager    manager.PromptManager
	workspaceService workspace.Service
	channelProvider  channel.ChannelMiddlewareProvider
	skillsService    skills.SkillService
	// Tool providers for adding default tools to all agents
	spawnAgentToolProv      tools.SpawnAgentToolProvider
	agentOutputToolProv     tools.AgentOutputToolProvider
	removeAgentToolProv     tools.RemoveAgentToolProvider
	resumeAgentToolProv     tools.ResumeAgentToolProvider
	listAgentsToolProv      tools.ListAgentsToolProvider
	currentTimeToolProv     tools.CurrentTimeToolProvider
	bashToolProv            tools.BashToolProvider
	writeFileToolProv       tools.WriteFileToolProvider
	readFileToolProv        tools.ReadFileToolProvider
	globToolProv            tools.GlobToolProvider
	grepToolProv            tools.GrepToolProvider
	editToolProv            tools.EditToolProvider
	sessionLogsToolProv     tools.SessionLogsToolProvider
	changeDirectoryToolProv tools.ChangeDirectoryToolProvider
	invokeSkillToolProv     tools.InvokeSkillToolProvider
	mcpToolProvider         mcp.MCPToolProvider
}

// NewAgentFactory creates a new AgentFactory
func NewAgentFactory(injector do.Injector) (shared.AgentFactory, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	configService := do.MustInvoke[config.ConfigService](injector)
	clientProvider := do.MustInvoke[llm.ClientProvider](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
	workspaceService := do.MustInvoke[workspace.Service](injector)
	channelProvider := do.MustInvoke[channel.ChannelMiddlewareProvider](injector)
	mcpToolProvider := do.MustInvoke[mcp.MCPToolProvider](injector)
	skillsService := do.MustInvoke[skills.SkillService](injector)

	// Get tool providers
	spawnAgentToolProv := do.MustInvoke[tools.SpawnAgentToolProvider](injector)
	agentOutputToolProv := do.MustInvoke[tools.AgentOutputToolProvider](injector)
	removeAgentToolProv := do.MustInvoke[tools.RemoveAgentToolProvider](injector)
	resumeAgentToolProv := do.MustInvoke[tools.ResumeAgentToolProvider](injector)
	listAgentsToolProv := do.MustInvoke[tools.ListAgentsToolProvider](injector)
	currentTimeToolProv := do.MustInvoke[tools.CurrentTimeToolProvider](injector)
	bashToolProv := do.MustInvoke[tools.BashToolProvider](injector)
	writeFileToolProv := do.MustInvoke[tools.WriteFileToolProvider](injector)
	readFileToolProv := do.MustInvoke[tools.ReadFileToolProvider](injector)
	globToolProv := do.MustInvoke[tools.GlobToolProvider](injector)
	grepToolProv := do.MustInvoke[tools.GrepToolProvider](injector)
	editToolProv := do.MustInvoke[tools.EditToolProvider](injector)
	sessionLogsToolProv := do.MustInvoke[tools.SessionLogsToolProvider](injector)
	changeDirectoryToolProv := do.MustInvoke[tools.ChangeDirectoryToolProvider](injector)
	invokeSkillToolProv := do.MustInvoke[tools.InvokeSkillToolProvider](injector)

	return &defaultAgentFactory{
		logService:              logService,
		skillsService:           skillsService,
		workspaceService:        workspaceService,
		configService:           configService,
		clientProvider:          clientProvider,
		mcpRegistry:             mcpRegistry,
		registry:                registry,
		promptManager:           promptManager,
		channelProvider:         channelProvider,
		spawnAgentToolProv:      spawnAgentToolProv,
		agentOutputToolProv:     agentOutputToolProv,
		removeAgentToolProv:     removeAgentToolProv,
		resumeAgentToolProv:     resumeAgentToolProv,
		listAgentsToolProv:      listAgentsToolProv,
		currentTimeToolProv:     currentTimeToolProv,
		bashToolProv:            bashToolProv,
		writeFileToolProv:       writeFileToolProv,
		readFileToolProv:        readFileToolProv,
		globToolProv:            globToolProv,
		grepToolProv:            grepToolProv,
		editToolProv:            editToolProv,
		sessionLogsToolProv:     sessionLogsToolProv,
		changeDirectoryToolProv: changeDirectoryToolProv,
		invokeSkillToolProv:     invokeSkillToolProv,
		mcpToolProvider:         mcpToolProvider,
	}, nil
}

// CreateAgent creates a new agent with the given configuration
func (f *defaultAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
	// Ensure the config has an ID, generate one if not set
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	// Resolve tools from AllowedTools
	tools, err := f.resolveTools(ctx, config.ID, config.AllowedTools)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to resolve tools").
			WithContext("agent_id", config.ID)
	}

	// Create the base agent
	defAgent := &DefaultAgent{
		clientProvider: f.clientProvider,
		configService:  f.configService,
		logService:     f.logService,
		registry:       f.registry,
		id:             config.ID,
		config:         config,
		promptManager:  f.promptManager,
		tools:          tools, // Store resolved tools for recreation
	}

	// Get LLM client
	client, err := f.clientProvider.GetClient(ctx, config.LLMClientConfig)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
			WithContext("agent_id", config.ID)
	}

	// Set the LLM client for session recreation
	defAgent.llmClient = client

	// Set default strategy
	if config.Strategy == nil {
		config.Strategy = simple.New()
	}

	// Set default output mode
	if config.OutputMode == "" {
		config.OutputMode = shared.OutputModeFull
	}

	// Build base gollem options (common to all modes)
	baseOptions := []gollem.Option{
		gollem.WithStrategy(config.Strategy),
		gollem.WithTools(tools...), // Use resolved tools
		gollem.WithSystemPrompt(config.SystemPrompt),
	}

	if config.History != nil {
		baseOptions = append(baseOptions,
			gollem.WithHistory(config.History),
		)
	}

	if config.AllowCompaction {
		// Create context compacter
		compacterPrompt, err := f.promptManager.GetPromptByID(ctx, prompt.PromptIDCompacter)
		if err != nil {
			return nil, errs.Wrap(err, errs.TypeInternal, "failed to get compacter prompt").
				WithContext("agent_id", config.ID)
		}

		contextCompacter := compacter.NewContentBlockMiddleware(client,
			compacter.WithSummaryPrompt(compacterPrompt.Content),
		)

		baseOptions = append(baseOptions,
			gollem.WithContentBlockMiddleware(contextCompacter),
		)
	}

	// Add channel middleware (always - routes all agent output to channel system)
	// The channel SDK handles all messaging, replacing the old DisplayMiddleware and SummaryMiddleware
	channelMiddleware := f.channelProvider.CreateChannelMiddleware(config.ID, config.Role)
	baseOptions = append(baseOptions,
		gollem.WithContentBlockMiddleware(channelMiddleware.ContentBlockMiddleware),
		gollem.WithToolMiddleware(channelMiddleware.ToolMiddleware),
	)

	// Create gollem agent with configured options
	defAgent.base = gollem.New(client, baseOptions...)

	return defAgent, nil
}

func (f *defaultAgentFactory) resolveTools(ctx context.Context, agentID uuid.UUID, allowedTools []string) ([]gollem.Tool, error) {
	if allowedTools == nil {
		return nil, nil
	}

	var tools []gollem.Tool
	var warnings []string

	for _, toolName := range allowedTools {
		// Check if MCP tool (format: "server_name/tool_name")
		if strings.Contains(toolName, "/") {
			tool, err := f.mcpToolProvider.CreateTool(agentID, toolName)
			if err != nil {
				// MCP tool not found - warn, don't fail
				warnings = append(warnings, fmt.Sprintf("MCP tool '%s': %v", toolName, err))
				continue
			}
			tools = append(tools, tool)
		} else {
			// Built-in tool
			tool, err := f.resolveBuiltinTool(agentID, toolName)
			if err != nil {
				return nil, err
			}
			tools = append(tools, tool)
		}
	}

	// Log warnings for missing MCP tools
	if len(warnings) > 0 {
		f.logService.Warn("Some MCP tools were not found",
			zap.Strings("missing_tools", warnings))
	}

	return tools, nil
}

func (p *defaultAgentFactory) CreateSupervisorAgent(ctx context.Context) (shared.Agent, *shared.AgentConfig, error) {

	// Get supervisor prompt from PromptManager
	systemPrompt, err := p.promptManager.GetPromptWithContext(ctx,
		prompt.PromptIDSupervisorSystem,
		&prompt.RenderContext{
			Workspace: &shared.WorkspaceContext{
				SkillsXML:   p.skillsService.GetSkillsXML(),
				Skills:      p.skillsService.GetSkillInfos(),
				CurrentPath: p.workspaceService.GetCurrentWorkspace(),
			},
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get supervisor prompt: %w", err)
	}

	// Combine MCP tools and built-in tools for the supervisor
	allowedTools := p.mcpRegistry.GetToolNames()
	if len(allowedTools) == 0 {
		p.logService.Warn("no mcp tools configured for supervision agent")
	}

	// Add built-in tools (excluding flow executor tools)
	for _, toolName := range shared.SupervisorBuiltinTools {
		allowedTools = append(allowedTools, toolName.String())
	}

	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		AllowedTools:    allowedTools,
		Role:            "Supervisor Agent",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/glm-4.7",
		},
	}

	// Create agent
	agent, err := p.CreateAgent(ctx, agentConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create supervisor agent: %v", err)
	}

	// Register in registry
	if err := p.registry.Register(agent, agentConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to register supervisor agent: %w", err)
	}
	p.logService.Infof("Supervisor agent %s registered", agent.GetID())

	return agent, agentConfig, nil
}

func (f *defaultAgentFactory) resolveBuiltinTool(agentID uuid.UUID, name string) (gollem.Tool, error) {
	switch shared.ToolName(name) {
	case shared.ToolNameBash:
		return f.bashToolProv.CreateTool(agentID), nil
	case shared.ToolNameCurrentTime:
		return f.currentTimeToolProv.CreateTool(agentID), nil
	case shared.ToolNameWriteFile:
		return f.writeFileToolProv.CreateTool(agentID), nil
	case shared.ToolNameReadFile:
		return f.readFileToolProv.CreateTool(agentID), nil
	case shared.ToolNameGlob:
		return f.globToolProv.CreateTool(agentID), nil
	case shared.ToolNameGrep:
		return f.grepToolProv.CreateTool(agentID), nil
	case shared.ToolNameEdit:
		return f.editToolProv.CreateTool(agentID), nil
	case shared.ToolNameSpawnAgent:
		return f.spawnAgentToolProv.CreateTool(agentID, f), nil
	case shared.ToolNameAgentOutput:
		return f.agentOutputToolProv.CreateTool(agentID), nil
	case shared.ToolNameRemoveAgent:
		return f.removeAgentToolProv.CreateTool(agentID), nil
	case shared.ToolNameResumeAgent:
		return f.resumeAgentToolProv.CreateTool(agentID), nil
	case shared.ToolNameListAgents:
		return f.listAgentsToolProv.CreateTool(agentID), nil
	case shared.ToolNameSessionLogs:
		return f.sessionLogsToolProv.CreateTool(agentID), nil
	case shared.ToolNameChangeDirectory:
		return f.changeDirectoryToolProv.CreateTool(agentID), nil
	case shared.ToolNameInvokeSkill:
		return f.invokeSkillToolProv.CreateTool(agentID, f), nil
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", name)
	}
}
