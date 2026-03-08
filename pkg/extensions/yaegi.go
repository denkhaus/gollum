package extensions

import (
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ExtensionState represents the lifecycle state of an extension
type ExtensionState int

const (
	StateLoaded       ExtensionState = iota
	StateInitializing
	StateReady
	StateFailed
	StateUnloading
	StateUnloaded
)

// String returns the string representation of the state
func (s ExtensionState) String() string {
	switch s {
	case StateLoaded:
		return "loaded"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateFailed:
		return "failed"
	case StateUnloading:
		return "unloading"
	case StateUnloaded:
		return "unloaded"
	default:
		return "unknown"
	}
}

// Extension represents a loaded Yaegi extension
type Extension struct {
	Name        string
	Path        string
	Interpreter *interp.Interpreter
	InitFunc    func() error
	Hooks       map[string]interface{}
	State       ExtensionState
	Error       error
	LoadedAt    time.Time
}

// YaegiLoader manages extension packages with full DI integration
type YaegiLoader interface {
	// LoadExtension loads an extension from directory
	LoadExtension(path string) (*Extension, error)

	// InitExtension calls the extension's init function
	InitExtension(ext *Extension) error

	// UnloadExtension cleans up extension resources
	UnloadExtension(ext *Extension) error

	// GetExtension retrieves a loaded extension by name
	GetExtension(name string) (*Extension, error)

	// ListExtensions returns all loaded extension names
	ListExtensions() []string
}

// yaegiLoaderImpl is the private implementation
type yaegiLoaderImpl struct {
	gateway DIGateway
	exts    map[string]*Extension
}

// Ensure yaegiLoaderImpl implements YaegiLoader
var _ YaegiLoader = (*yaegiLoaderImpl)(nil)

// NewYaegiLoader creates the Yaegi loader service
func NewYaegiLoader(injector do.Injector) (YaegiLoader, error) {
	gateway, err := do.Invoke[DIGateway](injector)
	if err != nil {
		return nil, fmt.Errorf("get gateway: %w", err)
	}

	return &yaegiLoaderImpl{
		gateway: gateway,
		exts:    make(map[string]*Extension),
	}, nil
}

func (p *yaegiLoaderImpl) LoadExtension(path string) (*Extension, error) {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	// Export the injector to the extension
	// Note: Using reflect.Value wrapper for compatibility with yaegi/interp.Exports
	// This is a placeholder - actual export will use proper reflection
	_ = p.gateway.Injector() // TODO: Export injector properly using reflect.Value

	// Load main.go - this will be implemented with actual file loading
	// For now, return a placeholder extension
	ext := &Extension{
		Name:        "test",
		Path:        path,
		Interpreter: i,
		InitFunc:    func() error { return nil },
		Hooks:       make(map[string]interface{}),
		State:       StateLoaded,
		LoadedAt:    time.Now(),
	}

	p.exts["test"] = ext
	return ext, nil
}

func (p *yaegiLoaderImpl) InitExtension(ext *Extension) error {
	ext.State = StateInitializing
	if err := ext.InitFunc(); err != nil {
		ext.State = StateFailed
		ext.Error = err
		return err
	}
	ext.State = StateReady
	return nil
}

func (p *yaegiLoaderImpl) UnloadExtension(ext *Extension) error {
	ext.State = StateUnloading
	delete(p.exts, ext.Name)
	ext.State = StateUnloaded
	return nil
}

func (p *yaegiLoaderImpl) GetExtension(name string) (*Extension, error) {
	ext, ok := p.exts[name]
	if !ok {
		return nil, fmt.Errorf("extension not found: %s", name)
	}
	return ext, nil
}

func (p *yaegiLoaderImpl) ListExtensions() []string {
	names := make([]string, 0, len(p.exts))
	for name := range p.exts {
		names = append(names, name)
	}
	return names
}
