// Package events provides a centralized event bus for decoupled communication
// between services in the Gollum application.
//
// The event bus implements a publish/subscribe pattern where:
//   - Publishers emit events when state changes occur
//   - Subscribers register handlers to react to specific event types
//   - Handlers can be synchronous (blocking) or asynchronous (non-blocking)
//
// # Key Features
//
//   - Type-safe event handling via generic helper functions
//   - Priority-based handler execution (higher priority runs first)
//   - Automatic retry with exponential backoff for failed handlers
//   - Support for both sync and async handlers
//   - DI singleton pattern ensures all services use the same bus
//
// # Quick Start
//
// Publishing events:
//
//	err := events.PublishTyped(bus, ctx,
//	    string(events.EventDirectoryChanged),
//	    "source-name",
//	    events.DirectoryChangedPayload{
//	        OldPath: "/old",
//	        NewPath: "/new",
//	    })
//
// Subscribing to events:
//
//	_, err := events.SubscribeTyped[events.DirectoryChangedPayload](bus,
//	    string(events.EventDirectoryChanged),
//	    func(ctx context.Context, p events.DirectoryChangedPayload) error {
//	        // Handle the event
//	        return nil
//	    },
//	    events.WithSync(),
//	    events.WithPriority(100),
//	)
//
// # Event Types
//
// Event types are defined as constants in this package. Each event type has a
// corresponding payload struct. See types.go for the complete list.
//
// # Handler Options
//
//   - WithSync(): Handler blocks publisher until complete (default)
//   - WithAsync(): Handler runs in goroutine, publisher continues immediately
//   - WithPriority(n): Higher priority handlers execute first within their group
//
// # Architecture
//
// The event bus enforces decoupling between publishers and subscribers:
//   - Publishers don't know which services will handle their events
//   - Subscribers self-register during initialization
//   - No direct service-to-service calls
//
// See the full documentation at docs/event-bus.md for detailed usage examples,
// best practices, and troubleshooting guides.
package events
