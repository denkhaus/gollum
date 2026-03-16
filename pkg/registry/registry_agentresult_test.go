package registry

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentResultStorage(t *testing.T) {
	// Create a minimal registry for testing (skip DI)
	limits := &config.AgentLimitsConfig{
		MaxTotalAgents:        100,
		MaxSubAgentsPerParent: 10,
	}

	r := &agentRegistry{
		agents:       make(map[uuid.UUID]*agentHandle),
		agentResults: make(map[uuid.UUID]*shared.AgentResult),
		config:       limits,
	}

	agentID := uuid.New()
	now := time.Now().Unix()

	t.Run("StoreAgentResult", func(t *testing.T) {
		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			Output:    map[string]interface{}{"message": "test"},
			StartedAt: now,
		}

		err := r.StoreAgentResult(result)
		require.NoError(t, err)

		// Verify it was stored
		r.mutex.RLock()
		stored, exists := r.agentResults[agentID]
		r.mutex.RUnlock()

		require.True(t, exists)
		assert.Equal(t, agentID, stored.AgentID)
		assert.Equal(t, shared.AgentStatusRunning, stored.Status)
	})

	t.Run("GetAgentResult", func(t *testing.T) {
		result, exists := r.GetAgentResult(agentID)

		require.True(t, exists)
		assert.Equal(t, agentID, result.AgentID)
		assert.Equal(t, shared.AgentStatusRunning, result.Status)
		assert.Equal(t, "test", result.Output["message"])

		// Verify returned copy is independent
		result.Status = shared.AgentStatusCompleted

		original, exists := r.GetAgentResult(agentID)
		require.True(t, exists)
		assert.Equal(t, shared.AgentStatusRunning, original.Status, "modifying return should not affect stored value")
	})

	t.Run("GetAgentResult not found", func(t *testing.T) {
		_, exists := r.GetAgentResult(uuid.New())
		assert.False(t, exists)
	})
}

