// Package extensions provides core services for the Gollum extension system.
//
// DEPRECATED: The hook definitions in this file are replaced by the comprehensive
// hook system in pkg/hooks. Use pkg/hooks.HookManager for flow execution hooks:
//
// - BeforeFlowStep / AfterFlowStep (instead of HookFlowPreExecute/PostExecute)
// - BeforeToolExecution / AfterToolExecution (instead of HookToolPreExecute/PostExecute)
// - BeforeAgentSpawn / AfterAgentSpawn (instead of HookAgentPreExecute/PostExecute)
//
// This file is kept for backward compatibility and will be removed in a future version.
package extensions

import (
	"fmt"
	"time"

	"github.com/samber/do/v2"
)

// HookType defines the lifecycle hook point
type HookType string

const (
	// HookAgentPreExecute fires before an agent executes a step
	//
	// DEPRECATED: Use hooks.BeforeAgentSpawn or hooks.BeforeFlowStep instead.
	// This constant will be removed in a future version.
	HookAgentPreExecute HookType = "agent_pre_execute"

	// HookAgentPostExecute fires after an agent executes a step
	//
	// DEPRECATED: Use hooks.AfterAgentSpawn or hooks.AfterFlowStep instead.
	// This constant will be removed in a future version.
	HookAgentPostExecute HookType = "agent_post_execute"

	// HookFlowPreExecute fires before a flow executes
	//
	// DEPRECATED: Use hooks.BeforeFlowStep instead.
	// This constant will be removed in a future version.
	HookFlowPreExecute HookType = "flow_pre_execute"

	// HookFlowPostExecute fires after a flow executes
	//
	// DEPRECATED: Use hooks.AfterFlowStep instead.
	// This constant will be removed in a future version.
	HookFlowPostExecute HookType = "flow_post_execute"

	// HookToolPreExecute fires before a tool executes
	//
	// DEPRECATED: Use hooks.BeforeToolExecution instead.
	// This constant will be removed in a future version.
	HookToolPreExecute HookType = "tool_pre_execute"

	// HookToolPostExecute fires after a tool executes
	//
	// DEPRECATED: Use hooks.AfterToolExecution instead.
	// This constant will be removed in a future version.
	HookToolPostExecute HookType = "tool_post_execute"
)

// HookContext provides context to hook functions
type HookContext struct {
	Type      HookType
	AgentID   string
	Timestamp time.Time
	Metadata  map[string]any
}

// HookFunction is the signature for hook callbacks
type HookFunction func(ctx *HookContext) error

// HookRegistry manages registered hooks
type HookRegistry interface {
	Register(hookType HookType, name string, fn HookFunction) error
	Unregister(hookType HookType, name string) error
	Execute(hookType HookType, ctx *HookContext) error
}

// hookRegistryImpl is the private implementation
type hookRegistryImpl struct {
	hooks map[HookType]map[string]HookFunction
}

// Ensure hookRegistryImpl implements HookRegistry
var _ HookRegistry = (*hookRegistryImpl)(nil)

// NewHookRegistry creates a new hook registry
func NewHookRegistry(injector do.Injector) (HookRegistry, error) {
	return &hookRegistryImpl{
		hooks: make(map[HookType]map[string]HookFunction),
	}, nil
}

func (p *hookRegistryImpl) Register(hookType HookType, name string, fn HookFunction) error {
	if fn == nil {
		return ErrNilHookFunction
	}
	if p.hooks[hookType] == nil {
		p.hooks[hookType] = make(map[string]HookFunction)
	}
	p.hooks[hookType][name] = fn
	return nil
}

func (p *hookRegistryImpl) Unregister(hookType HookType, name string) error {
	if p.hooks[hookType] != nil {
		delete(p.hooks[hookType], name)
	}
	return nil
}

func (p *hookRegistryImpl) Execute(hookType HookType, ctx *HookContext) error {
	if ctx == nil {
		return fmt.Errorf("hook context cannot be nil")
	}
	for name, fn := range p.hooks[hookType] {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("hook %s failed: %w", name, err)
		}
	}
	return nil
}
