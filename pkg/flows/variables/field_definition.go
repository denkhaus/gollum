package variables

import "github.com/denkhaus/gollum/pkg/flows"

// FieldDefinition defines the interface for field metadata.
// This interface allows the generic FieldValues[T] to work with different
// field types (flows.FieldDef, flows.ContextField) while maintaining type safety.
//
// The flows.FieldDef and flows.ContextField types implement this interface
// via methods defined in pkg/flows/types.go.
type FieldDefinition interface {
	GetName() string
	GetType() flows.ValueType
	GetDefault() string
}

// Ensure flows.FieldDef implements FieldDefinition
var _ FieldDefinition = flows.FieldDef{}

// Ensure flows.ContextField implements FieldDefinition
var _ FieldDefinition = flows.ContextField{}
