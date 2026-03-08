package registry

import (
	"fmt"
	"strings"
)

// Param represents a function parameter
type Param struct {
	Name string
	Type string
}

// Func is the function signature for executable functions
type Func func(args []any) (any, error)

// FunctionSignature represents a function's signature and implementation
type FunctionSignature struct {
	Name       string
	Params     []Param
	ReturnType string
	Variadic   bool
	Func       Func
}

// Registry holds function signatures and implementations
type Registry struct {
	functions map[string]FunctionSignature
}

// NewRegistry creates a new empty function registry
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]FunctionSignature),
	}
}

// Register adds a function to the registry
func (r *Registry) Register(name string, sig FunctionSignature) error {
	if _, exists := r.functions[name]; exists {
		return fmt.Errorf("function %s already registered", name)
	}
	r.functions[name] = sig
	return nil
}

// Lookup finds a function by name
func (r *Registry) Lookup(name string) (FunctionSignature, bool) {
	sig, ok := r.functions[name]
	return sig, ok
}

// ValidateCall checks if the provided arguments match the function signature
func (r *Registry) ValidateCall(name string, args map[string]any) error {
	sig, ok := r.Lookup(name)
	if !ok {
		return fmt.Errorf("function %s not found", name)
	}

	// Check required parameters
	for _, param := range sig.Params {
		if _, exists := args[param.Name]; !exists {
			return fmt.Errorf("missing required parameter: %s", param.Name)
		}
	}

	return nil
}

// Execute calls a function with the provided arguments
func (r *Registry) Execute(name string, args map[string]any) (any, error) {
	sig, ok := r.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("function %s not found", name)
	}

	if sig.Func == nil {
		return nil, fmt.Errorf("function %s has no implementation", name)
	}

	// Build argument list in parameter order
	var argList []any
	for _, param := range sig.Params {
		argList = append(argList, args[param.Name])
	}

	// Handle variadic functions
	if sig.Variadic {
		if variadicArgs, exists := args["args"]; exists {
			if va, ok := variadicArgs.([]any); ok {
				argList = append(argList, va...)
			}
		}
	}

	return sig.Func(argList)
}

// builtinRegistry is the global built-in function registry
var builtinRegistry = NewRegistry()

// GetBuiltinRegistry returns the built-in function registry
func GetBuiltinRegistry() *Registry {
	return builtinRegistry
}

// init registers all built-in stdlib functions
func init() {
	// strings package
	builtinRegistry.Register("strings.ToUpper", FunctionSignature{
		Name:       "ToUpper",
		Params:     []Param{{Name: "s", Type: "string"}},
		ReturnType: "string",
		Func: func(args []any) (any, error) {
			return strings.ToUpper(args[0].(string)), nil
		},
	})

	builtinRegistry.Register("strings.ToLower", FunctionSignature{
		Name:       "ToLower",
		Params:     []Param{{Name: "s", Type: "string"}},
		ReturnType: "string",
		Func: func(args []any) (any, error) {
			return strings.ToLower(args[0].(string)), nil
		},
	})

	builtinRegistry.Register("strings.Contains", FunctionSignature{
		Name:       "Contains",
		Params:     []Param{{Name: "s", Type: "string"}, {Name: "substr", Type: "string"}},
		ReturnType: "bool",
		Func: func(args []any) (any, error) {
			return strings.Contains(args[0].(string), args[1].(string)), nil
		},
	})

	builtinRegistry.Register("strings.HasPrefix", FunctionSignature{
		Name:       "HasPrefix",
		Params:     []Param{{Name: "s", Type: "string"}, {Name: "prefix", Type: "string"}},
		ReturnType: "bool",
		Func: func(args []any) (any, error) {
			return strings.HasPrefix(args[0].(string), args[1].(string)), nil
		},
	})

	builtinRegistry.Register("strings.HasSuffix", FunctionSignature{
		Name:       "HasSuffix",
		Params:     []Param{{Name: "s", Type: "string"}, {Name: "suffix", Type: "string"}},
		ReturnType: "bool",
		Func: func(args []any) (any, error) {
			return strings.HasSuffix(args[0].(string), args[1].(string)), nil
		},
	})

	// fmt package
	builtinRegistry.Register("fmt.Sprintf", FunctionSignature{
		Name:       "Sprintf",
		Params:     []Param{{Name: "format", Type: "string"}, {Name: "args", Type: "any"}},
		ReturnType: "string",
		Variadic:   true,
		Func: func(args []any) (any, error) {
			format := args[0].(string)
			variadicArgs := args[1:]
			if len(args) > 2 {
				// args[1] is the "args" parameter which should be []any
				if va, ok := args[1].([]any); ok {
					variadicArgs = va
				}
			}
			return fmt.Sprintf(format, variadicArgs...), nil
		},
	})

	// len function
	builtinRegistry.Register("len", FunctionSignature{
		Name:       "len",
		Params:     []Param{{Name: "v", Type: "any"}},
		ReturnType: "int",
		Func: func(args []any) (any, error) {
			v := args[0]
			switch val := v.(type) {
			case string:
				return len(val), nil
			case []any:
				return len(val), nil
			case []int:
				return len(val), nil
			case []string:
				return len(val), nil
			default:
				return 0, fmt.Errorf("unsupported type for len: %T", v)
			}
		},
	})
}
