package variables

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
)

// TestFieldDefinition is a test implementation of FieldDefinition
type TestFieldDefinition struct {
	name         string
	typ          string
	defaultValue string
}

func (t TestFieldDefinition) GetName() string         { return t.name }
func (t TestFieldDefinition) GetType() string          { return t.typ }
func (t TestFieldDefinition) GetDefault() string       { return t.defaultValue }

// TestNewFieldValues tests constructor and Has method
func TestNewFieldValues(t *testing.T) {
	defs := []TestFieldDefinition{
		{name: "name", typ: "string", defaultValue: ""},
		{name: "age", typ: "int", defaultValue: "25"},
		{name: "active", typ: "bool", defaultValue: "true"},
		{name: "score", typ: "float", defaultValue: "3.14"},
	}

	fv := NewFieldValues(defs)

	// Test Has method - all fields should be defined
	if !fv.Has("name") {
		t.Error("Expected field 'name' to be defined")
	}
	if !fv.Has("age") {
		t.Error("Expected field 'age' to be defined")
	}
	if !fv.Has("active") {
		t.Error("Expected field 'active' to be defined")
	}
	if !fv.Has("score") {
		t.Error("Expected field 'score' to be defined")
	}

	// Test Has returns false for unknown field
	if fv.Has("unknown") {
		t.Error("Expected 'unknown' field to not be defined")
	}

	// Test default value for age (int)
	age, err := fv.GetInt("age")
	if err != nil {
		t.Errorf("Expected no error getting age with default: %v", err)
	}
	if age != 25 {
		t.Errorf("Expected age default value 25, got %d", age)
	}

	// Test default value for active (bool)
	active, err := fv.GetBool("active")
	if err != nil {
		t.Errorf("Expected no error getting active with default: %v", err)
	}
	if !active {
		t.Error("Expected active default value true, got false")
	}

	// Test default value for score (float)
	score, err := fv.GetFloat("score")
	if err != nil {
		t.Errorf("Expected no error getting score with default: %v", err)
	}
	if score != 3.14 {
		t.Errorf("Expected score default value 3.14, got %f", score)
	}
}

