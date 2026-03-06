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
	_, err := NewLoggingHook(injector)
	if err != nil {
		return nil, err
	}

	// Return a no-op HookFunc - actual hooks registered via RegisterLoggingHooks
	return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
		return next()
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

	// Helper to create TypedHookMetadata
	meta := func(name string, point hooks.HookPoint) hooks.TypedHookMetadata {
		return hooks.TypedHookMetadata{
			Name:       name,
			Point:      point,
			Priority:   priority,
			FatalError: fatalError,
		}
	}

	// Helper to register hook and log errors (non-fatal)
	registerTool := func(fn hooks.TypedHookFunc[hooks.ToolPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterToolHook(fn, meta(name, point)); err != nil {
			// Log but don't fail - logging hooks are non-critical
			hook.log.Warn("Failed to register logging hook",
				zap.String("hook_name", name),
				zap.Error(err))
		}
	}

	registerAgent := func(fn hooks.TypedHookFunc[hooks.AgentPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterAgentHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register logging hook",
				zap.String("hook_name", name),
				zap.Error(err))
		}
	}

	registerFile := func(fn hooks.TypedHookFunc[hooks.FilePayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterFileHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register logging hook",
				zap.String("hook_name", name),
				zap.Error(err))
		}
	}

	registerLLM := func(fn hooks.TypedHookFunc[hooks.LLMPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterLLMHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register logging hook",
				zap.String("hook_name", name),
				zap.Error(err))
		}
	}

	// Tool execution hooks
	registerTool(hook.beforeToolExecutionHook, "logging-before-tool", hooks.BeforeToolExecution)
	registerTool(hook.afterToolExecutionHook, "logging-after-tool", hooks.AfterToolExecution)
	registerTool(hook.onToolErrorHook, "logging-on-tool-error", hooks.OnToolError)

	// Agent lifecycle hooks
	registerAgent(hook.beforeAgentSpawnHook, "logging-before-agent-spawn", hooks.BeforeAgentSpawn)
	registerAgent(hook.afterAgentSpawnHook, "logging-after-agent-spawn", hooks.AfterAgentSpawn)
	registerAgent(hook.beforeAgentRemoveHook, "logging-before-agent-remove", hooks.BeforeAgentRemove)
	registerAgent(hook.afterAgentRemoveHook, "logging-after-agent-remove", hooks.AfterAgentRemove)

	// File operation hooks
	registerFile(hook.beforeFileReadHook, "logging-before-file-read", hooks.BeforeFileRead)
	registerFile(hook.afterFileReadHook, "logging-after-file-read", hooks.AfterFileRead)
	registerFile(hook.beforeFileWriteHook, "logging-before-file-write", hooks.BeforeFileWrite)
	registerFile(hook.afterFileWriteHook, "logging-after-file-write", hooks.AfterFileWrite)
	registerFile(hook.beforeFileDeleteHook, "logging-before-file-delete", hooks.BeforeFileDelete)
	registerFile(hook.afterFileDeleteHook, "logging-after-file-delete", hooks.AfterFileDelete)
	registerFile(hook.beforeFileModifyHook, "logging-before-file-modify", hooks.BeforeFileModify)
	registerFile(hook.afterFileModifyHook, "logging-after-file-modify", hooks.AfterFileModify)

	// LLM hooks
	registerLLM(hook.beforeLLMRequestHook, "logging-before-llm-request", hooks.BeforeLLMRequest)
	registerLLM(hook.afterLLMResponseHook, "logging-after-llm-response", hooks.AfterLLMResponse)
	registerLLM(hook.onLLMErrorHook, "logging-on-llm-error", hooks.OnLLMError)

	return nil
}

// Tool execution hook methods
func (h *LoggingHook) beforeToolExecutionHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	err := next()
	h.logToolEvent(hookCtx, "before_tool", err)
	return err
}

func (h *LoggingHook) afterToolExecutionHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	err := next()
	h.logToolEvent(hookCtx, "after_tool", err)
	return err
}

func (h *LoggingHook) onToolErrorHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	err := next()
	h.logToolEvent(hookCtx, "tool_error", err)
	return err
}

// logToolEvent logs tool execution events with structured fields.
func (h *LoggingHook) logToolEvent(hookCtx *hooks.TypedHookContext[hooks.ToolPayload], event string, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("tool", hookCtx.Payload.Name.String()),
	}
	if hookCtx.SessionID != uuid.Nil {
		fields = append(fields, zap.String("session_id", hookCtx.SessionID.String()))
	}
	if hookCtx.AgentID != uuid.Nil {
		fields = append(fields, zap.String("agent_id", hookCtx.AgentID.String()))
	}
	if len(hookCtx.Payload.Args) > 0 {
		fields = append(fields, zap.Any("tool_args", hookCtx.Payload.Args))
	}
	if len(hookCtx.Payload.Result) > 0 {
		fields = append(fields, zap.Any("tool_result", hookCtx.Payload.Result))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		h.log.Error("Tool hook event", fields...)
	} else {
		h.log.Info("Tool hook event", fields...)
	}
}

