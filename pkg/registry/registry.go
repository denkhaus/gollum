// Package registry provides agent registry and management functionality.
package registry

import (
	"context"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

type (
	// AgentRegistry manages all active agents and their communication
	AgentRegistry interface {
		// Register registers a new agent with optional cancel function for background execution
		Register(agent shared.Agent, config *shared.AgentConfig, cancel ...context.CancelFunc) error
		// Unregister removes an agent and cleans up resources
		Unregister(agentID uuid.UUID) error
		// GetAgent retrieves an agent by ID
		GetAgent(agentID uuid.UUID) (shared.Agent, bool)
		// GetChildren returns all direct children of a parent agent
		GetChildren(parentID uuid.UUID) []shared.Agent
		// GetParent returns the parent of an agent
		GetParent(agentID uuid.UUID) (shared.Agent, bool)
		// IsDirectParent checks if callerID is the DIRECT parent of targetID
		// Used for permission validation - agents can only operate on their direct children
		IsDirectParent(callerID, targetID uuid.UUID) bool
		// ListAll returns all registered agents
		ListAll() map[uuid.UUID]shared.Agent
		// Cleanup removes an agent and all its descendants
		Cleanup(agentID uuid.UUID) error
		// GetTotalAgentCount returns the total number of registered agents
		GetTotalAgentCount() int
		// GetSubAgentCount returns the number of direct subagents for a parent
		GetSubAgentCount(parentID uuid.UUID) int
		// StoreAgentResult stores an agent result for background agent tracking
		StoreAgentResult(result shared.AgentResult) error
		// GetAgentResult retrieves an agent result by agent ID
		GetAgentResult(agentID uuid.UUID) (*shared.AgentResult, bool)
		// WaitForAgent waits for an agent to complete with timeout
		WaitForAgent(ctx context.Context, agentID uuid.UUID, timeout time.Duration) (*shared.AgentResult, error)
		// SetCancelFunc sets or updates the cancel function for an agent
		SetCancelFunc(agentID uuid.UUID, cancel context.CancelFunc) error
		// DeleteAgentResult removes a stored agent result
		DeleteAgentResult(agentID uuid.UUID) error
	}

	agentHandle struct {
		agent      shared.Agent
		config     *shared.AgentConfig
		cancel     context.CancelFunc // For cancelling background agent execution
		registered time.Time
		lastUsed   time.Time
	}

	agentRegistry struct {
		agents       map[uuid.UUID]*agentHandle
		agentResults map[uuid.UUID]*shared.AgentResult
		config       *config.AgentLimitsConfig
		mutex        sync.RWMutex
	}
)

// NewAgentRegistry creates a new agent registry service
func NewAgentRegistry(injector do.Injector) (AgentRegistry, error) {
	configService := do.MustInvoke[config.ConfigService](injector)

	return &agentRegistry{
		agents:       make(map[uuid.UUID]*agentHandle),
		agentResults: make(map[uuid.UUID]*shared.AgentResult),
		config:       configService.GetAgentLimits(),
	}, nil
}

func (r *agentRegistry) Register(agent shared.Agent, config *shared.AgentConfig, cancel ...context.CancelFunc) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.agents[config.ID]; exists {
		return errs.Conflictf("agent %s already registered", config.ID).
			WithContext("agent_id", config.ID)
	}

	// Check total agent limit
	if len(r.agents) >= r.config.MaxTotalAgents {
		return errs.Conflictf("maximum agent limit reached (%d)", r.config.MaxTotalAgents).
			WithContext("current_count", len(r.agents)).
			WithContext("max_limit", r.config.MaxTotalAgents)
	}

	// Check sub-agent limit if this is a subagent
	if config.ParentID != nil {
		subAgentCount := 0
		for _, handle := range r.agents {
			if handle.config.ParentID != nil && *handle.config.ParentID == *config.ParentID {
				subAgentCount++
			}
		}
		if subAgentCount >= r.config.MaxSubAgentsPerParent {
			return errs.Conflictf("maximum sub-agent limit reached for parent (%d)", r.config.MaxSubAgentsPerParent).
				WithContext("parent_id", *config.ParentID).
				WithContext("current_subagents", subAgentCount).
				WithContext("max_limit", r.config.MaxSubAgentsPerParent)
		}
	}

	// Extract cancel function if provided (for background agents)
	var cancelFunc context.CancelFunc
	if len(cancel) > 0 && cancel[0] != nil {
		cancelFunc = cancel[0]
	}

	handle := &agentHandle{
		agent:      agent,
		config:     config,
		cancel:     cancelFunc,
		registered: time.Now(),
		lastUsed:   time.Now(),
	}

	r.agents[config.ID] = handle

	return nil
}

func (r *agentRegistry) Unregister(agentID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	handle, exists := r.agents[agentID]
	if !exists {
		return errs.NotFoundf("agent %s not found", agentID).WithContext("agent_id", agentID)
	}

	// Cancel background execution if present
	if handle.cancel != nil {
		handle.cancel()
	}

	delete(r.agents, agentID)

	return nil
}

func (r *agentRegistry) GetAgent(agentID uuid.UUID) (shared.Agent, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	handle, exists := r.agents[agentID]
	if !exists {
		return nil, false
	}

	return handle.agent, true
}

