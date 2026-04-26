// Package hooks provides generic typed hook context and function types.
// These types provide type safety and better IDE support compared to the
// legacy untyped HookContext.
package hooks

import (
	"context"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
)

// TypedHookContext is a generic context that carries typed payload data for hook execution.
// The type parameter T represents the specific payload type for the hook category
// (e.g., ToolPayload, LLMPayload, FilePayload).
//
// This is the type-safe replacement for the legacy HookContext type.
// Migrate existing hooks to use TypedHookContext[T] for better type safety.
//
// Example usage:
//
//	func myToolHook(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
//	    // Access typed payload
//	    toolName := hookCtx.Payload.Name
//	    args := hookCtx.Payload.Args
//	    // ... process hook
//	    return next()
//	}
type TypedHookContext[T any] struct {
	// SessionContext provides common session and agent identification.
	shared.SessionContext

	// Payload contains the typed data specific to this hook category.
	Payload T

	// Tracing provides observability data for distributed tracing.
	// Always available regardless of payload type.
	Tracing TracingPayload
}

// TypedHookFunc is the generic type-safe signature for hook functions.
//
// The type parameter T matches the payload type in TypedHookContext[T].
//
// This is the type-safe replacement for the legacy HookFunc type.
// Migrate existing hooks to use TypedHookFunc[T] for better type safety.
//
// The function receives:
//   - ctx: context for cancellation/control
//   - hookCtx: typed contextual information about the event
//   - next: function to call the next hook in the chain
//
// The function should:
//   - Call next() to continue the chain (unless stopping intentionally)
//   - Return an error to stop execution (if FatalError=true in metadata)
//   - Modify hookCtx.Payload to pass data to subsequent hooks
//
// Example:
//
//	func myLLMHook(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
//	    // Modify the prompt before sending to LLM
//	    hookCtx.Payload.Input = "System: " + hookCtx.Payload.Input
//	    return next()
//	}
type TypedHookFunc[T any] func(ctx context.Context, hookCtx *TypedHookContext[T], next func() error) error

// TypedHookResult represents the result of a typed hook execution.
// It is the generic equivalent of HookResult.
type TypedHookResult[T any] struct {
	// Stopped indicates whether the hook chain was stopped.
	Stopped bool

	// Error contains the error from hook execution (if any).
	Error error

	// Payload contains the (potentially modified) payload after all hooks have run.
	Payload T
}

// TypedHookMetadata contains configuration for a typed hook.
// It is identical to HookMetadata but associated with typed hooks.
type TypedHookMetadata struct {
	// Name uniquely identifies this hook registration
	Name string

	// Point determines when this hook is triggered
	Point HookPoint

	// Priority determines execution order (0=first, higher=later)
	Priority int

	// FatalError controls error handling:
	// - true: stop execution and return error
	// - false: log error and continue to next hook
	FatalError bool
}

// NewTypedHookContext creates a new TypedHookContext with the given base context and payload.
// This is a convenience function for creating typed contexts in tests and implementations.
//
// Example:
//
//	ctx := hooks.NewTypedHookContext(
//	    hooks.BaseContext{SessionID: sessionID, AgentID: agentID},
//	    hooks.ToolPayload{Name: "myTool", Args: args},
//	)
func NewTypedHookContext[T any](loggingContext shared.SessionContext, payload T) *TypedHookContext[T] {
	return &TypedHookContext[T]{
		SessionContext: loggingContext,
		Payload:        payload,
		Tracing:        TracingPayload{},
	}
}

// NewTypedHookContextWithTracing creates a new TypedHookContext with tracing data.
// Use this when you need to provide tracing information from the start.
func NewTypedHookContextWithTracing[T any](loggingContext shared.SessionContext, payload T, tracing TracingPayload) *TypedHookContext[T] {
	return &TypedHookContext[T]{
		SessionContext: loggingContext,
		Payload:        payload,
		Tracing:        tracing,
	}
}

// Clone creates a deep copy of the TypedHookContext.
// Note: The payload T is shallow copied. If T contains reference types
// (maps, slices, pointers), they will share the same underlying data.
// Implement custom cloning if deep copy of payload is needed.
func (hc *TypedHookContext[T]) Clone() *TypedHookContext[T] {
	if hc == nil {
		var zero T
		return &TypedHookContext[T]{Payload: zero}
	}
	return &TypedHookContext[T]{
		SessionContext: hc.SessionContext,
		Payload:        hc.Payload,
		Tracing:        hc.Tracing,
	}
}

// WithSessionID returns a copy of the context with the session ID set.
// Useful for builder-style context construction.
func (hc *TypedHookContext[T]) WithSessionID(sessionID uuid.UUID) *TypedHookContext[T] {
	cpy := hc.Clone()
	cpy.SessionID = sessionID
	return cpy
}

// WithAgentID returns a copy of the context with the agent ID set.
// Useful for builder-style context construction.
func (hc *TypedHookContext[T]) WithAgentID(agentID uuid.UUID) *TypedHookContext[T] {
	cpy := hc.Clone()
	cpy.AgentID = agentID
	return cpy
}

// WithTracing returns a copy of the context with the tracing payload set.
// Useful for builder-style context construction.
func (hc *TypedHookContext[T]) WithTracing(tracing TracingPayload) *TypedHookContext[T] {
	cpy := hc.Clone()
	cpy.Tracing = tracing
	return cpy
}
