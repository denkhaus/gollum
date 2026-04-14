package extensions

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"unicode"

	"github.com/samber/do/v2"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// YaegiFuncRunner manages dynamic Go function execution using Yaegi interpreter.
// Functions are loaded once and can be called many times without hardcoded wrappers.
type YaegiFuncRunner interface {
	// LoadFunc compiles and caches a function from source code
	LoadFunc(name, source string) error

	// ExecuteFunc runs a loaded function with arguments
	ExecuteFunc(name string, args map[string]any) (any, error)

	// ListFuncs returns available function names
	ListFuncs() []string
}

// yaegiFuncRunnerImpl is the private implementation
type yaegiFuncRunnerImpl struct {
	gateway DIGateway
	i       *interp.Interpreter
	funcs   map[string]*funcInfo
}

// funcInfo stores metadata about loaded functions
type funcInfo struct {
	name       string
	pkgName    string
	paramNames []string // Parameter names in order
}

// Ensure yaegiFuncRunnerImpl implements YaegiFuncRunner
var _ YaegiFuncRunner = (*yaegiFuncRunnerImpl)(nil)

// NewYaegiFuncRunner creates the Yaegi function runner service
func NewYaegiFuncRunner(injector do.Injector) (YaegiFuncRunner, error) {
	gateway, err := do.Invoke[DIGateway](injector)
	if err != nil {
		return nil, fmt.Errorf("get gateway: %w", err)
	}

	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("use stdlib symbols: %w", err)
	}

	// Export injector for functions that need DI access
	if err := i.Use(interp.Exports{
		"github.com/denkhaus/gollum/pkg/extensions": {
			"injector": reflect.ValueOf(gateway.Injector()),
		},
	}); err != nil {
		return nil, fmt.Errorf("use exports: %w", err)
	}

	return &yaegiFuncRunnerImpl{
		gateway: gateway,
		i:       i,
		funcs:   make(map[string]*funcInfo),
	}, nil
}

func (p *yaegiFuncRunnerImpl) LoadFunc(name, source string) error {
	// Parse source to get package name
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse source: %w", err)
	}

	packageName := f.Name.Name

	// Load into Yaegi interpreter
	if _, err := p.i.Eval(source); err != nil {
		return fmt.Errorf("yaegi eval: %w", err)
	}

	// Find exported functions
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil { // Skip methods
			continue
		}
		if !unicode.IsUpper(rune(fn.Name.Name[0])) { // Skip unexported
			continue
		}

		// Extract parameter names
		paramNames := make([]string, 0)
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					paramNames = append(paramNames, name.Name)
				}
			}
		}

		fullName := packageName + "." + fn.Name.Name
		p.funcs[fullName] = &funcInfo{
			name:       fn.Name.Name,
			pkgName:    packageName,
			paramNames: paramNames,
		}
	}

	return nil
}

func (p *yaegiFuncRunnerImpl) ExecuteFunc(name string, args map[string]any) (any, error) {
	// Try bare name first, then try with "main." prefix (for .gollum/functions/ files)
	info, ok := p.funcs[name]
	if !ok {
		// Try with main package prefix for functions from .gollum/functions/
		info, ok = p.funcs["main."+name]
		if !ok {
			// Not in loaded func files, try standard library (e.g., fmt.Sprint)
			// Try to eval it directly from stdlib
			fnVal, err := p.i.Eval(name)
			if err != nil {
				return nil, fmt.Errorf("function not found: %s", name)
			}

			// For stdlib functions, we don't have parameter names, so we need to infer them
			// Build arg list from function signature
			argList := p.buildArgListFromArgs(fnVal, args)

			// Call the function dynamically
			results := fnVal.Call(argList)

			if len(results) > 0 {
				return results[0].Interface(), nil
			}
			return nil, nil
		}
	}

	// Get function value from interpreter
	fnVal, err := p.i.Eval(info.pkgName + "." + info.name)
	if err != nil {
		return nil, fmt.Errorf("eval function: %w", err)
	}

	// Build argument list using stored parameter names
	argList := p.buildArgList(fnVal, info.paramNames, args)

	// Call the function dynamically
	results := fnVal.Call(argList)

	if len(results) > 0 {
		return results[0].Interface(), nil
	}
	return nil, nil
}

// buildArgListFromArgs builds argument list for stdlib functions where we don't have param names
func (p *yaegiFuncRunnerImpl) buildArgListFromArgs(fnVal reflect.Value, args map[string]any) []reflect.Value {
	fnType := fnVal.Type()
	argList := make([]reflect.Value, fnType.NumIn())

	// For stdlib functions, use args by position ("a", "b", etc.) or just use the values in order
	// Try to match by common param names first
	argNames := []string{"a", "b", "c", "format", "args"}
	for i := 0; i < fnType.NumIn(); i++ {
		var found bool
		// Try each known param name
		for _, argName := range argNames {
			if val, ok := args[argName]; ok {
				argList[i] = p.convertValue(val, fnType.In(i))
				found = true
				break
			}
		}
		if !found {
			// Use zero value if arg not provided
			argList[i] = reflect.Zero(fnType.In(i))
		}
	}

	return argList
}

func (p *yaegiFuncRunnerImpl) ListFuncs() []string {
	names := make([]string, 0, len(p.funcs))
	for name := range p.funcs {
		names = append(names, name)
	}
	return names
}

// buildArgList converts map arguments to ordered slice using parameter names
func (p *yaegiFuncRunnerImpl) buildArgList(fnVal reflect.Value, paramNames []string, args map[string]any) []reflect.Value {
	fnType := fnVal.Type()
	argList := make([]reflect.Value, fnType.NumIn())

	for i := 0; i < fnType.NumIn(); i++ {
		var paramName string
		if i < len(paramNames) {
			paramName = paramNames[i]
		}

		if paramName != "" {
			if val, ok := args[paramName]; ok {
				argList[i] = p.convertValue(val, fnType.In(i))
				continue
			}
		}

		// Use zero value if arg not provided
		argList[i] = reflect.Zero(fnType.In(i))
	}

	return argList
}

// convertValue converts a value to the target type, handling string->int conversions
func (p *yaegiFuncRunnerImpl) convertValue(val any, target reflect.Type) reflect.Value {
	valValue := reflect.ValueOf(val)

	// If already the right type, return as-is
	if valValue.Type().AssignableTo(target) {
		return valValue
	}

	// Handle string -> int conversions (from template substitution)
	if valValue.Kind() == reflect.String && target.Kind() == reflect.Int {
		var i int
		if _, err := fmt.Sscanf(valValue.String(), "%d", &i); err == nil {
			return reflect.ValueOf(i).Convert(target)
		}
	}

	// Try direct conversion
	if valValue.Type().ConvertibleTo(target) {
		return valValue.Convert(target)
	}

	// Fallback: use zero value
	return reflect.Zero(target)
}
