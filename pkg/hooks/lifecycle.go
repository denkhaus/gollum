package hooks

import (
	"context"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/shared"
	"go.uber.org/zap"
)

// WithSessionHooks wraps a function with session lifecycle hooks.
func (p *hookManagerImpl) WithSessionHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
	work func() error,
) error {
	if !loggingContext.IsValid() {
		return errs.Validation("logging context is invalid")
	}

	// BeforeSessionStart hook with typed context
	hookCtx := NewTypedHookContext(loggingContext,
		SessionPayload{Metadata: make(map[string]any)},
	)

	result := p.TriggerSessionHooks(ctx, BeforeSessionStart, hookCtx)
	if result.Stopped {
		return result.Error
	}

	// Execute the work
	workErr := work()

	// AfterSessionEnd hook
	// Always trigger AfterSessionEnd, even if work failed, to ensure cleanup runs.
	result = p.TriggerSessionHooks(ctx, AfterSessionEnd, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence.
	if result.Error != nil {
		if workErr != nil {
			p.log.ErrorWithContext("The original work function also returned an error, which is being superseded by the AfterSessionEnd hook error",
				loggingContext,
				zap.Error(workErr))
		}
		return result.Error
	}

	// Otherwise, return the error from the original work function (if any).
	return workErr
}

// WithAgentHooks wraps a function with agent lifecycle hooks.
func (p *hookManagerImpl) WithAgentHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
	point HookPoint,
	work func() error,
) error {
	if !loggingContext.IsValid() {
		return errs.Validation("logging context is invalid")
	}

	// Validate hook point and determine agent event
	var event AgentEvent
	switch point {
	case BeforeAgentSpawn, AfterAgentSpawn:
		event = AgentEventSpawn
	case BeforeAgentRemove, AfterAgentRemove:
		event = AgentEventRemove
	default:
		return errs.Validationf("invalid agent hook point: %s", point)
	}

	// Create typed context
	hookCtx := NewTypedHookContext(loggingContext,
		AgentPayload{Event: event, NewAgentID: loggingContext.AgentID},
	)

	result := p.TriggerAgentHooks(ctx, point, hookCtx)
	if result.Stopped {
		return result.Error
	}

	// Execute work if provided
	if work != nil {
		return work()
	}

	return nil
}
