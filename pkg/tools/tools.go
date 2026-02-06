package tools

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// AddTool is a tool that adds two numbers.
// This is sample code only. Not used currently in the app.
type AddTool struct{}

// Run executes the AddTool to add two numbers
func (t *AddTool) Run(_ context.Context, args map[string]any) (map[string]any, error) {
	a := args["a"].(float64)
	b := args["b"].(float64)
	result := a + b
	return map[string]any{"result": result}, nil
}

// Spec returns the tool specification for the AddTool
func (t *AddTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "add",
		Description: "Adds two numbers together",
		Parameters: map[string]*gollem.Parameter{
			"a": {
				Type:        gollem.TypeNumber,
				Description: "First number",
			},
			"b": {
				Type:        gollem.TypeNumber,
				Description: "Second number",
			},
		},
	}
}

// MultiplyTool is a tool that multiplies two numbers
type MultiplyTool struct{}

// Run executes the MultiplyTool to multiply two numbers
func (t *MultiplyTool) Run(_ context.Context, args map[string]any) (map[string]any, error) {
	a := args["a"].(float64)
	b := args["b"].(float64)
	result := a * b
	return map[string]any{"result": result}, nil
}

// Spec returns the tool specification for the MultiplyTool
func (t *MultiplyTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "multiply",
		Description: "Multiplies two numbers together",
		Parameters: map[string]*gollem.Parameter{
			"a": {
				Type:        gollem.TypeNumber,
				Description: "First number",
			},
			"b": {
				Type:        gollem.TypeNumber,
				Description: "Second number",
			},
		},
	}
}
