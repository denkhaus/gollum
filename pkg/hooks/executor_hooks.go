package hooks

import (
	"time"

	"github.com/google/uuid"
)

// ExecutorPayload contains flow execution context for step hooks
type ExecutorPayload struct {
	// Flow identification
	FlowID    uuid.UUID
	FlowName  string
	SessionID uuid.UUID

	// Current execution state
	CurrentState string

	// Step information
	StepType  string // "llm", "shell", "func", "mcp"
	StepIndex int
	StateName string

	// Execution metadata (AfterFlowStep only)
	StepResult map[string]any
	StepError  error
	Duration   time.Duration
}

// Hook points for executor
const (
	// BeforeFlowStep is triggered before any step in a flow executes
	BeforeFlowStep HookPoint = "BeforeFlowStep"

	// AfterFlowStep is triggered after any step in a flow completes
	AfterFlowStep HookPoint = "AfterFlowStep"
)
