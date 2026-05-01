// Package registry provides agent registry and management functionality.
package registry

import (
	"context"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// SourceName is the event source identifier for this service
const SourceName = "agent_registry"

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
		// GetSupervisorAgent returns the singleton supervisor agent
		GetSupervisorAgent() (shared.Agent, error)
	}

	agentHandle struct {
		agent          shared.Agent
		config         *shared.AgentConfig
		cancel         context.CancelFunc       // For cancelling background agent execution
		pendingContext *shared.WorkspaceContext // Pending workspace context update for running agents
		registered     time.Time
		lastUsed       time.Time
	}

	agentRegistry struct {
		agents           map[uuid.UUID]*agentHandle
		agentResults     map[uuid.UUID]*shared.AgentResult
		config           *config.AgentLimitsConfig
		eventBus         events.Bus
		promptManager    manager.PromptManager
		workspaceService workspace.Service
		workspaceContext shared.WorkspaceContext
		skillService     skills.SkillService
		log              *zap.Logger
		mutex            sync.RWMutex
	}
)

// NewAgentRegistry creates a new agent registry service
func NewAgentRegistry(injector do.Injector) (AgentRegistry, error) {
	configService := do.MustInvoke[config.ConfigService](injector)
	logService := do.MustInvoke[logger.LoggerService](injector)
	bus := do.MustInvoke[events.Bus](injector)
	pm := do.MustInvoke[manager.PromptManager](injector)
	ws := do.MustInvoke[workspace.Service](injector)
	ss := do.MustInvoke[skills.SkillService](injector)

	// Create registry first (without skillService - will be set later)
	registry := &agentRegistry{
		agents:           make(map[uuid.UUID]*agentHandle),
		agentResults:     make(map[uuid.UUID]*shared.AgentResult),
		config:           configService.GetAgentLimits(),
		eventBus:         bus,
		promptManager:    pm,
		workspaceService: ws,
		skillService:     ss,
		log:              logService.GetLogger(),
		workspaceContext: shared.WorkspaceContext{
			SkillsXML:   ss.GetSkillsXML(),
			Skills:      ss.GetSkillInfos(),
			CurrentPath: ws.GetCurrentWorkspace(),
		},
	}

	if err := registry.subscribeToEvents(); err != nil {
		return nil, err
	}

	return registry, nil
}

// subscribeToEvents registers event handlers for the registry
func (r *agentRegistry) subscribeToEvents() error {
	// Subscribe to skills updated - async with normal priority
	_, err := events.SubscribeTyped(r.eventBus, events.EventSkillsUpdated,
		r.handleSkillsUpdated,
		events.WithAsync(),
		events.WithPriority(50),
	)
	if err != nil {
		return err
	}

	// Subscribe to directory changes - async with normal priority
	_, err = events.SubscribeTyped(r.eventBus, events.EventDirectoryChanged,
		r.handleDirectoryChanged,
		events.WithAsync(),
		events.WithPriority(50),
	)
	if err != nil {
		return err
	}

	return nil
}

// handleSkillsUpdated handles skills updated events
// When skills change, we update idle agents' system prompts
func (r *agentRegistry) handleSkillsUpdated(ctx context.Context, payload events.SkillsUpdatedPayload) error {
	r.log.Debug("agent registry received skills updated event",
		zap.Int("skill_count", len(payload.Skills)),
		zap.Bool("has_skills_xml", payload.SkillsXML != ""),
	)

	r.workspaceContext.Skills = payload.Skills
	r.workspaceContext.SkillsXML = payload.SkillsXML
	return r.updateIdleAgentsWithWorkspaceContext(ctx)
}

// handleDirectoryChanged handles directory change events
// When directory changes, we update idle agents' system prompts with new path
func (r *agentRegistry) handleDirectoryChanged(ctx context.Context, payload events.DirectoryChangedPayload) error {
	r.log.Debug("agent registry received directory changed event",
		zap.String("old_path", payload.OldPath),
		zap.String("new_path", payload.NewPath))

	r.workspaceContext.CurrentPath = payload.NewPath
	return r.updateIdleAgentsWithWorkspaceContext(ctx)
}

// updateIdleAgentsWithWorkspaceContext updates all agents with the current workspace context.
// Idle agents are updated immediately, running agents receive a pending update that is applied
// when they become idle.
func (r *agentRegistry) updateIdleAgentsWithWorkspaceContext(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Track counts for logging
	idleUpdatedCount := 0
	runningPendingCount := 0

	// Create a copy of the workspace context for pending updates
	contextCopy := r.copyWorkspaceContext(&r.workspaceContext)

	for id, handle := range r.agents {
		// Determine the correct prompt ID based on agent type
		promptID := r.getPromptIDForAgent(handle)

		if r.isAgentIdle(handle) {
			// Agent is idle - update immediately
			if err := r.updateAgentPrompt(ctx, handle, promptID, nil); err != nil {
				r.log.Warn("failed to update idle agent system prompt",
					zap.Error(err),
					zap.String("agent_id", id.String()),
					zap.String("prompt_id", string(promptID)))
				continue
			}
			idleUpdatedCount++
			r.log.Debug("updated idle agent system prompt",
				zap.String("agent_id", id.String()),
				zap.String("prompt_id", string(promptID)))
		} else {
			// Agent is running - store pending context for later application
			handle.pendingContext = contextCopy
			runningPendingCount++
			r.log.Debug("stored pending context update for running agent",
				zap.String("agent_id", id.String()))
		}
	}

	r.log.Info("workspace context update processed",
		zap.Int("total_agents", len(r.agents)),
		zap.Int("idle_updated", idleUpdatedCount),
		zap.Int("running_pending", runningPendingCount))

	return nil
}

