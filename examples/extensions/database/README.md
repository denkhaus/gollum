# Database Extension Template

This is a template extension for Gollum that demonstrates how to create and register custom services.

## Installation

1. Create directory: `~/.config/gollum/extensions/database/`
2. Copy `main.go` to that directory
3. Restart Gollum

## Usage

The extension registers a `DatabaseService` that can be injected into other extensions or used by the core system.

### Accessing from another extension

```go
func MyExtensionInit() error {
    db := do.MustInvoke[DatabaseService](injector)
    rows, err := db.Query("SELECT * FROM users")
    // ...
}
```

## Development

- Implement the service interface
- Register services in `Init()` function
- Use `do.Provide()` to register services
- Use `do.MustInvoke[T]()` to consume services