// TestFieldValuesSetString tests SetString and GetString methods
func TestFieldValuesSetString(t *testing.T) {
	defs := []TestFieldDefinition{
			{name: "name", typ: "string", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test SetString
	err := fv.SetString("name", "Alice")
	if err != nil {
		t.Errorf("Expected no error setting string: %v", err)
	}

	// Test GetString
	value, err := fv.GetString("name")
	if err != nil {
		t.Errorf("Expected no error getting string: %v", err)
	}
	if value != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", value)
	}

	// Test SetString on unknown field
	err = fv.SetString("unknown", "value")
	if err == nil {
		t.Error("Expected error setting unknown field")
	}
	_, okAsUnknownErr := err.(*errors.UnknownFieldError)
	if !okAsUnknownErr {
		t.Errorf("Expected UnknownFieldError, got %T", err)
	}

	// Test SetString with wrong type
	defs2 := []TestFieldDefinition{{name: "age", typ: "int", defaultValue: ""}}
	fv2 := NewFieldValues(defs2)
	err = fv2.SetString("age", "not-an-int")
	if err == nil {
		t.Error("Expected error setting string on int field")
	}
	_, okAsTypeError := err.(*errors.TypeError)
	if !okAsTypeError {
		t.Errorf("Expected TypeError, got %T", err)
	}
}

// TestFieldValuesSetInt tests SetInt and GetInt methods
func TestFieldValuesSetInt(t *testing.T) {
	defs := []TestFieldDefinition{
		{name: "age", typ: "int", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test SetInt
	err := fv.SetInt("age", 30)
	if err != nil {
		t.Errorf("Expected no error setting int: %v", err)
	}

	// Test GetInt
	value, err := fv.GetInt("age")
	if err != nil {
		t.Errorf("Expected no error getting int: %v", err)
	}
	if value != 30 {
		t.Errorf("Expected 30, got %d", value)
	}

	// Test SetInt on unknown field
	err = fv.SetInt("unknown", 10)
	if err == nil {
		t.Error("Expected error setting unknown field")
	}

	// Test SetInt with wrong type
	defs2 := []TestFieldDefinition{{name: "name", typ: "string", defaultValue: ""}}
	fv2 := NewFieldValues(defs2)
	err = fv2.SetInt("name", 10)
	if err == nil {
		t.Error("Expected error setting int on string field")
	}
}

// TestFieldValuesSetBool tests SetBool and GetBool methods
func TestFieldValuesSetBool(t *testing.T) {
	defs := []TestFieldDefinition{
		{name: "active", typ: "bool", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test SetBool
	err := fv.SetBool("active", true)
	if err != nil {
		t.Errorf("Expected no error setting bool: %v", err)
	}

	// Test GetBool
	value, err := fv.GetBool("active")
	if err != nil {
		t.Errorf("Expected no error getting bool: %v", err)
	}
	if !value {
		t.Error("Expected true, got false")
	}

	// Test SetBool on unknown field
	err = fv.SetBool("unknown", true)
	if err == nil {
		t.Error("Expected error setting unknown field")
	}

	// Test SetBool with wrong type
	defs2 := []TestFieldDefinition{{name: "name", typ: "string", defaultValue: ""}}
	fv2 := NewFieldValues(defs2)
	err = fv2.SetBool("name", true)
	if err == nil {
		t.Error("Expected error setting bool on string field")
	}
}

// TestFieldValuesSetFloat tests SetFloat and GetFloat methods
func TestFieldValuesSetFloat(t *testing.T) {
	defs := []TestFieldDefinition{
		{name: "score", typ: "float", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test SetFloat
	err := fv.SetFloat("score", 9.5)
	if err != nil {
		t.Errorf("Expected no error setting float: %v", err)
	}

	// Test GetFloat
	value, err := fv.GetFloat("score")
	if err != nil {
		t.Errorf("Expected no error getting float: %v", err)
	}
	if value != 9.5 {
		t.Errorf("Expected 9.5, got %f", value)
	}

	// Test SetFloat on unknown field
	err = fv.SetFloat("unknown", 1.0)
	if err == nil {
		t.Error("Expected error setting unknown field")
	}

	// Test SetFloat with wrong type
	defs2 := []TestFieldDefinition{{name: "name", typ: "string", defaultValue: ""}}
	fv2 := NewFieldValues(defs2)
	err = fv2.SetFloat("name", 1.0)
	if err == nil {
		t.Error("Expected error setting float on string field")
	}
}

// TestFieldValuesSetFromString tests SetFromString method with type conversion
func TestFieldValuesSetFromString(t *testing.T) {
	defs := []TestFieldDefinition{
			{name: "name", typ: "string", defaultValue: ""},
		{name: "age", typ: "int", defaultValue: ""},
		{name: "active", typ: "bool", defaultValue: ""},
		{name: "score", typ: "float", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test string conversion
	err := fv.SetFromString("name", "Bob")
	if err != nil {
		t.Errorf("Expected no error setting string from string: %v", err)
	}
	value, _ := fv.GetString("name")
	if value != "Bob" {
		t.Errorf("Expected 'Bob', got '%s'", value)
	}

	// Test int conversion
	err = fv.SetFromString("age", "42")
	if err != nil {
		t.Errorf("Expected no error setting int from string: %v", err)
	}
	age, _ := fv.GetInt("age")
	if age != 42 {
		t.Errorf("Expected 42, got %d", age)
	}

	// Test bool conversion
	err = fv.SetFromString("active", "true")
	if err != nil {
		t.Errorf("Expected no error setting bool from string: %v", err)
	}
	active, _ := fv.GetBool("active")
	if !active {
		t.Error("Expected true, got false")
	}

	// Test float conversion
	err = fv.SetFromString("score", "8.75")
	if err != nil {
		t.Errorf("Expected no error setting float from string: %v", err)
	}
	score, _ := fv.GetFloat("score")
	if score != 8.75 {
		t.Errorf("Expected 8.75, got %f", score)
	}

	// Test invalid int conversion
	err = fv.SetFromString("age", "not-a-number")
	if err == nil {
		t.Error("Expected error for invalid int conversion")
	}

	// Test invalid bool conversion
	err = fv.SetFromString("active", "not-a-bool")
	if err == nil {
		t.Error("Expected error for invalid bool conversion")
	}

	// Test invalid float conversion
	err = fv.SetFromString("score", "not-a-float")
	if err == nil {
		t.Error("Expected error for invalid float conversion")
	}

	// Test unknown field
	err = fv.SetFromString("unknown", "value")
	if err == nil {
		t.Error("Expected error for unknown field")
	}
}

// TestFieldValuesGetRaw tests GetRaw method
func TestFieldValuesGetRaw(t *testing.T) {
	defs := []TestFieldDefinition{
			{name: "name", typ: "string", defaultValue: ""},
		{name: "age", typ: "int", defaultValue: ""},
	}
	fv := NewFieldValues(defs)

	// Test GetRaw on unset field
	value, ok := fv.GetRaw("name")
	if ok {
		t.Error("Expected ok=false for unset field")
	}
	if value != nil {
		t.Errorf("Expected nil value for unset field, got %v", value)
	}

	// Test GetRaw on set field
	fv.SetString("name", "Charlie")
	value, ok = fv.GetRaw("name")
	if !ok {
		t.Error("Expected ok=true for set field")
	}
	if value != "Charlie" {
		t.Errorf("Expected 'Charlie', got %v", value)
	}

	// Test GetRaw with int
	fv.SetInt("age", 35)
	value, ok = fv.GetRaw("age")
	if !ok {
		t.Error("Expected ok=true for set int field")
	}
	if value != 35 {
		t.Errorf("Expected 35, got %v", value)
	}

	// Test GetRaw on unknown field
	value, ok = fv.GetRaw("unknown")
	if ok {
		t.Error("Expected ok=false for unknown field")
	}
}

// TestFieldValuesWithFlowsTypes tests compatibility with flows.FieldDef
func TestFieldValuesWithFlowsTypes(t *testing.T) {
	defs := []flows.FieldDef{
		{Name: "title", Type: "string", Default: ""},
		{Name: "count", Type: "int", Default: "10"},
	}

	fv := NewFieldValues(defs)

	// Test Has
	if !fv.Has("title") {
		t.Error("Expected 'title' field to be defined")
	}

	// Test default value
	count, err := fv.GetInt("count")
	if err != nil {
		t.Errorf("Expected no error getting count with default: %v", err)
	}
	if count != 10 {
		t.Errorf("Expected default count 10, got %d", count)
	}

	// Test setting value
	err = fv.SetString("title", "Test")
	if err != nil {
		t.Errorf("Expected no error setting title: %v", err)
	}

	title, err := fv.GetString("title")
	if err != nil {
		t.Errorf("Expected no error getting title: %v", err)
	}
	if title != "Test" {
		t.Errorf("Expected 'Test', got '%s'", title)
	}
}