func (r *agentRegistry) GetChildren(parentID uuid.UUID) []shared.Agent {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var children []shared.Agent
	for _, handle := range r.agents {
		if handle.config.ParentID != nil && *handle.config.ParentID == parentID {
			children = append(children, handle.agent)
		}
	}

	return children
}

func (r *agentRegistry) GetParent(agentID uuid.UUID) (shared.Agent, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	handle, exists := r.agents[agentID]
	if !exists || handle.config.ParentID == nil {
		return nil, false
	}

	parentHandle, exists := r.agents[*handle.config.ParentID]
	if !exists {
		return nil, false
	}

	return parentHandle.agent, true
}

// IsDirectParent checks if callerID is the DIRECT parent of targetID
// Used for permission validation - agents can only operate on their direct children
func (r *agentRegistry) IsDirectParent(callerID, targetID uuid.UUID) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	handle, exists := r.agents[targetID]
	if !exists {
		return false
	}

	// Target must have a parent, and that parent must be the caller
	if handle.config.ParentID == nil {
		return false
	}

	return *handle.config.ParentID == callerID
}

func (r *agentRegistry) ListAll() map[uuid.UUID]shared.Agent {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[uuid.UUID]shared.Agent)
	for id, handle := range r.agents {
		result[id] = handle.agent
	}

	return result
}

func (r *agentRegistry) Cleanup(agentID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// First, find all descendants
	var toCleanup []uuid.UUID
	var findDescendants func(id uuid.UUID)

	findDescendants = func(id uuid.UUID) {
		for _, handle := range r.agents {
			if handle.config.ParentID != nil && *handle.config.ParentID == id {
				toCleanup = append(toCleanup, handle.config.ID)
				findDescendants(handle.config.ID)
			}
		}
	}

	findDescendants(agentID)
	toCleanup = append(toCleanup, agentID)

	// Clean up all agents in reverse order (children first)
	for i := len(toCleanup) - 1; i >= 0; i-- {
		id := toCleanup[i]
		if handle, exists := r.agents[id]; exists {
			// Cancel background execution if present
			if handle.cancel != nil {
				handle.cancel()
			}
			delete(r.agents, id)
		}
		// Also clean up associated agent results
		delete(r.agentResults, id)
	}

	return nil
}

func (r *agentRegistry) GetTotalAgentCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.agents)
}

func (r *agentRegistry) GetSubAgentCount(parentID uuid.UUID) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, handle := range r.agents {
		if handle.config.ParentID != nil && *handle.config.ParentID == parentID {
			count++
		}
	}
	return count
}

// StoreAgentResult stores an agent result for background agent tracking
func (r *agentRegistry) StoreAgentResult(result shared.AgentResult) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Store a copy to avoid external modifications
	stored := result
	r.agentResults[result.AgentID] = &stored
	return nil
}

// GetAgentResult retrieves an agent result by agent ID
func (r *agentRegistry) GetAgentResult(agentID uuid.UUID) (*shared.AgentResult, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result, exists := r.agentResults[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid external modifications
	resultCopy := *result
	if result.CompletedAt != nil {
		completed := *result.CompletedAt
		resultCopy.CompletedAt = &completed
	}

	return &resultCopy, true
}

// WaitForAgent waits for an agent to complete with timeout
func (r *agentRegistry) WaitForAgent(ctx context.Context, agentID uuid.UUID, timeout time.Duration) (*shared.AgentResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled or timeout
			r.mutex.RLock()
			result, exists := r.agentResults[agentID]
			if exists {
				// Return a copy
				resultCopy := *result
				if result.CompletedAt != nil {
					completed := *result.CompletedAt
					resultCopy.CompletedAt = &completed
				}
				r.mutex.RUnlock()
				return &resultCopy, errs.Timeoutf("timeout waiting for agent %s", agentID).
					WithContext("agent_id", agentID)
			}
			r.mutex.RUnlock()
			return nil, errs.NotFoundf("timeout: agent %s not found", agentID).
				WithContext("agent_id", agentID)

		case <-ticker.C:
			r.mutex.RLock()
			result, exists := r.agentResults[agentID]
			r.mutex.RUnlock()

			if !exists {
				return nil, errs.NotFoundf("agent %s not found", agentID).
					WithContext("agent_id", agentID)
			}

			// Return if agent is completed or failed
			if result.Status == shared.AgentStatusCompleted || result.Status == shared.AgentStatusFailed {
				// Return a copy
				resultCopy := *result
				if result.CompletedAt != nil {
					completed := *result.CompletedAt
					resultCopy.CompletedAt = &completed
				}
				return &resultCopy, nil
			}

			// Continue waiting for running agents
		}
	}
}

// SetCancelFunc sets or updates the cancel function for an agent
func (r *agentRegistry) SetCancelFunc(agentID uuid.UUID, cancel context.CancelFunc) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	handle, exists := r.agents[agentID]
	if !exists {
		return errs.NotFoundf("agent %s not found", agentID).WithContext("agent_id", agentID)
	}

	// Cancel old function if exists
	if handle.cancel != nil {
		handle.cancel()
	}

	// Set new cancel function
	handle.cancel = cancel
	return nil
}

// DeleteAgentResult removes a stored agent result
func (r *agentRegistry) DeleteAgentResult(agentID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.agentResults, agentID)
	return nil
}
