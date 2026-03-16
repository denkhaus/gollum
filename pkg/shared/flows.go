package shared

import "time"

// ErrorContext holds error lifecycle information
type ErrorContext struct {
	StepName  string
	StepType  string
	Message   string
	Timestamp time.Time
}

// FlowContext defines the interface for accessing flow execution state
type FlowContext interface {
	// SetOutputField sets an output field value
	SetOutputField(name string, value any) error
	// GetOutputField retrieves an output field value
	GetOutputField(name string) (any, error)
	// SetContextField sets a context field value
	SetContextField(name string, value any) error
	// GetContextField retrieves a context field value
	GetContextField(name string) (any, error)
	// GetCurrentState returns the current state name
	GetCurrentState() string
	// GetAllContextFields returns all context fields
	GetAllContextFields() map[string]any
	// ValidateTransition checks if a transition is allowed
	ValidateTransition(from, to string) error
	// RequestTransition signals that the flow should transition to the target state
	RequestTransition(to string) error
}

type ExecutionContext interface {
	SetContextField(name string, value any) error
	GetContextField(name string) (any, error)
	GetOutputField(name string) (any, error)
	SetOutputField(name string, value any) error
	GetComputedField(name string) (any, error)
	GetInput(name string) any
	EvaluateComputed() error
	SetError(ctx *ErrorContext)
	GetError() *ErrorContext
}

// FlowExecutorInstance defines the interface for a flow executor instance
type FlowExecutorInstance interface {
	// SetInput sets input field values
	SetInput(vals map[string]string)
	// Validate validates the flow before execution
	Validate() error
	// Run executes the flow from the initial state
	Run() error
	// GetContext returns the execution context (for testing)
	GetContext() ExecutionContext
}
