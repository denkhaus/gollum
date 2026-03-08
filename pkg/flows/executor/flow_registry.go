package executor

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

// FlowRegistry is the interface for looking up flows by reference
type FlowRegistry interface {
	// GetFlow retrieves a flow by reference name
	GetFlow(ref string) (*flows.Flow, error)
}

// SimpleFlowRegistry is an in-memory implementation of FlowRegistry for testing
type SimpleFlowRegistry struct {
	flows map[string]*flows.Flow
}

// NewSimpleFlowRegistry creates a new in-memory flow registry
func NewSimpleFlowRegistry() *SimpleFlowRegistry {
	return &SimpleFlowRegistry{
		flows: make(map[string]*flows.Flow),
	}
}

// Register adds a flow to the registry
func (r *SimpleFlowRegistry) Register(name string, flow *flows.Flow) {
	r.flows[name] = flow
}

// GetFlow retrieves a flow by reference name
func (r *SimpleFlowRegistry) GetFlow(ref string) (*flows.Flow, error) {
	flow, ok := r.flows[ref]
	if !ok {
		return nil, ErrFlowNotFound
	}
	return flow, nil
}

// ErrFlowNotFound is returned when a flow reference doesn't exist
var ErrFlowNotFound = &flowNotFoundError{}

type flowNotFoundError struct{}

func (e *flowNotFoundError) Error() string {
	return "flow not found"
}

// IsFlowNotFound checks if an error is a flow not found error
func IsFlowNotFound(err error) bool {
	_, ok := err.(*flowNotFoundError)
	return ok
}
