package events

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// subscription represents a single event subscription.
type subscription struct {
	id       string
	handler  Handler
	async    bool
	priority int
}

// memoryBus is an in-process event bus implementation.
// It is safe for concurrent use.
type memoryBus struct {
	logger logger.LoggerService
	config *config.EventsConfig
	mu     sync.RWMutex
	subs   map[string][]*subscription // eventType -> subscriptions
}

// newMemoryBus creates a new in-memory event bus.
func newMemoryBus(logService logger.LoggerService, cfg *config.EventsConfig) *memoryBus {
	return &memoryBus{
		logger: logService,
		config: cfg,
		subs:   make(map[string][]*subscription),
	}
}

// Subscribe registers a handler for an event type.
func (b *memoryBus) Subscribe(eventType string, handler Handler, opts ...SubscriptionOption) (string, error) {
	cfg := &SubscriptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	sub := &subscription{
		id:       uuid.New().String(),
		handler:  handler,
		async:    cfg.Async,
		priority: cfg.Priority,
	}

	b.mu.Lock()
	b.subs[eventType] = append(b.subs[eventType], sub)
	b.mu.Unlock()

	b.logger.Debug("event subscription created",
		zap.String("subscription_id", sub.id),
		zap.String("event_type", eventType),
		zap.Bool("async", sub.async),
		zap.Int("priority", sub.priority))

	return sub.id, nil
}

// Unsubscribe removes a subscription by ID.
func (b *memoryBus) Unsubscribe(subscriptionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for eventType, subs := range b.subs {
		for i, sub := range subs {
			if sub.id == subscriptionID {
				b.subs[eventType] = append(subs[:i], subs[i+1:]...)
				b.logger.Debug("event subscription removed",
					zap.String("subscription_id", subscriptionID),
					zap.String("event_type", eventType))
				return nil
			}
		}
	}

	// Not found is not an error
	return nil
}

// Publish sends an event to all subscribed handlers.
func (b *memoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	subs := b.subs[event.Type()]
	b.mu.RUnlock()

	if len(subs) == 0 {
		b.logger.Debug("no subscribers for event", zap.String("event_type", event.Type()))
		return nil
	}

	b.logger.Debug("publishing event",
		zap.String("event_type", event.Type()),
		zap.String("source", event.Source()),
		zap.Int("subscriber_count", len(subs)))

	// Sort by priority (descending) - higher priority first
	sorted := make([]*subscription, len(subs))
	copy(sorted, subs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].priority > sorted[j].priority
	})

	// Execute sync handlers first (block until complete)
	var asyncHandlers []*subscription
	for _, sub := range sorted {
		if !sub.async {
			if err := b.executeWithRetry(ctx, sub, event); err != nil {
				b.logger.Error("sync handler failed after retries",
					zap.String("subscription_id", sub.id),
					zap.String("event_type", event.Type()),
					zap.Error(err))
				// Continue with other handlers - don't fail the whole publish
			}
		} else {
			asyncHandlers = append(asyncHandlers, sub)
		}
	}

	// Execute async handlers in background
	for _, sub := range asyncHandlers {
		go func(s *subscription) {
			if err := b.executeWithRetry(context.Background(), s, event); err != nil {
				b.logger.Error("async handler failed after retries",
					zap.String("subscription_id", s.id),
					zap.String("event_type", event.Type()),
					zap.Error(err))
			}
		}(sub)
	}

	return nil
}

// executeWithRetry executes a handler with exponential backoff retry.
func (b *memoryBus) executeWithRetry(ctx context.Context, sub *subscription, event Event) error {
	var lastErr error
	delay := time.Duration(b.config.RetryDelayMs) * time.Millisecond

	for attempt := 0; attempt <= b.config.MaxRetries; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := sub.handler(ctx, event); err != nil {
			lastErr = err
			if attempt < b.config.MaxRetries {
				b.logger.Warn("handler failed, retrying",
					zap.String("subscription_id", sub.id),
					zap.String("event_type", event.Type()),
					zap.Int("attempt", attempt+1),
					zap.Duration("delay", delay),
					zap.Error(err))
				time.Sleep(delay)
				delay *= time.Duration(b.config.RetryBackoff)
				continue
			}
		}
		// Success
		return nil
	}

	return lastErr
}
