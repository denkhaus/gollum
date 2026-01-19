package hooks

import (
	"context"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WithLLMHooks wraps an LLM API call with hooks for request, response, and error handling.
//
// The workflow is:
// 1. BeforeLLMRequest hooks run with prompt in HookContext.LLMInput
//   - Hooks can modify the prompt via HookContext.LLMInput
//   - Hooks can block execution by not calling next()
//
// 2. LLM API call (work function) runs with potentially modified prompt
//
// 3. If LLM succeeds, AfterLLMResponse hooks run with response in HookContext.LLMResponse
//   - Hooks can modify the response via HookContext.LLMResponse
//
// 4. If LLM fails, OnLLMError hooks run with error in HookContext.LLMError
//   - Hooks can recover by setting HookContext.LLMResponse to a fallback response
//   - Or hooks can allow the error to propagate
func (p *hookManagerImpl) WithLLMHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	prompt string,
	model string,
	work func(string) (string, error),
) (string, error) {
	// Validate inputs immediately (fail fast)
	if work == nil {
		return "", errs.Validation("work function cannot be nil")
	}

	// BeforeLLMRequest hook
	hookCtx := &HookContext{
		SessionID:  sessionID,
		AgentID:    agentID,
		LLMInput:   prompt,
		LLMModel:   model,
		LLMOptions: make(map[string]any),
		Data:       make(map[string]any),
	}

	result := p.TriggerHooks(ctx, BeforeLLMRequest, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return "", result.Error
		}
		// Hook stopped without error - blocked successfully
		// Check if hook provided a modified response
		if hookCtx.LLMResponse != "" {
			return hookCtx.LLMResponse, nil
		}
		return "", nil
	}

	// Execute the LLM work with potentially modified prompt
	finalPrompt := hookCtx.LLMInput
	response, workErr := work(finalPrompt)

	// Store response in context for after hooks
	hookCtx.LLMResponse = response

	if workErr != nil {
		// OnLLMError hook
		hookCtx.LLMError = workErr
		errorResult := p.TriggerHooks(ctx, OnLLMError, hookCtx)

		// If hooks provided a fallback response, use it
		if hookCtx.LLMResponse != "" {
			p.log.Debug("LLM error recovered by hook",
				zap.String("model", model),
				zap.Error(workErr))
			return hookCtx.LLMResponse, nil
		}

		// If error hook had fatal error, return that
		if errorResult.Error != nil {
			return "", errorResult.Error
		}

		// Otherwise return the original LLM error
		return "", workErr
	}

	// AfterLLMResponse hook
	hookCtx.LLMError = nil

	afterResult := p.TriggerHooks(ctx, AfterLLMResponse, hookCtx)
	if afterResult.Stopped && afterResult.Error != nil {
		return "", afterResult.Error
	}

	// Return potentially modified response from hooks
	if hookCtx.LLMResponse != "" {
		return hookCtx.LLMResponse, nil
	}

	// response should never be empty here, but handle defensively
	if response == "" {
		response = hookCtx.LLMInput // Return original prompt as fallback
	}
	return response, nil
}
