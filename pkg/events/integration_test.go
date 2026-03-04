package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestBus creates an event bus for testing
func setupTestBus(t *testing.T) Bus {
	t.Helper()

	injector := do.New()
	do.Provide(injector, config.NewService)
	do.Provide(injector, logger.NewService)

	logService := do.MustInvoke[logger.LoggerService](injector)
	cfg := &config.EventsConfig{
		MaxRetries:   3,
		RetryDelayMs: 10,
		RetryBackoff: 2,
	}

	return newMemoryBus(logService, cfg)
}

func TestEventBus_BasicPublishSubscribe(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var receivedPayload DirectoryChangedPayload
	var mu sync.Mutex
	handlerCalled := false

	// Subscribe to event
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			mu.Lock()
			defer mu.Unlock()
			handlerCalled = true
			receivedPayload = payload
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Publish event
	testPayload := DirectoryChangedPayload{
		OldPath: "/old/path",
		NewPath: "/new/path",
	}
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source", testPayload)
	require.NoError(t, err)

	// Verify handler was called
	assert.True(t, handlerCalled)
	assert.Equal(t, "/old/path", receivedPayload.OldPath)
	assert.Equal(t, "/new/path", receivedPayload.NewPath)
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var callCount atomic.Int32

	// Subscribe multiple handlers
	for i := 0; i < 3; i++ {
		_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
			func(ctx context.Context, payload DirectoryChangedPayload) error {
				callCount.Add(1)
				return nil
			},
			WithSync(),
		)
		require.NoError(t, err)
	}

	// Publish event
	err := PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)

	// All handlers should be called
	assert.Equal(t, int32(3), callCount.Load())
}

func TestEventBus_PriorityOrder(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var executionOrder []int
	var mu sync.Mutex

	// Subscribe with different priorities
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			mu.Lock()
			defer mu.Unlock()
			executionOrder = append(executionOrder, 1)
			return nil
		},
		WithSync(),
		WithPriority(10),
	)
	require.NoError(t, err)

	_, err = SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			mu.Lock()
			defer mu.Unlock()
			executionOrder = append(executionOrder, 2)
			return nil
		},
		WithSync(),
		WithPriority(20), // Higher priority - should execute first
	)
	require.NoError(t, err)

	_, err = SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			mu.Lock()
			defer mu.Unlock()
			executionOrder = append(executionOrder, 3)
			return nil
		},
		WithSync(),
		WithPriority(5),
	)
	require.NoError(t, err)

	// Publish event
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)

	// Verify execution order (higher priority first)
	assert.Equal(t, []int{2, 1, 3}, executionOrder)
}

func TestEventBus_AsyncHandlers(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var syncCalled atomic.Bool
	var asyncCalled atomic.Bool

	// Subscribe sync handler
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			syncCalled.Store(true)
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Subscribe async handler
	_, err = SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			asyncCalled.Store(true)
			return nil
		},
		WithAsync(),
	)
	require.NoError(t, err)

	// Publish event
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)

	// Sync handler should be called immediately
	assert.True(t, syncCalled.Load())

	// Async handler might not have been called yet
	// Wait a bit for async handler
	time.Sleep(50 * time.Millisecond)
	assert.True(t, asyncCalled.Load())
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var callCount atomic.Int32

	// Subscribe
	subID, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			callCount.Add(1)
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Publish - should be received
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load())

	// Unsubscribe
	err = bus.Unsubscribe(subID)
	require.NoError(t, err)

	// Publish again - should not be received
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load()) // Still 1
}

func TestEventBus_NoSubscribers(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	// Publish with no subscribers - should not error
	err := PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	assert.NoError(t, err)
}

func TestEventBus_HandlerRetry(t *testing.T) {
	// Create bus with retry config
	injector := do.New()
	do.Provide(injector, config.NewService)
	do.Provide(injector, logger.NewService)
	logService := do.MustInvoke[logger.LoggerService](injector)

	cfg := &config.EventsConfig{
		MaxRetries:   2,
		RetryDelayMs: 5,
		RetryBackoff: 2,
	}
	bus := newMemoryBus(logService, cfg)
	ctx := context.Background()

	var attemptCount atomic.Int32

	// Subscribe with a handler that fails initially
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			attempt := attemptCount.Add(1)
			if attempt < 3 {
				return assert.AnError // Fail first 2 attempts
			}
			return nil // Succeed on 3rd attempt
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Publish event
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)

	// Should have been called 3 times (initial + 2 retries)
	assert.Equal(t, int32(3), attemptCount.Load())
}

func TestEventBus_DifferentEventTypes(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var directoryChangedCalled atomic.Bool
	var skillsUpdatedCalled atomic.Bool

	// Subscribe to different event types
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			directoryChangedCalled.Store(true)
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	_, err = SubscribeTyped[SkillsUpdatedPayload](bus, EventSkillsUpdated,
		func(ctx context.Context, payload SkillsUpdatedPayload) error {
			skillsUpdatedCalled.Store(true)
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Publish directory changed event
	err = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
		DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
	require.NoError(t, err)

	assert.True(t, directoryChangedCalled.Load())
	assert.False(t, skillsUpdatedCalled.Load())

	// Publish skills updated event
	err = PublishTyped(bus, ctx, EventSkillsUpdated, "test-source",
		SkillsUpdatedPayload{
			Skills:    []shared.SkillInfo{{Name: "test-skill", Description: "Test skill", Location: "/path/to/skill"}},
			SkillsXML: "<skill>test</skill>",
		})
	require.NoError(t, err)

	assert.True(t, skillsUpdatedCalled.Load())
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := setupTestBus(t)
	ctx := context.Background()

	var callCount atomic.Int32

	// Subscribe
	_, err := SubscribeTyped[DirectoryChangedPayload](bus, EventDirectoryChanged,
		func(ctx context.Context, payload DirectoryChangedPayload) error {
			callCount.Add(1)
			return nil
		},
		WithSync(),
	)
	require.NoError(t, err)

	// Publish concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = PublishTyped(bus, ctx, EventDirectoryChanged, "test-source",
				DirectoryChangedPayload{OldPath: "/old", NewPath: "/new"})
		}()
	}
	wg.Wait()

	// All events should be delivered
	assert.Equal(t, int32(10), callCount.Load())
}
