package tools

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// ptrTo returns a pointer to the given UUID value
func ptrTo(id uuid.UUID) *uuid.UUID {
	return &id
}

// ContextMode determines how skill execution inherits or isolates context
type ContextMode string

const (
	// ContextModeInherited shares parent agent's message history
	ContextModeInherited ContextMode = "inherited"
	// ContextModeIsolated starts with a fresh context
	ContextModeIsolated ContextMode = "isolated"
)

// IsValid checks if the context mode is valid
func (m ContextMode) IsValid() bool {
	switch m {
	case ContextModeInherited, ContextModeIsolated:
		return true
	default:
		return false
	}
}

type (
	// invokeSkillToolImpl executes discovered skills at runtime
	invokeSkillToolImpl struct {
		logService      logger.LoggerService
		agentFactory    shared.AgentFactory
		registry        registry.AgentRegistry
		executionHelper AgentExecutionHelper
		hookManager     hooks.HookManager
		skillService    skills.SkillService
		agent           shared.Agent
	}

	// InvokeSkillToolProvider creates InvokeSkillTool instances via DI
	InvokeSkillToolProvider interface {
		CreateTool(agent shared.Agent, agentFactory shared.AgentFactory) gollem.Tool
	}

	invokeSkillToolProvider struct {
		logService      logger.LoggerService
		registry        registry.AgentRegistry
		executionHelper AgentExecutionHelper
		hookManager     hooks.HookManager
		skillService    skills.SkillService
	}
)

// NewInvokeSkillToolProvider creates a provider for InvokeSkill tools
func NewInvokeSkillToolProvider(injector do.Injector) (InvokeSkillToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)
	executionHelper := do.MustInvoke[AgentExecutionHelper](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	skillSvc := do.MustInvoke[skills.SkillService](injector)

	return &invokeSkillToolProvider{
		logService:      logService,
		registry:        registry,
		executionHelper: executionHelper,
		hookManager:     hookManager,
		skillService:    skillSvc,
	}, nil
}

// CreateTool creates a new InvokeSkillTool for a specific agent
func (p *invokeSkillToolProvider) CreateTool(agent shared.Agent, agentFactory shared.AgentFactory) gollem.Tool {
	return &invokeSkillToolImpl{
		logService:      p.logService,
		agentFactory:    agentFactory,
		registry:        p.registry,
		executionHelper: p.executionHelper,
		hookManager:     p.hookManager,
		skillService:    p.skillService,
		agent:           agent,
	}
}

// Spec returns the tool specification for InvokeSkillTool
func (t *invokeSkillToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name: shared.ToolNameInvokeSkill.String(),
		Description: `Executes a discovered skill by name. Skills can run as subagents with inherited or isolated context,
		or as simple template transformations. Use this to invoke specialized capabilities defined in SKILL.md files.`,
		Parameters: map[string]*gollem.Parameter{
			"name": {
				Type:        gollem.TypeString,
				Description: "The name of the skill to invoke (case-insensitive)",
			},
			"input": {
				Type:        gollem.TypeString,
				Description: "The input/task description for the skill to process",
			},
			"context_mode": {
				Type:        gollem.TypeString,
				Description: "Optional: Override skill's default context mode - 'inherited' (shares parent context) or 'isolated' (fresh context). Defaults to skill's configured mode.",
			},
			"model": {
				Type:        gollem.TypeString,
				Description: "Optional: Override skill's default model - 'sonnet', 'opus', or 'haiku'. Defaults to skill's configured model or parent's model.",
			},
		},
	}
}

// Run executes the InvokeSkill tool
func (t *invokeSkillToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameInvokeSkill, args,
		func() (map[string]any, error) {
			return t.runInvokeSkill(ctx, args)
		})
}

