package hooks

import (
	"context"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/shared"
	"go.uber.org/zap"
)

// WithToolHooks wraps a function with tool execution hooks.
//
// The workflow is:
// 1. BeforeToolExecution hooks run with args in TypedHookContext[ToolPayload]
//   - Hooks can modify args via Payload.Args
//   - Hooks can block execution by not calling next()
//
// 2. Tool execution (work function) runs
// 3. If tool succeeds, AfterToolExecution hooks run with result
//   - Hooks can modify result via Payload.Result
//
// 4. If tool fails, OnToolError hooks run with error
//   - Hooks can recover by returning a new result
//   - Or hooks can allow the error to propagate
func (p *hookManagerImpl) WithToolHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
	toolName shared.ToolName,
	args map[string]any,
	work func() (map[string]any, error),
) (map[string]any, error) {
	// Validate inputs immediately (fail fast)
	if toolName == "" {
		return nil, errs.Validation("tool name cannot be empty")
	}
	if work == nil {
		return nil, errs.Validation("work function cannot be nil")
	}

	// Make a copy of args to avoid modifying the original
	argsCopy := make(map[string]any, len(args))
	for k, v := range args {
		argsCopy[k] = v
	}

	// BeforeToolExecution hook with typed context
	hookCtx := NewTypedHookContext(loggingContext,
		ToolPayload{Name: toolName, Args: argsCopy},
	)

	result := p.TriggerToolHooks(ctx, BeforeToolExecution, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return nil, result.Error
		}
		// Hook stopped without error - return Result if set, or empty result
		if hookCtx.Payload.Result != nil {
			return hookCtx.Payload.Result, nil
		}
		return make(map[string]any), nil
	}

	// Note: Modified args are not passed to work() due to design limitations.
	// Hooks can validate/block execution but cannot modify what work() receives.
	// The work function closes over the original args parameter.
	_ = hookCtx.Payload.Args // Explicitly document we're not using modified args

	// Execute the tool work
	toolResult, workErr := work()

	// Check if BeforeToolExecution hooks set a modified result
	if hookCtx.Payload.Result != nil {
		toolResult = hookCtx.Payload.Result
		workErr = nil
	}

	if workErr != nil {
		// OnToolError hook
		hookCtx.Payload.Error = workErr
		errorResult := p.TriggerToolHooks(ctx, OnToolError, hookCtx)

		// If hooks provided a fallback result, use it
		if hookCtx.Payload.Result != nil {
			p.log.DebugWithContext("Tool error recovered by hook",
				loggingContext,
				zap.String("tool", toolName.String()),
				zap.Error(workErr),
			)
			return hookCtx.Payload.Result, nil
		}

		// If error hook had fatal error, return that
		if errorResult.Error != nil {
			return nil, errorResult.Error
		}

		// Otherwise return the original tool error
		return nil, workErr
	}

	// AfterToolExecution hook
	hookCtx.Payload.Result = toolResult
	hookCtx.Payload.Error = nil

	afterResult := p.TriggerToolHooks(ctx, AfterToolExecution, hookCtx)
	if afterResult.Stopped && afterResult.Error != nil {
		return nil, afterResult.Error
	}

	// Return potentially modified result from hooks
	if hookCtx.Payload.Result != nil {
		return hookCtx.Payload.Result, nil
	}

	// toolResult should never be nil here, but handle defensively
	if toolResult == nil {
		toolResult = make(map[string]any)
	}
	return toolResult, nil
}
