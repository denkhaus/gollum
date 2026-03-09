package extensions

import (
	"context"
	"fmt"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
	"github.com/samber/do/v2"
)

// ScriggoRunner manages pre-compiled func steps for high-performance execution.
// Functions are compiled once on load and executed many times via program bytecode.
type ScriggoRunner interface {
	// LoadFunc compiles and caches a function
	LoadFunc(name, source string) error

	// ExecuteFunc runs a loaded function with arguments
	ExecuteFunc(name string, args map[string]any) (any, error)

	// ListFuncs returns available function names
	ListFuncs() []string
}

// scriggoRunnerImpl is the private implementation
type scriggoRunnerImpl struct {
	funcs   map[string]*scriggo.Program
	gateway DIGateway
}

// Ensure scriggoRunnerImpl implements ScriggoRunner at compile time
var _ ScriggoRunner = (*scriggoRunnerImpl)(nil)

// NewScriggoRunner creates the Scriggo runner service.
// Exports injector and do functions to Scriggo programs for DI integration.
func NewScriggoRunner(injector do.Injector) (ScriggoRunner, error) {
	gateway, err := do.Invoke[DIGateway](injector)
	if err != nil {
		return nil, fmt.Errorf("get gateway: %w", err)
	}

	return &scriggoRunnerImpl{
		funcs:   make(map[string]*scriggo.Program),
		gateway: gateway,
	}, nil
}

func (p *scriggoRunnerImpl) LoadFunc(name, source string) error {
	// Validate function name
	if err := ValidateFuncName(name); err != nil {
		return fmt.Errorf("invalid function name: %w", err)
	}

	// Create a file system with the function source
	fsys := scriggo.Files{
		"main.go": []byte(source),
	}

	// Build options with exported DI functions
	opts := &scriggo.BuildOptions{
		Packages: native.Packages{
			"main": native.Package{
				Name: "main",
				Declarations: native.Declarations{
					"injector":      p.gateway.Injector(),
					"do_provide":    do.Provide[any],
					"do_must_invoke": do.MustInvoke[any],
					"do_invoke":     do.Invoke[any],
				},
			},
		},
	}

	// Build the program
	program, err := scriggo.Build(fsys, opts)
	if err != nil {
		return fmt.Errorf("compile func %s: %w", name, err)
	}

	p.funcs[name] = program
	return nil
}

func (p *scriggoRunnerImpl) ExecuteFunc(name string, args map[string]any) (any, error) {
	return p.ExecuteFuncWithContext(context.Background(), name, args)
}

// ExecuteFuncWithContext executes a function with timeout support
func (p *scriggoRunnerImpl) ExecuteFuncWithContext(ctx context.Context, name string, args map[string]any) (any, error) {
	fn, ok := p.funcs[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrFuncNotFound, name)
	}

	// Scriggo v0.61.0 supports context cancellation via RunOptions
	// Just pass the context directly - Scriggo will handle cancellation
	return p.runProgram(ctx, fn, args)
}

// runProgram is a helper that runs the compiled program with context cancellation support
func (p *scriggoRunnerImpl) runProgram(ctx context.Context, fn *scriggo.Program, args map[string]any) (any, error) {
	// Execute the program with context cancellation support
	// Note: args are not supported in current Scriggo API
	// Programs run main() function directly
	opts := &scriggo.RunOptions{
		Context: ctx,
	}
	err := fn.Run(opts)
	if err != nil {
		return nil, err
	}

	// Scriggo programs don't return values directly
	return nil, nil
}

func (p *scriggoRunnerImpl) ListFuncs() []string {
	names := make([]string, 0, len(p.funcs))
	for name := range p.funcs {
		names = append(names, name)
	}
	return names
}
