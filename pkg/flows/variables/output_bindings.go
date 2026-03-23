package variables

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// OutputBinding represents a declarative output field binding
type OutputBinding struct {
	TargetName  string                  // output field name
	SourceScope flows.FlowVariableScope // "input", "context", "computed", "output"
	SourceName  string                  // source field name
	ValueType   flows.ValueType         // type of the value
}

// ParseFieldReference parses a field reference like "computed.sum" into scope and name
func ParseFieldReference(ref string) (flows.FlowVariableScope, string, error) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid field reference format: '%s' (expected 'scope.fieldname')", ref)
	}

	scope := flows.FlowVariableScope(parts[0])
	name := parts[1]

	if scope == "" || name == "" {
		return "", "", fmt.Errorf("invalid field reference: '%s' (empty scope or name)", ref)
	}

	return scope, name, nil
}
