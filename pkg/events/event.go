package events

import "time"

// Event is the base interface for all events.
// Events are immutable once created.
type Event interface {
	// Type returns the event type identifier (e.g., "workspace.directory_changed")
	Type() string

	// Timestamp returns when the event was created
	Timestamp() time.Time

	// Source returns the identifier of the event publisher
	Source() string
}

// TypedEvent wraps a payload with type safety and event metadata.
type TypedEvent[T any] struct {
	eventType string
	payload   T
	timestamp time.Time
	source    string
}

// Type returns the event type identifier.
func (e *TypedEvent[T]) Type() string {
	return e.eventType
}

// Payload returns the typed payload.
func (e *TypedEvent[T]) Payload() T {
	return e.payload
}

// Timestamp returns when the event was created.
func (e *TypedEvent[T]) Timestamp() time.Time {
	return e.timestamp
}

// Source returns the identifier of the event publisher.
func (e *TypedEvent[T]) Source() string {
	return e.source
}

// NewEvent creates a new typed event with the current timestamp.
func NewEvent[T any](eventType, source string, payload T) *TypedEvent[T] {
	return &TypedEvent[T]{
		eventType: eventType,
		payload:   payload,
		timestamp: time.Now(),
		source:    source,
	}
}
