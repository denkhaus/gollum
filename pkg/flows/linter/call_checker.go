package linter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/parser"
)

// CallChecker validates call references to modules and flows
type CallChecker struct {
	Resolver *parser.Resolver
}

// NewCallChecker creates a new call checker
func NewCallChecker() *CallChecker {
	return &CallChecker{
		Resolver: parser.DefaultResolver(),
	}
}

// Check validates all call references in a flow
func (c *CallChecker) Check(flowPath string, flow *flows.Flow, result *flows.LinterResult) {
	// Collect all unique call references
	calls := make(map[string]string) // ref -> location

	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.Ref == "" {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrInvalidTransition,
					Message: "call element missing 'ref' attribute",
				})
				continue
			}

			// Track location for error messages
			loc := fmt.Sprintf("state '%s'", state.Name)
			calls[call.Ref] = loc
		}
	}

	// Resolve each call reference
	for ref, loc := range calls {
		_, err := c.Resolver.ResolveCall(ref, flowPath)
		if err != nil {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrInvalidTransition,
				Message: fmt.Sprintf("unresolved call reference '%s' in %s: %s", ref, loc, err),
			})
		}
	}

	// Validate that modules have main.xml when referenced by module name
	for ref := range calls {
		// Only validate module references (simple names, no slashes)
		if !containsSlash(ref) {
			resolved, err := c.Resolver.ResolveCall(ref, flowPath)
			if err != nil {
				continue // Will be caught by other validation
			}

			// Skip validation if:
			// 1. The call is within the same module (current flow and resolved are in same module dir)
			// 2. The resolved file is already main.xml
			if c.isSameModule(flowPath, resolved) || filepath.Base(resolved) == "main.xml" {
				continue
			}

			// Only validate if the resolved path is in a modules/ directory
			if c.isInModulesDirectory(resolved) {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrInvalidTransition,
					Message: fmt.Sprintf("module '%s' must have main.xml as entry point (found %s)", ref, filepath.Base(resolved)),
				})
			}
		}
	}
}

// isInModulesDirectory checks if a path is inside a modules directory
func (c *CallChecker) isInModulesDirectory(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if path contains /modules/ directory separator
	return strings.Contains(absPath, string(filepath.Separator)+"modules"+string(filepath.Separator)) ||
		strings.Contains(absPath, "/modules/")
}

// isSameModule checks if two paths are within the same module directory
func (c *CallChecker) isSameModule(currentFlowPath, resolvedPath string) bool {
	currentDir := filepath.Dir(currentFlowPath)
	resolvedDir := filepath.Dir(resolvedPath)

	// Normalize paths for comparison
	currentAbs, _ := filepath.Abs(currentDir)
	resolvedAbs, _ := filepath.Abs(resolvedDir)

	return currentAbs == resolvedAbs
}

func containsSlash(s string) bool {
	return len(s) > 0 && (s[0] == '/' || s[len(s)-1] == '/' || indexOfChar(s, '/') != -1)
}

func indexOfChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