// Agent lifecycle hook methods
func (h *LoggingHook) beforeAgentSpawnHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	err := next()
	h.logAgentEvent(hookCtx, "before_agent_spawn", err)
	return err
}

func (h *LoggingHook) afterAgentSpawnHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	err := next()
	h.logAgentEvent(hookCtx, "after_agent_spawn", err)
	return err
}

func (h *LoggingHook) beforeAgentRemoveHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	err := next()
	h.logAgentEvent(hookCtx, "before_agent_remove", err)
	return err
}

func (h *LoggingHook) afterAgentRemoveHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	err := next()
	h.logAgentEvent(hookCtx, "after_agent_remove", err)
	return err
}

// logAgentEvent logs agent lifecycle events with structured fields.
func (h *LoggingHook) logAgentEvent(hookCtx *hooks.TypedHookContext[hooks.AgentPayload], event string, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("agent_event", string(hookCtx.Payload.Event)),
	}
	if hookCtx.SessionID != uuid.Nil {
		fields = append(fields, zap.String("session_id", hookCtx.SessionID.String()))
	}
	if hookCtx.AgentID != uuid.Nil {
		fields = append(fields, zap.String("agent_id", hookCtx.AgentID.String()))
	}
	if hookCtx.Payload.NewAgentID != uuid.Nil {
		fields = append(fields, zap.String("new_agent_id", hookCtx.Payload.NewAgentID.String()))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		h.log.Error("Agent hook event", fields...)
	} else {
		h.log.Info("Agent hook event", fields...)
	}
}

// File operation hook methods
func (h *LoggingHook) beforeFileReadHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "before_file_read", err)
	return err
}

func (h *LoggingHook) afterFileReadHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "after_file_read", err)
	return err
}

func (h *LoggingHook) beforeFileWriteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "before_file_write", err)
	return err
}

func (h *LoggingHook) afterFileWriteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "after_file_write", err)
	return err
}

func (h *LoggingHook) beforeFileDeleteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "before_file_delete", err)
	return err
}

func (h *LoggingHook) afterFileDeleteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "after_file_delete", err)
	return err
}

func (h *LoggingHook) beforeFileModifyHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "before_file_modify", err)
	return err
}

func (h *LoggingHook) afterFileModifyHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	err := next()
	h.logFileEvent(hookCtx, "after_file_modify", err)
	return err
}

// logFileEvent logs file operation events with structured fields.
func (h *LoggingHook) logFileEvent(hookCtx *hooks.TypedHookContext[hooks.FilePayload], event string, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("file_path", hookCtx.Payload.Path),
		zap.String("operation", string(hookCtx.Payload.Operation)),
	}
	if hookCtx.SessionID != uuid.Nil {
		fields = append(fields, zap.String("session_id", hookCtx.SessionID.String()))
	}
	if hookCtx.AgentID != uuid.Nil {
		fields = append(fields, zap.String("agent_id", hookCtx.AgentID.String()))
	}
	if hookCtx.Payload.Content != "" {
		fields = append(fields, zap.Int("content_length", len(hookCtx.Payload.Content)))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		h.log.Error("File hook event", fields...)
	} else {
		h.log.Info("File hook event", fields...)
	}
}

// LLM hook methods
func (h *LoggingHook) beforeLLMRequestHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	err := next()
	h.logLLMEvent(hookCtx, "before_llm_request", err)
	return err
}

func (h *LoggingHook) afterLLMResponseHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	err := next()
	h.logLLMEvent(hookCtx, "after_llm_response", err)
	return err
}

func (h *LoggingHook) onLLMErrorHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	err := next()
	h.logLLMEvent(hookCtx, "llm_error", err)
	return err
}

// logLLMEvent logs LLM events with structured fields.
func (h *LoggingHook) logLLMEvent(hookCtx *hooks.TypedHookContext[hooks.LLMPayload], event string, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("model", hookCtx.Payload.Model),
	}
	if hookCtx.SessionID != uuid.Nil {
		fields = append(fields, zap.String("session_id", hookCtx.SessionID.String()))
	}
	if hookCtx.AgentID != uuid.Nil {
		fields = append(fields, zap.String("agent_id", hookCtx.AgentID.String()))
	}
	if hookCtx.Payload.Input != "" {
		fields = append(fields, zap.String("llm_input", truncateString(hookCtx.Payload.Input, 200)))
	}
	if hookCtx.Payload.Response != "" {
		fields = append(fields, zap.String("llm_response", truncateString(hookCtx.Payload.Response, 200)))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		h.log.Error("LLM hook event", fields...)
	} else {
		h.log.Info("LLM hook event", fields...)
	}
}
