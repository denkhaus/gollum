package variables

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// OutputBinding represents a declarative output field binding
type OutputBinding struct {
	TargetName  string          // output field name
	SourceScope string          // "input", "context", "computed", "output"
	SourceName  string          // source field name
	ValueType   flows.ValueType // type of the value
}

// ParseFieldReference parses a field reference like "computed.sum" into scope and name
func ParseFieldReference(ref string) (scope string, name string, err error) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid field reference format: '%s' (expected 'scope.fieldname')", ref)
	}

	scope = parts[0]
	name = parts[1]

	if scope == "" || name == "" {
		return "", "", fmt.Errorf("invalid field reference: '%s' (empty scope or name)", ref)
	}

	return scope, name, nil
}

// IsValidSourceScope checks if the scope is valid for output bindings
func IsValidSourceScope(scope string) bool {
	switch scope {
	case "input", "context", "computed", "output":
		return true
	default:
		return false
	}
}
