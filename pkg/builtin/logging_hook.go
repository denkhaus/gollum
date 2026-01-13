// Package builtin provides production-ready built-in hooks for common use cases.
// These hooks can be registered out-of-the-box and configured via environment variables.
package builtin

import (
	"context"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// LoggingHook provides structured logging for all hook events.
// It logs tool, agent, file, and LLM operations with configurable log levels.
type LoggingHook struct {
	log    logger.LoggerService
	config *config.HooksConfig
}

// NewLoggingHook creates a new LoggingHook instance.
func NewLoggingHook(injector do.Injector) (*LoggingHook, error) {
	log := do.MustInvoke[logger.LoggerService](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	return &LoggingHook{
		log:    log,
		config: cfg.GetHooksConfig(),
	}, nil
}

// NewLoggingHookProvider creates a LoggingHook provider for DI registration.
// This provider function registers the LoggingHook as a singleton in the DI container.
func NewLoggingHookProvider(injector do.Injector) (hooks.HookFunc, error) {
	hook, err := NewLoggingHook(injector)
	if err != nil {
		return nil, err
	}

	// Return a HookFunc that logs before and after operations
	return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
		return hook.logOperation(ctx, hookCtx, next)
	}, nil
}

// RegisterLoggingHooks registers all logging hooks with the HookManager.
// It registers hooks for all hook points (tool, agent, file, LLM).
func RegisterLoggingHooks(hm hooks.HookManager, hook *LoggingHook) error {
	if !hook.config.LoggingEnabled {
		return nil // Skip registration if logging is disabled
	}

	priority := 1000    // High priority to log early in the chain
	fatalError := false // Logging errors should not stop execution

	// Helper to register hook and log errors (non-fatal)
	register := func(fn hooks.HookFunc, meta hooks.HookMetadata) {
		if err := hm.RegisterHook(fn, meta); err != nil {
			// Log but don't fail - logging hooks are non-critical
			hook.log.Warn("Failed to register logging hook",
				zap.String("hook_name", meta.Name),
				zap.Error(err))
		}
	}

	// Tool execution hooks
	register(hook.createHookFunc(hooks.BeforeToolExecution, "before_tool"),
		hooks.HookMetadata{Name: "logging-before-tool", Point: hooks.BeforeToolExecution, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterToolExecution, "after_tool"),
		hooks.HookMetadata{Name: "logging-after-tool", Point: hooks.AfterToolExecution, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.OnToolError, "on_tool_error"),
		hooks.HookMetadata{Name: "logging-on-tool-error", Point: hooks.OnToolError, Priority: priority, FatalError: fatalError})

	// Agent lifecycle hooks
	register(hook.createHookFunc(hooks.BeforeAgentSpawn, "before_agent_spawn"),
		hooks.HookMetadata{Name: "logging-before-agent-spawn", Point: hooks.BeforeAgentSpawn, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterAgentSpawn, "after_agent_spawn"),
		hooks.HookMetadata{Name: "logging-after-agent-spawn", Point: hooks.AfterAgentSpawn, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.BeforeAgentRemove, "before_agent_remove"),
		hooks.HookMetadata{Name: "logging-before-agent-remove", Point: hooks.BeforeAgentRemove, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterAgentRemove, "after_agent_remove"),
		hooks.HookMetadata{Name: "logging-after-agent-remove", Point: hooks.AfterAgentRemove, Priority: priority, FatalError: fatalError})

	// File operation hooks
	register(hook.createHookFunc(hooks.BeforeFileRead, "before_file_read"),
		hooks.HookMetadata{Name: "logging-before-file-read", Point: hooks.BeforeFileRead, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterFileRead, "after_file_read"),
		hooks.HookMetadata{Name: "logging-after-file-read", Point: hooks.AfterFileRead, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.BeforeFileWrite, "before_file_write"),
		hooks.HookMetadata{Name: "logging-before-file-write", Point: hooks.BeforeFileWrite, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterFileWrite, "after_file_write"),
		hooks.HookMetadata{Name: "logging-after-file-write", Point: hooks.AfterFileWrite, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.BeforeFileDelete, "before_file_delete"),
		hooks.HookMetadata{Name: "logging-before-file-delete", Point: hooks.BeforeFileDelete, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterFileDelete, "after_file_delete"),
		hooks.HookMetadata{Name: "logging-after-file-delete", Point: hooks.AfterFileDelete, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.BeforeFileModify, "before_file_modify"),
		hooks.HookMetadata{Name: "logging-before-file-modify", Point: hooks.BeforeFileModify, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterFileModify, "after_file_modify"),
		hooks.HookMetadata{Name: "logging-after-file-modify", Point: hooks.AfterFileModify, Priority: priority, FatalError: fatalError})

	// LLM hooks
	register(hook.createHookFunc(hooks.BeforeLLMRequest, "before_llm_request"),
		hooks.HookMetadata{Name: "logging-before-llm-request", Point: hooks.BeforeLLMRequest, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.AfterLLMResponse, "after_llm_response"),
		hooks.HookMetadata{Name: "logging-after-llm-response", Point: hooks.AfterLLMResponse, Priority: priority, FatalError: fatalError})
	register(hook.createHookFunc(hooks.OnLLMError, "on_llm_error"),
		hooks.HookMetadata{Name: "logging-on-llm-error", Point: hooks.OnLLMError, Priority: priority, FatalError: fatalError})

	return nil
}

// logOperation logs the hook event and calls the next function in the chain.
func (h *LoggingHook) logOperation(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	// Call next first to allow other hooks to modify context
	err := next()

	// Log after execution with context data
	h.logWithContext(hookCtx, err)

	return err
}

// logWithContext logs the hook context with structured fields.
func (h *LoggingHook) logWithContext(hookCtx *hooks.HookContext, err error) {
	fields := []zap.Field{}

	// Add session and agent IDs
	if hookCtx.SessionID != uuid.Nil {
		fields = append(fields, zap.String("session_id", hookCtx.SessionID.String()))
	}
	if hookCtx.AgentID != uuid.Nil {
		fields = append(fields, zap.String("agent_id", hookCtx.AgentID.String()))
	}

	// Add tool information
	if hookCtx.ToolName != "" {
		fields = append(fields, zap.String("tool", hookCtx.ToolName))
		if len(hookCtx.ToolArgs) > 0 {
			fields = append(fields, zap.Any("tool_args", hookCtx.ToolArgs))
		}
		if len(hookCtx.ToolResult) > 0 {
			fields = append(fields, zap.Any("tool_result", hookCtx.ToolResult))
		}
	}

	// Add file information
	if hookCtx.FilePath != "" {
		fields = append(fields, zap.String("file_path", hookCtx.FilePath))
	}

	// Add LLM information
	if hookCtx.LLMModel != "" {
		fields = append(fields, zap.String("llm_model", hookCtx.LLMModel))
	}
	if hookCtx.LLMInput != "" {
		fields = append(fields, zap.String("llm_input", truncateString(hookCtx.LLMInput, 200)))
	}
	if hookCtx.LLMResponse != "" {
		fields = append(fields, zap.String("llm_response", truncateString(hookCtx.LLMResponse, 200)))
	}

	// Add error if present
	if err != nil {
		fields = append(fields, zap.Error(err))
		h.log.Error("Hook event", fields...)
	} else {
		h.log.Info("Hook event", fields...)
	}
}

// createHookFunc creates a hook function for a specific hook point.
func (h *LoggingHook) createHookFunc(_ hooks.HookPoint, _ string) hooks.HookFunc {
	return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
		return h.logOperation(ctx, hookCtx, next)
	}
}
