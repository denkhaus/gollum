package agents

import (
	"context"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/middleware/compacter"
	"github.com/m-mizutani/gollem/strategy/simple"
	"github.com/samber/do/v2"
)

// defaultAgentFactory implements the AgentFactory interface
type defaultAgentFactory struct {
	logService      logger.LoggerService
	configService   config.ConfigService
	clientProvider  llm.ClientProvider
	registry        registry.AgentRegistry
	promptManager   manager.PromptManager
	channelProvider channel.ChannelMiddlewareProvider
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
}

// NewAgentFactory creates a new AgentFactory
func NewAgentFactory(injector do.Injector) (shared.AgentFactory, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	configService := do.MustInvoke[config.ConfigService](injector)
	clientProvider := do.MustInvoke[llm.ClientProvider](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)
	channelProvider := do.MustInvoke[channel.ChannelMiddlewareProvider](injector)

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
		configService:           configService,
		clientProvider:          clientProvider,
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
	}, nil
}

// CreateAgent creates a new agent with the given configuration
func (f *defaultAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
	// Ensure the config has an ID, generate one if not set
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	// Add default tools to all agents (both root and sub-agents)
	// This ensures subagents created by SpawnAgentTool have access to tools
	defaultTools := []gollem.Tool{
		f.spawnAgentToolProv.CreateTool(config.ID, f),
		f.agentOutputToolProv.CreateTool(config.ID),
		f.removeAgentToolProv.CreateTool(config.ID),
		f.resumeAgentToolProv.CreateTool(config.ID),
		f.listAgentsToolProv.CreateTool(config.ID),
		f.currentTimeToolProv.CreateTool(config.ID),
		f.bashToolProv.CreateTool(config.ID),
		f.writeFileToolProv.CreateTool(config.ID),
		f.readFileToolProv.CreateTool(config.ID),
		f.globToolProv.CreateTool(config.ID),
		f.grepToolProv.CreateTool(config.ID),
		f.editToolProv.CreateTool(config.ID),
		f.sessionLogsToolProv.CreateTool(config.ID),
		f.changeDirectoryToolProv.CreateTool(config.ID),
		f.invokeSkillToolProv.CreateTool(config.ID, f),
	}
	config.Tools = append(config.Tools, defaultTools...)

	// Create the base agent
	defAgent := &defaultAgent{
		clientProvider: f.clientProvider,
		configService:  f.configService,
		logService:     f.logService,
		registry:       f.registry,
		id:             config.ID,
		config:         config,
		promptManager:  f.promptManager,
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
		gollem.WithTools(config.Tools...),
		gollem.WithToolSets(config.ToolSets...),
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
