package hooks

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WithFlowStepHooks wraps a function with flow step execution hooks.
// BeforeFlowStep hooks can inspect/validate before execution.
// AfterFlowStep hooks can log/audit after execution.
func (p *hookManagerImpl) WithFlowStepHooks(
	ctx context.Context,
	sessionID, flowID uuid.UUID,
	flowName, stepType, stateName string,
	work func() (map[string]any, error),
) (map[string]any, error) {
	// Validate inputs immediately (fail fast)
	if work == nil {
		return nil, errs.Validation("work function cannot be nil")
	}

	startTime := time.Now()

	// Create initial payload for BeforeFlowStep
	beforePayload := ExecutorPayload{
		FlowID:       flowID,
		FlowName:     flowName,
		SessionID:    sessionID,
		CurrentState: stateName,
		StepType:     stepType,
		StateName:    stateName,
	}

	beforeCtx := NewTypedHookContext(
		shared.SessionContext{SessionID: sessionID, ChannelID: uuid.Nil, AgentID: uuid.Nil},
		beforePayload,
	)

	// Execute BeforeFlowStep hooks
	beforeResult := p.executorRegistry.Trigger(ctx, BeforeFlowStep, beforeCtx)
	if beforeResult.Stopped {
		return nil, errs.Validation("step blocked by BeforeFlowStep hook")
	}

	if beforeResult.Error != nil {
		return nil, beforeResult.Error
	}

	// Execute the actual step
	result, err := work()
	duration := time.Since(startTime)

	// Create payload for AfterFlowStep with execution results
	afterPayload := ExecutorPayload{
		FlowID:       flowID,
		FlowName:     flowName,
		SessionID:    sessionID,
		CurrentState: stateName,
		StepType:     stepType,
		StateName:    stateName,
		StepResult:   result,
		StepError:    err,
		Duration:     duration,
	}

	afterCtx := NewTypedHookContext(
		shared.SessionContext{SessionID: sessionID, ChannelID: uuid.Nil, AgentID: uuid.Nil},
		afterPayload,
	)

	// Execute AfterFlowStep hooks (always, even on error)
	afterResult := p.executorRegistry.Trigger(ctx, AfterFlowStep, afterCtx)
	if afterResult.Error != nil {
		// Log hook error but don't override the original work error
		p.log.WarnWithContext("AfterFlowStep hook error",
			shared.SessionContext{SessionID: sessionID, ChannelID: uuid.Nil, AgentID: uuid.Nil}, // No agentID in flow context
			zap.Error(afterResult.Error),
			zap.String("flow_id", flowID.String()),
			zap.String("flow_name", flowName),
			zap.String("state", stateName),
		)
	}

	return result, err
}
