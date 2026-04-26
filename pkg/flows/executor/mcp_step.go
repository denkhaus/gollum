package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/objx"
)

// executeMCPStep executes an MCP (Model Context Protocol) step.
func (p *flowExecutorImpl) executeMCPStep(_ context.Context, step *flows.Step, stateName string) error {
	// step.Tool format: "server/tool" (e.g., "tavily/search")
	// We need to parse this to match against tool specs which only have the tool name
	toolName := step.Tool

	// Build args from step params by resolving bare notation references
	args := make(map[string]any)
	for _, param := range step.Params {
		// Resolve bare notation reference
		value := p.resolveAssignFrom(param.AssignFrom)
		args[param.Name] = value
	}

	// Get all available MCP tool sets
	toolSets := p.mcpRegistry.GetToolSets()

	// Find the tool by name in any of the available tool sets
	for _, toolSet := range toolSets {
		// Get tool specs from this tool set
		ctx := context.Background()
		specs, err := toolSet.Specs(ctx)
		if err != nil {
			continue
		}

		// Check if any spec matches our tool name
		for _, spec := range specs {
			// Match against the full server/tool format or just the tool name
			// The flow uses "server/tool" format, but spec.Name is just "tool"
			// We need to check if our toolName ends with the spec name
			if toolName == spec.Name || strings.HasSuffix(toolName, "/"+spec.Name) {
				// Found the tool, execute it using the spec name (not the full server/tool)
				result, err := toolSet.Run(ctx, spec.Name, args)
				if err != nil {
					return &MCPError{
						Server: toolName,
						Tool:   spec.Name,
						Step:   stateName,
						Err:    err,
					}
				}

				// Unwrap MCP result format if needed
				// MCP tools return: {"content": [{"type": "text", "text": "{"key": ...}"}]}
				// We need to extract the actual data from content[0].text and parse it as JSON
				result = p.unwrapMCPResult(result)
				if err != nil {
					return &MCPError{
						Server: toolName,
						Tool:   spec.Name,
						Step:   stateName,
						Err:    fmt.Errorf("failed to unwrap MCP result: %w", err),
					}
				}

				// Map result to output fields if specified
				if step.Result != nil {
					// Handle simple assign
					if step.Result.AssignTo != "" {
						scope, fieldName, err := p.parseAssignTarget(step.Result.AssignTo)
						if err != nil {
							return &MCPError{
								Server: toolName,
								Tool:   toolName,
								Step:   stateName,
								Err:    fmt.Errorf("invalid assignTo: %w", err),
							}
						}
						// For MCP tools, we'll map the entire result to the field
						if scope == flows.FlowVariableScopeContext {
							if err := p.ctx.SetContextField(fieldName, shared.AnyToString(result)); err != nil {
								return fmt.Errorf("failed to set context field '%s': %w", fieldName, err)
							}
						} else {
							if err := p.ctx.SetOutputField(fieldName, shared.AnyToString(result)); err != nil {
								return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
							}
						}
					}

					// Handle path-based outputs with JSONPath extraction
					for _, path := range step.Result.Paths {
						if path.AssignTo == "" || path.Path == "" {
							continue
						}

						scope, fieldName, err := p.parseAssignTarget(path.AssignTo)
						if err != nil {
							return &MCPError{
								Server: toolName,
								Tool:   toolName,
								Step:   stateName,
								Err:    fmt.Errorf("invalid assignTo for path '%s': %w", path.Path, err),
							}
						}

						// Extract value using JSONPath-like syntax
						extracted, err := p.extractJSONPath(result, path.Path)
						if err != nil {
							// Log the MCP result to help debug flow definitions
							p.logService.Errorf("MCP tool result for %s (path: %s): %+v", toolName, path.Path, result)
							return &MCPError{
								Server: toolName,
								Tool:   toolName,
								Step:   stateName,
								Err:    fmt.Errorf("failed to extract path '%s': %w", path.Path, err),
							}
						}

						if scope == flows.FlowVariableScopeContext {
							if err := p.ctx.SetContextField(fieldName, shared.AnyToString(extracted)); err != nil {
								return fmt.Errorf("failed to set context field '%s': %w", fieldName, err)
							}
						} else {
							if err := p.ctx.SetOutputField(fieldName, shared.AnyToString(extracted)); err != nil {
								return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
							}
						}
					}
				}

				return nil
			}
		}
	}

	return &MCPError{
		Server: toolName,
		Tool:   toolName,
		Step:   stateName,
		Err:    fmt.Errorf("tool not found"),
	}
}

// unwrapMCPResult unwraps the MCP protocol result format.
// MCP tools can return different formats:
// 1. {"Result": {...}} - Direct result map
// 2. {"content": [{"type": "text", "text": "{\"key\": ...}"}]} - Standard MCP format
// This function extracts the actual data from whichever format is present
func (p *flowExecutorImpl) unwrapMCPResult(result map[string]any) (map[string]any) {
	// First, check for "Result" key (some MCP servers return data directly under "Result")
	if resultMap, hasResult := result["Result"]; hasResult {
		if resultMapMap, ok := resultMap.(map[string]any); ok {
			return resultMapMap
		}
	}

	// Fall back to standard MCP "content" array format
	contentField, hasContent := result["content"]
	if !hasContent {
		// No content field, return result as-is
		return result
	}

	// content should be an array
	contentArray, ok := contentField.([]any)
	if !ok || len(contentArray) == 0 {
		// Invalid content format, return result as-is
		return result
	}

	// Get first content item
	firstContent := contentArray[0]
	contentMap, ok := firstContent.(map[string]any)
	if !ok {
		// Not a map, return result as-is
		return result
	}

	// Check for "text" field
	textField, hasText := contentMap["text"]
	if !hasText {
		// No text field, return result as-is
		return result
	}

	// Text should be a JSON string
	textStr, ok := textField.(string)
	if !ok {
		// Not a string, return result as-is
		return result
	}

	// Parse the JSON string
	var parsedResult map[string]any
	if err := json.Unmarshal([]byte(textStr), &parsedResult); err != nil {
		// Failed to parse, return original result
		return result
	}

	return parsedResult
}

// extractJSONPath extracts a value from data using JSONPath-like syntax.
// Uses the objx library for robust field access.
// Supports:
//   - field - extracts top-level field
//   - field.nested - extracts nested field
//   - field[0] - extracts array element
//   - field[0].nested - extracts field from array element
//
// Returns: extracted value or error
func (p *flowExecutorImpl) extractJSONPath(data any, path string) (any, error) {
	// Remove leading $ if present (objx doesn't use $ prefix)
	path = strings.TrimPrefix(path, "$")

	// Remove leading dot if present (objx doesn't use dot prefix for root fields)
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return data, nil
	}

	// Use objx for robust path access
	m := objx.New(data)
	result := m.Get(path)

	// Check if the result exists (objx returns nil for missing paths)
	if result.IsNil() {
		return nil, fmt.Errorf("path '%s' not found", path)
	}

	return result.Data(), nil
}
