package tools

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
)

func TestAddTool_Run_Success(t *testing.T) {
	tool := &AddTool{}

	args := map[string]any{
		"a": float64(5),
		"b": float64(3),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(8) {
		t.Errorf("Expected result=8, got %v", result["result"])
	}
}

func TestAddTool_Run_NegativeNumbers(t *testing.T) {
	tool := &AddTool{}

	args := map[string]any{
		"a": float64(-10),
		"b": float64(5),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(-5) {
		t.Errorf("Expected result=-5, got %v", result["result"])
	}
}

func TestAddTool_Run_Decimals(t *testing.T) {
	tool := &AddTool{}

	args := map[string]any{
		"a": float64(2.5),
		"b": float64(3.7),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 2.5 + 3.7
	if result["result"] != expected {
		t.Errorf("Expected result=%v, got %v", expected, result["result"])
	}
}

func TestAddTool_Run_Zero(t *testing.T) {
	tool := &AddTool{}

	args := map[string]any{
		"a": float64(0),
		"b": float64(0),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(0) {
		t.Errorf("Expected result=0, got %v", result["result"])
	}
}

func TestAddTool_Spec(t *testing.T) {
	tool := &AddTool{}

	spec := tool.Spec()

	if spec.Name != "add" {
		t.Errorf("Expected tool name 'add', got '%s'", spec.Name)
	}

	if spec.Description != "Adds two numbers together" {
		t.Errorf("Expected description 'Adds two numbers together', got '%s'", spec.Description)
	}

	// Check parameter 'a'
	paramA, exists := spec.Parameters["a"]
	if !exists {
		t.Fatal("Missing 'a' parameter in spec")
	}
	if paramA.Type != gollem.TypeNumber {
		t.Errorf("Expected 'a' parameter type to be Number, got %v", paramA.Type)
	}
	if paramA.Description != "First number" {
		t.Errorf("Expected 'a' parameter description 'First number', got '%s'", paramA.Description)
	}

	// Check parameter 'b'
	paramB, exists := spec.Parameters["b"]
	if !exists {
		t.Fatal("Missing 'b' parameter in spec")
	}
	if paramB.Type != gollem.TypeNumber {
		t.Errorf("Expected 'b' parameter type to be Number, got %v", paramB.Type)
	}
	if paramB.Description != "Second number" {
		t.Errorf("Expected 'b' parameter description 'Second number', got '%s'", paramB.Description)
	}
}

func TestMultiplyTool_Run_Success(t *testing.T) {
	tool := &MultiplyTool{}

	args := map[string]any{
		"a": float64(5),
		"b": float64(3),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(15) {
		t.Errorf("Expected result=15, got %v", result["result"])
	}
}

func TestMultiplyTool_Run_NegativeNumbers(t *testing.T) {
	tool := &MultiplyTool{}

	args := map[string]any{
		"a": float64(-4),
		"b": float64(3),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(-12) {
		t.Errorf("Expected result=-12, got %v", result["result"])
	}
}

func TestMultiplyTool_Run_TwoNegatives(t *testing.T) {
	tool := &MultiplyTool{}

	args := map[string]any{
		"a": float64(-5),
		"b": float64(-2),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(10) {
		t.Errorf("Expected result=10, got %v", result["result"])
	}
}

func TestMultiplyTool_Run_Decimals(t *testing.T) {
	tool := &MultiplyTool{}

	args := map[string]any{
		"a": float64(2.5),
		"b": float64(4),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 2.5 * 4
	if result["result"] != expected {
		t.Errorf("Expected result=%v, got %v", expected, result["result"])
	}
}

func TestMultiplyTool_Run_Zero(t *testing.T) {
	tool := &MultiplyTool{}

	args := map[string]any{
		"a": float64(100),
		"b": float64(0),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["result"] != float64(0) {
		t.Errorf("Expected result=0, got %v", result["result"])
	}
}

func TestMultiplyTool_Spec(t *testing.T) {
	tool := &MultiplyTool{}

	spec := tool.Spec()

	if spec.Name != "multiply" {
		t.Errorf("Expected tool name 'multiply', got '%s'", spec.Name)
	}

	if spec.Description != "Multiplies two numbers together" {
		t.Errorf("Expected description 'Multiplies two numbers together', got '%s'", spec.Description)
	}

	// Check parameter 'a'
	paramA, exists := spec.Parameters["a"]
	if !exists {
		t.Fatal("Missing 'a' parameter in spec")
	}
	if paramA.Type != gollem.TypeNumber {
		t.Errorf("Expected 'a' parameter type to be Number, got %v", paramA.Type)
	}
	if paramA.Description != "First number" {
		t.Errorf("Expected 'a' parameter description 'First number', got '%s'", paramA.Description)
	}

	// Check parameter 'b'
	paramB, exists := spec.Parameters["b"]
	if !exists {
		t.Fatal("Missing 'b' parameter in spec")
	}
	if paramB.Type != gollem.TypeNumber {
		t.Errorf("Expected 'b' parameter type to be Number, got %v", paramB.Type)
	}
	if paramB.Description != "Second number" {
		t.Errorf("Expected 'b' parameter description 'Second number', got '%s'", paramB.Description)
	}
}