// isAgentIdle checks if an agent is currently idle (not running).
// An agent is considered idle if:
// 1. It has no active cancel function (not a background execution), OR
// 2. Its result status is not "running" (completed, failed, or no result yet)
func (r *agentRegistry) isAgentIdle(handle *agentHandle) bool {
	// If there's an active cancel function, check the result status
	if handle.cancel != nil {
		// This is a background agent with cancel - check if still running
		result, exists := r.agentResults[handle.config.SessionContext.AgentID]
		if exists && result.Status == shared.AgentStatusRunning {
			return false // Still running
		}
		return true // Has cancel but not running anymore
	}

	// No cancel function - check result status
	result, exists := r.agentResults[handle.config.SessionContext.AgentID]
	if !exists {
		return true // No result = never started = idle
	}
	return result.Status != shared.AgentStatusRunning
}

// getPromptIDForAgent returns the appropriate prompt ID based on whether the agent is a supervisor or subagent.
func (r *agentRegistry) getPromptIDForAgent(handle *agentHandle) prompt.PromptID {
	if handle.config.ParentID == nil {
		return prompt.PromptIDSupervisorSystem
	}
	return prompt.PromptIDSubagentSystem
}

// updateAgentPrompt renders and applies a new system prompt to an agent.
// If wsContext is nil, uses the registry's current workspace context.
func (r *agentRegistry) updateAgentPrompt(ctx context.Context,
	handle *agentHandle,
	promptID prompt.PromptID,
	wsContext *shared.WorkspaceContext,
) error {
	// Use provided context or fall back to registry's current context
	if wsContext == nil {
		wsContext = &r.workspaceContext
	}

	renderCtx := &prompt.RenderContext{
		Workspace: wsContext,
	}

	newPrompt, err := r.promptManager.GetPromptWithContext(ctx, promptID, renderCtx)
	if err != nil {
		return err
	}

	return handle.agent.UpdateSystemPrompt(ctx, newPrompt)
}

// copyWorkspaceContext creates a deep copy of WorkspaceContext to avoid shared state issues.
func (r *agentRegistry) copyWorkspaceContext(src *shared.WorkspaceContext) *shared.WorkspaceContext {
	if src == nil {
		return nil
	}

	dst := &shared.WorkspaceContext{
		CurrentPath: src.CurrentPath,
		SkillsXML:   src.SkillsXML,
	}

	// Deep copy Skills slice
	if src.Skills != nil {
		dst.Skills = make([]shared.SkillInfo, len(src.Skills))
		copy(dst.Skills, src.Skills)
	}

	return dst
}

func (r *agentRegistry) Register(agent shared.Agent, config *shared.AgentConfig, cancel ...context.CancelFunc) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.agents[config.SessionContext.AgentID]; exists {
		return errs.Conflictf("agent %s already registered", config.SessionContext.AgentID).
			WithContext("agent_id", config.SessionContext.AgentID)
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

	r.agents[config.SessionContext.AgentID] = handle

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
				toCleanup = append(toCleanup, handle.config.SessionContext.AgentID)
				findDescendants(handle.config.SessionContext.AgentID)
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

// StoreAgentResult stores an agent result for background agent tracking.
// When an agent transitions from running to completed/failed, pending context updates are applied.
func (r *agentRegistry) StoreAgentResult(result shared.AgentResult) error {
	r.mutex.Lock()

	// Check if this is a transition from running to completed/failed
	isCompletion := result.Status == shared.AgentStatusCompleted || result.Status == shared.AgentStatusFailed

	// Store a copy to avoid external modifications
	stored := result
	r.agentResults[result.AgentID] = &stored

	// Extract pending context for later application (if any)
	var pendingContext *shared.WorkspaceContext
	var handle *agentHandle
	if isCompletion {
		if h, exists := r.agents[result.AgentID]; exists && h.pendingContext != nil {
			pendingContext = h.pendingContext
			h.pendingContext = nil // Clear it while we hold the lock
			handle = h
		}
	}

	r.mutex.Unlock()

	// Apply pending context update outside the lock
	// This is safe because we've already extracted and cleared the pending context
	if pendingContext != nil && handle != nil {
		ctx := context.Background()
		promptID := r.getPromptIDForAgent(handle)
		if err := r.updateAgentPrompt(ctx, handle, promptID, pendingContext); err != nil {
			r.log.Warn("failed to apply pending update on agent completion",
				zap.Error(err),
				zap.String("agent_id", result.AgentID.String()))
		} else {
			r.log.Debug("applied pending context update to completed agent",
				zap.String("agent_id", result.AgentID.String()))
		}
	}

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

// GetSupervisorAgent returns the singleton supervisor agent
func (r *agentRegistry) GetSupervisorAgent() (shared.Agent, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, handle := range r.agents {
		if handle.config != nil && handle.config.Type == shared.AgentTypeSupervisor {
			return handle.agent, nil
		}
	}

	return nil, errs.NotFoundf("no supervisor agent registered")
}
