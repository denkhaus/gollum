package executor

import (
	"time"
)

// ExecutionHistory tracks flow execution
type ExecutionHistory struct {
	states    []StateExecution
	errors    []ErrorRecord
	startTime time.Time
	endTime   time.Time
}

// StateExecution tracks a single state execution
type StateExecution struct {
	Name      string
	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
}

// ErrorRecord tracks an execution error
type ErrorRecord struct {
	StepName  string
	StepType  string
	Message   string
	Timestamp time.Time
}

// NewExecutionHistory creates a new execution history
func NewExecutionHistory() *ExecutionHistory {
	return &ExecutionHistory{
		states:    make([]StateExecution, 0),
		errors:    make([]ErrorRecord, 0),
		startTime: time.Now(),
	}
}

// RecordStateEntry records entering a state
func (h *ExecutionHistory) RecordStateEntry(name string, t time.Time) {
	h.states = append(h.states, StateExecution{
		Name:      name,
		StartedAt: t,
	})
}

// RecordStateExit records exiting a state
func (h *ExecutionHistory) RecordStateExit(name string, t time.Time) {
	for i := len(h.states) - 1; i >= 0; i-- {
		if h.states[i].Name == name && h.states[i].EndedAt.IsZero() {
			h.states[i].EndedAt = t
			h.states[i].Duration = t.Sub(h.states[i].StartedAt)
			break
		}
	}
}

// RecordError records an error
func (h *ExecutionHistory) RecordError(stepName, stepType, message string, t time.Time) {
	h.errors = append(h.errors, ErrorRecord{
		StepName:  stepName,
		StepType:  stepType,
		Message:   message,
		Timestamp: t,
	})
}

// GetStates returns all state executions
func (h *ExecutionHistory) GetStates() []StateExecution {
	return h.states
}

// GetErrors returns all errors
func (h *ExecutionHistory) GetErrors() []ErrorRecord {
	return h.errors
}

// Complete marks execution as complete
func (h *ExecutionHistory) Complete(t time.Time) {
	h.endTime = t
}

// GetDuration returns total execution duration
func (h *ExecutionHistory) GetDuration() time.Duration {
	if h.endTime.IsZero() {
		return time.Since(h.startTime)
	}
	return h.endTime.Sub(h.startTime)
}
