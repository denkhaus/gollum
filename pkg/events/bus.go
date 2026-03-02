package events

import "context"

// SubscriptionOption configures how events are delivered to a handler.
type SubscriptionOption func(*SubscriptionConfig)

// SubscriptionConfig holds configuration for a subscription.
type SubscriptionConfig struct {
	// Async determines if the handler runs asynchronously.
	// If true, the handler runs in a goroutine and does not block the publisher.
	// If false, the handler blocks until completion.
	Async bool

	// Priority determines execution order within sync/async groups.
	// Higher priority handlers execute first. Default is 0.
	Priority int
}

// WithAsync configures the handler to run asynchronously.
func WithAsync() SubscriptionOption {
	return func(c *SubscriptionConfig) {
		c.Async = true
	}
}

// WithSync configures the handler to run synchronously (default).
func WithSync() SubscriptionOption {
	return func(c *SubscriptionConfig) {
		c.Async = false
	}
}

// WithPriority sets the handler priority.
// Higher priority handlers execute first within their sync/async group.
func WithPriority(priority int) SubscriptionOption {
	return func(c *SubscriptionConfig) {
		c.Priority = priority
	}
}

// Handler processes events. Implementations should be thread-safe.
type Handler func(ctx context.Context, event Event) error

// Bus is the central event dispatcher.
// Implementations must be safe for concurrent use.
type Bus interface {
	// Subscribe registers a handler for an event type.
	// Returns a subscription ID that can be used to unsubscribe.
	Subscribe(eventType string, handler Handler, opts ...SubscriptionOption) (subscriptionID string, err error)

	// Unsubscribe removes a subscription by ID.
	// Returns nil if the subscription doesn't exist.
	Unsubscribe(subscriptionID string) error

	// Publish sends an event to all subscribed handlers.
	// Sync handlers are executed first (in priority order), then async handlers.
	// Returns nil even if handlers fail (errors are logged).
	Publish(ctx context.Context, event Event) error
}
