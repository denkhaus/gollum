package linter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/parser"
)

// ModuleLinterResult contains the combined results of linting an entire module tree
type ModuleLinterResult struct {
	Valid    bool
	Flows    map[string]*flows.LinterResult // map[flowPath]result
	Errors   []flows.LinterError
	Warnings []flows.LinterError
}

// String returns a formatted summary of the module lint result
func (r *ModuleLinterResult) String() string {
	var sb strings.Builder
	for path, flowResult := range r.Flows {
		if flowResult.Valid {
			fmt.Fprintf(&sb, "✓ %s: valid\n", path)
		} else {
			fmt.Fprintf(&sb, "✗ %s: invalid\n", path)
			for _, e := range flowResult.Errors {
				fmt.Fprintf(&sb, "  %s\n", e.String())
			}
		}
		// Always show warnings
		if len(flowResult.Warnings) > 0 {
			for _, w := range flowResult.Warnings {
				fmt.Fprintf(&sb, "  Warning: %s\n", w.String())
			}
		}
	}
	return sb.String()
}

// LintModule lints a module directory, starting from main.xml and recursively
// linting all referenced flows
func LintModule(modulePath string) *ModuleLinterResult {
	result := &ModuleLinterResult{
		Flows: make(map[string]*flows.LinterResult),
	}

	// Check if path is a directory
	info, err := os.Stat(modulePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidExpr,
			Message: fmt.Sprintf("cannot access module path: %v", err),
		})
		return result
	}

	var entryPoint string
	if info.IsDir() {
		// Look for main.xml as entry point
		entryPoint = filepath.Join(modulePath, "main.xml")
		if _, err := os.Stat(entryPoint); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrInvalidExpr,
				Message: fmt.Sprintf("module entry point not found: %s", entryPoint),
			})
			return result
		}
	} else {
		// Single file provided
		entryPoint = modulePath
	}

	// Lint recursively starting from entry point
	visited := make(map[string]bool)
	result.lintFlowRecursive(entryPoint, visited)

	// Collect all errors and warnings
	result.Valid = true
	for _, flowResult := range result.Flows {
		if !flowResult.Valid {
			result.Valid = false
		}
		result.Errors = append(result.Errors, flowResult.Errors...)
		result.Warnings = append(result.Warnings, flowResult.Warnings...)
	}

	return result
}

// lintFlowRecursive recursively lints a flow and all its dependencies
func (r *ModuleLinterResult) lintFlowRecursive(flowPath string, visited map[string]bool) {
	// Normalize path
	absPath, err := filepath.Abs(flowPath)
	if err != nil {
		absPath = flowPath
	}

	// Skip if already visited (prevents infinite recursion)
	if visited[absPath] {
		return
	}
	visited[absPath] = true

	// Read the flow file for validation
	data, err := os.ReadFile(absPath)
	if err != nil {
		r.Flows[absPath] = &flows.LinterResult{
			Valid: false,
			Errors: []flows.LinterError{
				{
					Code:    flows.ErrXMLParse,
					Message: fmt.Sprintf("read error: %v", err),
				},
			},
		}
		return
	}

	// Phase 0: XML structure validation (before parsing)
	structValidator := NewXMLStructureChecker()
	preParseResult := &flows.LinterResult{}
	structValidator.CheckRawXML(string(data), preParseResult)
	if len(preParseResult.Errors) > 0 {
		r.Flows[absPath] = preParseResult
		return
	}

	// Parse the flow
	flow, err := parser.ParseBytes(data)
	if err != nil {
		r.Flows[absPath] = &flows.LinterResult{
			Valid: false,
			Errors: []flows.LinterError{
				{
					Code:    flows.ErrXMLParse,
					Message: fmt.Sprintf("parse error: %v", err),
				},
			},
		}
		return
	}

	// Lint this flow
	flowResult := LintPath(absPath, flow)
	r.Flows[absPath] = flowResult

	// Resolve and lint all called flows
	resolver := parser.DefaultResolver()
	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.Ref == "" {
				continue
			}

			// Resolve the call reference
			resolvedPath, err := resolver.ResolveCall(call.Ref, absPath)
			if err != nil {
				// This error is already caught by CallChecker
				continue
			}

			// Recursively lint the called flow
			r.lintFlowRecursive(resolvedPath, visited)
		}
	}
}