// runInvokeSkill implements the core skill invocation logic
func (t *invokeSkillToolImpl) runInvokeSkill(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Validate required parameters
	skillName, errResp := args.MustGetString(shared.ParamAgentName)
	if errResp != nil {
		return t.executionHelper.ErrorResponse("name is required and must be a non-empty string"), nil
	}

	input, errResp := args.MustGetString(shared.ParamInput)
	if errResp != nil {
		return t.executionHelper.ErrorResponse("input is required and must be a non-empty string"), nil
	}

	// Get the skill
	skill, err := t.skillService.Get(skillName)
	if err != nil {
		t.logService.ErrorWithContext("Skill not found", t.agent.ToLoggingContext(),
			zap.String("skill_name", skillName),
			zap.Error(err))
		return t.executionHelper.ErrorResponse(fmt.Sprintf("skill '%s' not found. Use list_skills to see available skills.", skillName)), nil
	}

	// Determine context mode (skill default or override)
	contextMode := ContextModeInherited // Default
	modeStr := args.GetString(shared.ParamContextMode, "")
	if modeStr != "" {
		mode := ContextMode(modeStr)
		if !mode.IsValid() {
			return t.executionHelper.ErrorResponse(fmt.Sprintf("invalid context_mode '%s'. Use 'inherited' or 'isolated'.", modeStr)), nil
		}
		contextMode = mode
	}

	// Determine model (parent's model or override)
	llmClientConfig := &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}
	if parentAgent, hasParent := t.registry.GetAgent(t.agent.GetID()); hasParent {
		llmClientConfig = parentAgent.GetConfig().LLMClientConfig
	}

	modelStr := args.GetString(shared.ParamModel, "")
	if modelStr != "" {
		switch modelStr {
		case "sonnet":
			llmClientConfig = &shared.LLMClientConfig{Model: "anthropic/claude-3-5-sonnet-20241022"}
		case "opus":
			llmClientConfig = &shared.LLMClientConfig{Model: "anthropic/claude-3-opus-20240229"}
		case "haiku":
			llmClientConfig = &shared.LLMClientConfig{Model: "anthropic/claude-3-5-haiku-20241022"}
		default:
			return t.executionHelper.ErrorResponse(fmt.Sprintf("invalid model '%s'. Use 'sonnet', 'opus', or 'haiku'.", modelStr)), nil
		}
	}

	// Build skill hook context with typed payload
	modelName, _ := llmClientConfig.ModelName()
	skillHookCtx := &hooks.TypedHookContext[hooks.SkillPayload]{
		LoggingContext: t.agent.ToLoggingContext(),
		Payload: hooks.SkillPayload{
			Name:        skill.Name,
			Type:        hooks.SkillType(skill.Type),
			ContextMode: hooks.SkillContextMode(contextMode),
			Model:       modelName,
			FilePath:    skill.FilePath,
			Version:     skill.Version,
		},
	}

	// Trigger BeforeSkillInvoked hooks
	beforeResult := t.hookManager.TriggerSkillHooks(ctx, hooks.BeforeSkillInvoked, skillHookCtx)
	if beforeResult.Stopped {
		t.logService.WarnWithContext("Skill invocation blocked by hook", t.agent.ToLoggingContext(),
			zap.Error(beforeResult.Error))
		return t.executionHelper.ErrorResponse(fmt.Sprintf("skill invocation blocked: %v", beforeResult.Error)), nil
	}

	t.logService.InfoWithContext("Invoking skill", t.agent.ToLoggingContext(),
		zap.String("skill_name", skill.Name),
		zap.String("context_mode", string(contextMode)),
		zap.String("model", modelName))

	// Get skill system prompt (content from SKILL.md)
	systemPrompt := skill.Content
	if systemPrompt == "" {
		return t.executionHelper.ErrorResponse(fmt.Sprintf("skill '%s' has no content defined", skillName)), nil
	}

	// Build the skill invocation prompt with input
	skillPrompt := fmt.Sprintf("%s\n\nTask: %s", systemPrompt, input)

	// Get parent agent for context inheritance
	parentAgent, hasParent := t.registry.GetAgent(t.agent.GetID())

	var history *gollem.History
	if contextMode == ContextModeInherited && hasParent {
		history, err = parentAgent.GetMessageHistory(ctx)
		if err != nil {
			t.logService.WarnWithContext("Failed to get message history from parent", t.agent.ToLoggingContext(), zap.Error(err))
			// Continue without history - non-fatal error
		}
	}

	// Create subagent configuration for skill execution
	taskID := uuid.New()
	subagentConfig := &shared.AgentConfig{
		AllowCompaction: false,
		ID:              taskID,
		ParentID:        ptrTo(t.agent.GetID()),
		SystemPrompt:    skillPrompt,
		Role:            fmt.Sprintf("Skill: %s", skill.Name),
		Description:     fmt.Sprintf("Executing skill: %s", skill.Name),
		LLMClientConfig: llmClientConfig,
		OutputMode:      shared.OutputModeSummary,
		History:         history,
	}

	// Apply skill's tool restrictions if specified
	if len(skill.Tools) > 0 || skill.ToolScope != skills.ToolScopeAll {
		// For now, we pass all default tools. Tool filtering could be implemented
		// by modifying the agent factory or adding tool filtering logic here.
		t.logService.DebugWithContext("Skill has tool restrictions", t.agent.ToLoggingContext(),
			zap.Any("tools", skill.Tools),
			zap.String("scope", string(skill.ToolScope)))
	}

	// Create the subagent using the factory
	subagent, err := t.agentFactory.CreateAgent(ctx, subagentConfig)
	if err != nil {
		t.logService.ErrorWithContext("Failed to create skill subagent", t.agent.ToLoggingContext(), zap.Error(err))
		// Trigger OnSkillError hooks
		skillHookCtx.Payload.Error = err
		t.hookManager.TriggerSkillHooks(ctx, hooks.OnSkillError, skillHookCtx)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to create skill subagent: %v", err)), nil
	}

	t.logService.InfoWithContext("Created skill subagent", t.agent.ToLoggingContext(),
		zap.String("subagent_id", subagent.GetID().String()),
		zap.String("skill_name", skill.Name))

	// Register and execute synchronously
	if err := t.registry.Register(subagent, subagentConfig); err != nil {
		t.logService.ErrorWithContext("Failed to register skill subagent", t.agent.ToLoggingContext(), zap.Error(err))
		// Trigger OnSkillError hooks
		skillHookCtx.Payload.Error = err
		t.hookManager.TriggerSkillHooks(ctx, hooks.OnSkillError, skillHookCtx)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("failed to register skill subagent: %v", err)), nil
	}

	// Execute the skill
	response, err := t.executionHelper.ExecuteSynchronously(ctx, subagent, input)
	if err != nil {
		t.logService.ErrorWithContext("Skill execution failed", t.agent.ToLoggingContext(),
			zap.String("skill_name", skill.Name),
			zap.Error(err))
		// Trigger OnSkillError hooks
		skillHookCtx.Payload.Error = err
		t.hookManager.TriggerSkillHooks(ctx, hooks.OnSkillError, skillHookCtx)
		return t.executionHelper.ErrorResponse(fmt.Sprintf("skill execution failed: %v", err)), nil
	}

	t.logService.InfoWithContext("Skill completed successfully", t.agent.ToLoggingContext(),
		zap.String("skill_name", skill.Name))

	// Add skill metadata to response
	if response == nil {
		response = make(map[string]any)
	}
	response["skill_name"] = skill.Name
	response["skill_type"] = string(skill.Type)
	response["context_mode"] = string(contextMode)

	// Update hook context with result and trigger AfterSkillInvoked hooks
	skillHookCtx.Payload.Result = response
	t.hookManager.TriggerSkillHooks(ctx, hooks.AfterSkillInvoked, skillHookCtx)

	return response, nil
}