func TestWaitForAgent(t *testing.T) {
	limits := &config.AgentLimitsConfig{
		MaxTotalAgents:        100,
		MaxSubAgentsPerParent: 10,
	}

	r := &agentRegistry{
		agents:       make(map[uuid.UUID]*agentHandle),
		agentResults: make(map[uuid.UUID]*shared.AgentResult),
		config:       limits,
	}

	t.Run("WaitForAgent completes successfully", func(t *testing.T) {
		agentID := uuid.New()
		now := time.Now().Unix()

		// Start with running status
		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			Output:    map[string]interface{}{"message": "test"},
			StartedAt: now,
		}
		require.NoError(t, r.StoreAgentResult(result))

		// Complete the agent in a goroutine
		go func() {
			time.Sleep(50 * time.Millisecond)
			completed := now + 1
			result.Status = shared.AgentStatusCompleted
			result.CompletedAt = &completed
			result.Output = map[string]interface{}{"result": "done"}
			require.NoError(t, r.StoreAgentResult(result))
		}()

		// Wait for completion
		ctx := context.Background()
		finalResult, err := r.WaitForAgent(ctx, agentID, 200*time.Millisecond)

		require.NoError(t, err)
		assert.Equal(t, agentID, finalResult.AgentID)
		assert.Equal(t, shared.AgentStatusCompleted, finalResult.Status)
		assert.Equal(t, "done", finalResult.Output["result"])
		assert.NotNil(t, finalResult.CompletedAt)
	})

	t.Run("WaitForAgent timeout", func(t *testing.T) {
		agentID := uuid.New()
		now := time.Now().Unix()

		// Store running agent that never completes
		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			Output:    map[string]interface{}{},
			StartedAt: now,
		}
		require.NoError(t, r.StoreAgentResult(result))

		// Wait with short timeout
		ctx := context.Background()
		finalResult, err := r.WaitForAgent(ctx, agentID, 100*time.Millisecond)

		// Should return current result with timeout error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.NotNil(t, finalResult)
		assert.Equal(t, agentID, finalResult.AgentID)
		assert.Equal(t, shared.AgentStatusRunning, finalResult.Status)
	})

	t.Run("WaitForAgent agent not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := r.WaitForAgent(ctx, uuid.New(), 100*time.Millisecond)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("WaitForAgent failed status", func(t *testing.T) {
		agentID := uuid.New()
		now := time.Now().Unix()

		// Store failed agent
		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusFailed,
			Error:     "something went wrong",
			StartedAt: now,
		}
		require.NoError(t, r.StoreAgentResult(result))

		// Should return immediately for failed agents
		ctx := context.Background()
		finalResult, err := r.WaitForAgent(ctx, agentID, 1*time.Second)

		require.NoError(t, err)
		assert.Equal(t, shared.AgentStatusFailed, finalResult.Status)
		assert.Equal(t, "something went wrong", finalResult.Error)
	})

	t.Run("WaitForAgent context cancellation", func(t *testing.T) {
		agentID := uuid.New()
		now := time.Now().Unix()

		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			Output:    map[string]interface{}{},
			StartedAt: now,
		}
		require.NoError(t, r.StoreAgentResult(result))

		// Cancel context after 50ms
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := r.WaitForAgent(ctx, agentID, 10*time.Second)

		assert.Error(t, err)
	})

	t.Run("WaitForAgent already completed", func(t *testing.T) {
		agentID := uuid.New()
		now := time.Now().Unix()
		completed := now + 100

		result := shared.AgentResult{
			AgentID:     agentID,
			Status:      shared.AgentStatusCompleted,
			Output:      map[string]interface{}{"done": true},
			StartedAt:   now,
			CompletedAt: &completed,
		}
		require.NoError(t, r.StoreAgentResult(result))

		// Should return immediately
		ctx := context.Background()
		finalResult, err := r.WaitForAgent(ctx, agentID, 1*time.Second)

		require.NoError(t, err)
		assert.Equal(t, shared.AgentStatusCompleted, finalResult.Status)
		assert.True(t, finalResult.Output["done"].(bool))
	})
}

