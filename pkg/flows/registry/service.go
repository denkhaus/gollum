package registry

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
)

// FlowRegistry defines the interface for looking up flows by reference
type FlowRegistry interface {
	GetFlow(ref string) (*flows.Flow, error)
}

// flowRegistryServiceImpl is the private implementation
type flowRegistryServiceImpl struct {
	flows map[string]*flows.Flow
}

// Ensure flowRegistryServiceImpl implements FlowRegistry
var _ FlowRegistry = (*flowRegistryServiceImpl)(nil)

// NewFlowRegistryService creates the flow registry service (DI constructor)
func NewFlowRegistryService(injector do.Injector) (FlowRegistry, error) {
	return &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}, nil
}

// Register adds a flow to the registry
func (p *flowRegistryServiceImpl) Register(name string, flow *flows.Flow) {
	p.flows[name] = flow
}

// GetFlow retrieves a flow by reference name
func (p *flowRegistryServiceImpl) GetFlow(ref string) (*flows.Flow, error) {
	flow, ok := p.flows[ref]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", ref)
	}
	return flow, nil
}

// LoadFromMap loads flows from a map (for initialization)
func (p *flowRegistryServiceImpl) LoadFromMap(flows map[string]*flows.Flow) {
	for name, flow := range flows {
		p.flows[name] = flow
	}
}
