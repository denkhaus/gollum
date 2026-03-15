package variables

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
)

func TestFieldDefinitionInterface(t *testing.T) {
	// Test that FieldDef implements FieldDefinition
	var fd FieldDefinition = flows.FieldDef{
		Name:    "test_field",
		Type:    "string",
		Default: "default_value",
	}
	if fd.GetName() != "test_field" {
		t.Errorf("expected GetName() to return 'test_field', got '%s'", fd.GetName())
	}
	if fd.GetType() != "string" {
		t.Errorf("expected GetType() to return 'string', got '%s'", fd.GetType())
	}
	if fd.GetDefault() != "default_value" {
		t.Errorf("expected GetDefault() to return 'default_value', got '%s'", fd.GetDefault())
	}
}

func TestContextFieldImplementsFieldDefinition(t *testing.T) {
	var fd FieldDefinition = flows.ContextField{
		Name:    "context_field",
		Type:    "int",
		Default: "42",
	}
	if fd.GetName() != "context_field" {
		t.Errorf("expected GetName() to return 'context_field', got '%s'", fd.GetName())
	}
}