func TestAgentResultThreadSafety(t *testing.T) {
	limits := &config.AgentLimitsConfig{
		MaxTotalAgents:        100,
		MaxSubAgentsPerParent: 10,
	}

	r := &agentRegistry{
		agents:       make(map[uuid.UUID]*agentHandle),
		agentResults: make(map[uuid.UUID]*shared.AgentResult),
		config:       limits,
	}

	t.Run("Concurrent StoreAgentResult", func(t *testing.T) {
		numGoroutines := 100
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				agentID := uuid.New()
				result := shared.AgentResult{
					AgentID:   agentID,
					Status:    shared.AgentStatusRunning,
					Output:    map[string]interface{}{"index": idx},
					StartedAt: time.Now().Unix(),
				}
				err := r.StoreAgentResult(result)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all agents were stored
		r.mutex.RLock()
		count := len(r.agentResults)
		r.mutex.RUnlock()

		assert.Equal(t, numGoroutines, count)
	})

	t.Run("Concurrent GetAgentResult", func(t *testing.T) {
		agentID := uuid.New()
		result := shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			Output:    map[string]interface{}{"data": "test"},
			StartedAt: time.Now().Unix(),
		}
		require.NoError(t, r.StoreAgentResult(result))

		numGoroutines := 50
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, exists := r.GetAgentResult(agentID)
				assert.True(t, exists)
				done <- true
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("Cleanup removes agent results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create test agents with results
		parentID := uuid.New()
		childID := uuid.New()
		grandchildID := uuid.New()
		now := time.Now().Unix()

		// Create mock agents using centralized mocks
		parentAgent := shared.NewMockAgent(ctrl)
		childAgent := shared.NewMockAgent(ctrl)
		grandchildAgent := shared.NewMockAgent(ctrl)

		// Setup mock expectations for GetID and GetConfig
		parentAgent.EXPECT().GetID().Return(parentID).AnyTimes()
		parentAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{ID: parentID}).AnyTimes()
		childAgent.EXPECT().GetID().Return(childID).AnyTimes()
		childAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{ID: childID, ParentID: &parentID}).AnyTimes()
		grandchildAgent.EXPECT().GetID().Return(grandchildID).AnyTimes()
		grandchildAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{ID: grandchildID, ParentID: &childID}).AnyTimes()

		// Register agents
		require.NoError(t, r.Register(parentAgent, &shared.AgentConfig{
			ID:   parentID,
			Role: "parent",
		}))
		require.NoError(t, r.Register(childAgent, &shared.AgentConfig{
			ID:       childID,
			ParentID: &parentID,
			Role:     "child",
		}))
		require.NoError(t, r.Register(grandchildAgent, &shared.AgentConfig{
			ID:       grandchildID,
			ParentID: &childID,
			Role:     "grandchild",
		}))

		// Store agent results for all agents
		require.NoError(t, r.StoreAgentResult(shared.AgentResult{
			AgentID:   parentID,
			Status:    shared.AgentStatusRunning,
			StartedAt: now,
		}))
		require.NoError(t, r.StoreAgentResult(shared.AgentResult{
			AgentID:   childID,
			Status:    shared.AgentStatusRunning,
			StartedAt: now,
		}))
		require.NoError(t, r.StoreAgentResult(shared.AgentResult{
			AgentID:   grandchildID,
			Status:    shared.AgentStatusRunning,
			StartedAt: now,
		}))

		// Verify all results exist
		_, exists := r.GetAgentResult(parentID)
		assert.True(t, exists)
		_, exists = r.GetAgentResult(childID)
		assert.True(t, exists)
		_, exists = r.GetAgentResult(grandchildID)
		assert.True(t, exists)

		// Cleanup parent agent (should recursively remove all)
		err := r.Cleanup(parentID)
		require.NoError(t, err)

		// Verify all agent results were removed
		_, exists = r.GetAgentResult(parentID)
		assert.False(t, exists, "parent result should be removed")
		_, exists = r.GetAgentResult(childID)
		assert.False(t, exists, "child result should be removed")
		_, exists = r.GetAgentResult(grandchildID)
		assert.False(t, exists, "grandchild result should be removed")

		// Verify agents were removed from registry
		assert.Equal(t, 0, r.GetTotalAgentCount())
	})

	t.Run("Cleanup cancels background agent context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		agentID := uuid.New()
		now := time.Now().Unix()
		cancelCalled := false

		// Create a cancel function that tracks when it's called
		var cancelFunc context.CancelFunc
		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = func() {
			cancelCalled = true
			cancel()
		}

		// Create mock agent
		mockAgent := shared.NewMockAgent(ctrl)
		mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
		mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
			ID:   agentID,
			Role: "test",
		}).AnyTimes()

		// Register agent with cancel function
		require.NoError(t, r.Register(mockAgent, &shared.AgentConfig{
			ID:   agentID,
			Role: "test",
		}, cancelFunc))

		// Store agent result
		require.NoError(t, r.StoreAgentResult(shared.AgentResult{
			AgentID:   agentID,
			Status:    shared.AgentStatusRunning,
			StartedAt: now,
		}))

		// Verify agent and result exist
		_, exists := r.GetAgentResult(agentID)
		assert.True(t, exists)
		assert.Equal(t, 1, r.GetTotalAgentCount())

		// Cleanup should call cancel function
		err := r.Cleanup(agentID)
		require.NoError(t, err)

		// Verify cancel was called
		assert.True(t, cancelCalled, "cancel function should have been called during Cleanup")

		// Verify agent and result were removed
		_, exists = r.GetAgentResult(agentID)
		assert.False(t, exists)
		assert.Equal(t, 0, r.GetTotalAgentCount())

		// Verify context is cancelled
		assert.Equal(t, context.Canceled, ctx.Err())
	})
}
