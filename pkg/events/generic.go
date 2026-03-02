package events

import (
	"context"
	"fmt"
)

// TypedHandler receives a typed payload, eliminating the need for type assertions.
type TypedHandler[T any] func(ctx context.Context, payload T) error

// SubscribeTyped registers a typed handler for an event type.
// This is a generic helper function that wraps the handler with type safety.
//
// Example:
//
//	events.SubscribeTyped(bus, events.EventDirectoryChanged,
//	    func(ctx context.Context, payload events.DirectoryChangedPayload) error {
//	        fmt.Println("New path:", payload.NewPath)
//	        return nil
//	    },
//	    events.WithSync(),
//	)
func SubscribeTyped[T any](bus Bus, eventType string, handler TypedHandler[T], opts ...SubscriptionOption) (string, error) {
	wrapper := func(ctx context.Context, event Event) error {
		typed, ok := event.(*TypedEvent[T])
		if !ok {
			return fmt.Errorf("unexpected event type for %s: expected *TypedEvent[T], got %T", eventType, event)
		}
		return handler(ctx, typed.Payload())
	}
	return bus.Subscribe(eventType, wrapper, opts...)
}

// PublishTyped publishes a typed event with the given payload.
// This is a generic helper function that creates a typed event.
//
// Example:
//
//	events.PublishTyped(bus, ctx, events.EventDirectoryChanged, "change-directory-tool",
//	    events.DirectoryChangedPayload{
//	        OldPath: "/old",
//	        NewPath: "/new",
//	    })
func PublishTyped[T any](bus Bus, ctx context.Context, eventType, source string, payload T) error {
	return bus.Publish(ctx, NewEvent(eventType, source, payload))
}
