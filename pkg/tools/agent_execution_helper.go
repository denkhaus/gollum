package tools

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// AgentExecutionHelper provides common agent execution functionality for tools
type AgentExecutionHelper interface {
	// ExecuteSynchronously runs an agent and waits for completion
	ExecuteSynchronously(ctx context.Context, agent shared.Agent, prompt string) (map[string]any, error)

	// ExecuteInBackground runs an agent asynchronously
	ExecuteInBackground(ctx context.Context, agent shared.Agent, prompt string)

	// SuccessResponseSync creates a success response for synchronous execution
	SuccessResponseSync(agentID uuid.UUID, response string) map[string]any

	// SuccessResponseAsync creates a success response for asynchronous execution
	SuccessResponseAsync(agentID uuid.UUID, role, description string) map[string]any

	// SuccessResponseResumeAsync creates a success response for resume asynchronous execution
	SuccessResponseResumeAsync(agentID uuid.UUID, role, description string) map[string]any

	// ErrorResponse creates an error response
	ErrorResponse(errMsg string) map[string]any
}

type agentExecutionHelper struct {
	logService logger.LoggerService
	registry   registry.AgentRegistry
}

// NewAgentExecutionHelper creates a new AgentExecutionHelper
func NewAgentExecutionHelper(injector do.Injector) (AgentExecutionHelper, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &agentExecutionHelper{
		logService: logService,
		registry:   registry,
	}, nil
}

// ExecuteSynchronously runs the agent and waits for completion
func (h *agentExecutionHelper) ExecuteSynchronously(ctx context.Context, agent shared.Agent, prompt string) (map[string]any, error) {
	start := time.Now()
	agentID := agent.GetID()
	h.logService.DebugWithContext("Starting synchronous execution", agent.ToSessionContext())

	// Execute the prompt
	response, err := agent.Execute(ctx, gollem.Text(prompt))
	if err != nil {
		duration := time.Since(start)
		h.logService.ErrorWithContext("Agent execution failed", agent.ToSessionContext(),
			zap.Duration("duration", duration),
			zap.Error(err))

		// Update agent result with error
		now := time.Now().Unix()
		if storeErr := h.registry.StoreAgentResult(shared.AgentResult{
			AgentID:     agent.GetID(),
			Status:      shared.AgentStatusFailed,
			Error:       err.Error(),
			CompletedAt: &now,
		}); storeErr != nil {
			h.logService.WarnWithContext("Failed to store error agent result", agent.ToSessionContext(),
				zap.Error(storeErr))
		}

		return nil, errs.Wrap(err, errs.TypeInternal, "execution failed").
			WithContext("agent_id", agentID)
	}

	// Extract response content
	var responseContent string
	if response != nil && len(response.Texts) > 0 {
		for _, text := range response.Texts {
			responseContent += text
		}
	}

	duration := time.Since(start)
	h.logService.DebugWithContext("Agent execution completed", agent.ToSessionContext(),
		zap.Duration("duration", duration))

	// Update agent result with completion
	now := time.Now().Unix()
	if storeErr := h.registry.StoreAgentResult(shared.AgentResult{
		AgentID:     agent.GetID(),
		Status:      shared.AgentStatusCompleted,
		Output:      map[string]any{"response": responseContent},
		CompletedAt: &now,
	}); storeErr != nil {
		h.logService.WarnWithContext("Failed to store completion agent result", agent.ToSessionContext(),
			zap.Error(storeErr))
	}

	// Return successful response
	return h.SuccessResponseSync(agent.GetID(), responseContent), nil
}

// ExecuteInBackground runs the agent asynchronously
func (h *agentExecutionHelper) ExecuteInBackground(ctx context.Context, agent shared.Agent, prompt string) {
	start := time.Now()
	h.logService.DebugWithContext("Starting background execution", agent.ToSessionContext())

	response, err := agent.Execute(ctx, gollem.Text(prompt))
	duration := time.Since(start)
	now := time.Now().Unix()

	// Check if execution was cancelled
	if ctx.Err() != nil {
		h.logService.InfoWithContext("Background agent was cancelled", agent.ToSessionContext(),
			zap.Duration("duration", duration))
		_ = h.registry.StoreAgentResult(shared.AgentResult{
			AgentID:     agent.GetID(),
			Status:      shared.AgentStatusFailed,
			Error:       "agent was cancelled",
			CompletedAt: &now,
		})
		return
	}

	if err != nil {
		h.logService.ErrorWithContext("Background agent failed", agent.ToSessionContext(),
			zap.Duration("duration", duration),
			zap.Error(err))
		_ = h.registry.StoreAgentResult(shared.AgentResult{
			AgentID:     agent.GetID(),
			Status:      shared.AgentStatusFailed,
			Error:       err.Error(),
			CompletedAt: &now,
		})
		return
	}

	// Extract response content
	var responseContent string
	if response != nil && len(response.Texts) > 0 {
		for _, text := range response.Texts {
			responseContent += text
		}
	}

	h.logService.InfoWithContext("Background agent completed", agent.ToSessionContext(),
		zap.Duration("duration", duration))

	_ = h.registry.StoreAgentResult(shared.AgentResult{
		AgentID:     agent.GetID(),
		Status:      shared.AgentStatusCompleted,
		Output:      map[string]any{"response": responseContent},
		CompletedAt: &now,
	})
}

// SuccessResponseSync creates a success response for synchronous execution
func (h *agentExecutionHelper) SuccessResponseSync(agentID uuid.UUID, response string) map[string]any {
	return map[string]any{
		"success":  true,
		"agent_id": agentID.String(),
		"response": response,
		"status":   "completed",
		"message":  "Agent completed successfully",
	}
}

// SuccessResponseAsync creates a success response for asynchronous execution
func (h *agentExecutionHelper) SuccessResponseAsync(agentID uuid.UUID, role, description string) map[string]any {
	return map[string]any{
		"success":     true,
		"agent_id":    agentID.String(),
		"role":        role,
		"description": description,
		"status":      "running",
		"message":     "Agent started in background. Use AgentOutputTool to retrieve results.",
	}
}

// SuccessResponseResumeAsync creates a success response for resume asynchronous execution
func (h *agentExecutionHelper) SuccessResponseResumeAsync(agentID uuid.UUID, role, description string) map[string]any {
	return map[string]any{
		"success":     true,
		"agent_id":    agentID.String(),
		"role":        role,
		"description": description,
		"status":      "running",
		"message":     "Agent resumed in background. Use AgentOutputTool to retrieve results.",
	}
}

// ErrorResponse creates an error response
func (h *agentExecutionHelper) ErrorResponse(errMsg string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   errMsg,
	}
}
