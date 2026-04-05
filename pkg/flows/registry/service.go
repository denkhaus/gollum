// Package registry provides flow lookup and registration services.
// It maintains a registry of flows that can be referenced by name,
// supporting executor call step operations.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/parser"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// FlowRegistry defines the interface for looking up and registering flows
type FlowRegistry interface {
	// Register adds a flow to the registry
	Register(name string, flow *flows.Flow)
	// GetFlow retrieves a flow by reference name
	GetFlow(ref string) (*flows.Flow, error)
	// ListFlows returns information about all registered flows
	ListFlows() ([]*FlowInfo, error)
}

// FlowInfo holds metadata about a flow for tool discovery
type FlowInfo struct {
	Name         string
	Description  string
	Version      string
	InputFields  []FieldInfo
	OutputFields []FieldInfo
	States       []string
}

// FieldInfo describes a flow field
type FieldInfo struct {
	Name     string
	Type     string
	Required bool
}

// flowRegistryServiceImpl is the private implementation
type flowRegistryServiceImpl struct {
	flows  map[string]*flows.Flow
	logger logger.LoggerService
}

// Ensure flowRegistryServiceImpl implements FlowRegistry
var _ FlowRegistry = (*flowRegistryServiceImpl)(nil)

// NewFlowRegistryService creates the flow registry service (DI constructor)
// It automatically loads flows from:
// 1. ~/.config/gollum/flows
// 2. <current_workspace>/.gollum/flows
func NewFlowRegistryService(injector do.Injector) (FlowRegistry, error) {
	log := do.MustInvoke[logger.LoggerService](injector)
	svc := &flowRegistryServiceImpl{
		flows:  make(map[string]*flows.Flow),
		logger: log,
	}

	// Load flows from ~/.config/gollum/flows
	if homeDir, err := os.UserHomeDir(); err == nil {
		configFlowDir := filepath.Join(homeDir, ".config", "gollum", "flows")
		if err := svc.LoadFromDirectory(configFlowDir); err != nil {
			// Directory doesn't exist or isn't accessible - that's ok
			// Only log if it's an unexpected error
			if !os.IsNotExist(err) {
				svc.logger.Warn("failed to load flows", zap.String("dir", configFlowDir), zap.Error(err))
			}
		}
	}

	// Load flows from <workspace>/.gollum/flows
	if cwd, err := os.Getwd(); err == nil {
		workspaceFlowDir := filepath.Join(cwd, ".gollum", "flows")
		if err := svc.LoadFromDirectory(workspaceFlowDir); err != nil {
			// Directory doesn't exist or isn't accessible - that's ok
			if !os.IsNotExist(err) {
				svc.logger.Warn("failed to load flows", zap.String("dir", workspaceFlowDir), zap.Error(err))
			}
		}
	}

	return svc, nil
}

// Register adds a flow to the registry
func (s *flowRegistryServiceImpl) Register(name string, flow *flows.Flow) {
	s.flows[name] = flow
}

// GetFlow retrieves a flow by reference name
func (s *flowRegistryServiceImpl) GetFlow(ref string) (*flows.Flow, error) {
	flow, ok := s.flows[ref]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", ref)
	}
	return flow, nil
}

// LoadFromMap loads flows from a map (for initialization)
func (s *flowRegistryServiceImpl) LoadFromMap(flows map[string]*flows.Flow) {
	for name, flow := range flows {
		s.flows[name] = flow
	}
}

// LoadFromDirectory scans a directory for XML flow files and loads them
func (s *flowRegistryServiceImpl) LoadFromDirectory(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("flow directory not accessible: %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	// Walk the directory and find all .xml files
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories we can't access
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .xml files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".xml") {
			return nil
		}

		// Parse the flow file
		flow, err := parser.Parse(path)
		if err != nil {
			// Log the error but continue loading other flows
			if s.logger != nil {
				s.logger.Warn("failed to parse flow file", zap.String("path", path), zap.Error(err))
			}
			return nil
		}

		// Register the flow
		s.Register(flow.Name, flow)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", dir, err)
	}

	return nil
}

// ListFlows returns information about all registered flows
func (s *flowRegistryServiceImpl) ListFlows() ([]*FlowInfo, error) {
	result := make([]*FlowInfo, 0, len(s.flows))
	for _, flow := range s.flows {
		info := s.flowToFlowInfo(flow)
		result = append(result, info)
	}
	return result, nil
}

// flowToFlowInfo converts a Flow to FlowInfo
func (s *flowRegistryServiceImpl) flowToFlowInfo(flow *flows.Flow) *FlowInfo {
	var inputFields []FieldInfo
	if flow.Input != nil {
		inputFields = make([]FieldInfo, 0, len(flow.Input.Strings)+len(flow.Input.Ints)+len(flow.Input.Bools)+len(flow.Input.Floats)+len(flow.Input.Arrays)+len(flow.Input.Maps)+len(flow.Input.Objects))
		for _, f := range flow.Input.Strings {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeString),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Ints {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeInt),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Bools {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeBool),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Floats {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeFloat),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Arrays {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeArray),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Maps {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeMap),
				Required: f.Required,
			})
		}
		for _, f := range flow.Input.Objects {
			inputFields = append(inputFields, FieldInfo{
				Name:     f.Name,
				Type:     string(flows.TypeObject),
				Required: false,
			})
		}
	}

	var outputFields []FieldInfo
	if flow.Output != nil {
		outputFields = make([]FieldInfo, 0, len(flow.Output.Strings)+len(flow.Output.Ints)+len(flow.Output.Bools)+len(flow.Output.Floats)+len(flow.Output.Objects))
		for _, f := range flow.Output.Strings {
			outputFields = append(outputFields, FieldInfo{
				Name: f.Name,
				Type: string(flows.TypeString),
			})
		}
		for _, f := range flow.Output.Ints {
			outputFields = append(outputFields, FieldInfo{
				Name: f.Name,
				Type: string(flows.TypeInt),
			})
		}
		for _, f := range flow.Output.Bools {
			outputFields = append(outputFields, FieldInfo{
				Name: f.Name,
				Type: string(flows.TypeBool),
			})
		}
		for _, f := range flow.Output.Floats {
			outputFields = append(outputFields, FieldInfo{
				Name: f.Name,
				Type: string(flows.TypeFloat),
			})
		}
		for _, f := range flow.Output.Objects {
			outputFields = append(outputFields, FieldInfo{
				Name: f.Name,
				Type: string(flows.TypeObject),
			})
		}
	}

	states := make([]string, 0, len(flow.States))
	for _, s := range flow.States {
		states = append(states, s.Name)
	}

	return &FlowInfo{
		Name:         flow.Name,
		Description:  flow.Description,
		Version:      flow.Version,
		InputFields:  inputFields,
		OutputFields: outputFields,
		States:       states,
	}
}
