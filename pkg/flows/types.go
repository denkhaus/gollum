package flows

import (
	"encoding/xml"

	"github.com/denkhaus/gollum/pkg/shared"
)

// FlowContext defines the interface for accessing flow execution state
type FlowContext interface {
	// SetOutputField sets an output field value
	SetOutputField(name string, value any) error
	// GetOutputField retrieves an output field value
	GetOutputField(name string) (any, error)
	// SetContextField sets a context field value
	SetContextField(name string, value any) error
	// GetContextField retrieves a context field value
	GetContextField(name string) (any, error)
	// GetCurrentState returns the current state name
	GetCurrentState() string
	// GetAllContextFields returns all context fields
	GetAllContextFields() map[string]any
	// ValidateTransition checks if a transition is allowed
	ValidateTransition(from, to string) error
	// RequestTransition signals that the flow should transition to the target state
	RequestTransition(to string) error
}

type VarContainerTarget string

// ValueType represents the type of a field
type ValueType string

func (p ValueType) IsEmpty() bool {
	return p == ""
}

const (
	VarContainerTargetInput   VarContainerTarget = "input"
	VarContainerTargetContext VarContainerTarget = "context"
	VarContainerTargetOutput  VarContainerTarget = "output"

	// Field type constants
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
	TypeBool   ValueType = "bool"
	TypeFloat  ValueType = "float"
	TypeArray  ValueType = "array"
	TypeMap    ValueType = "map"
	TypeObject ValueType = "object"
)

// Flow represents a complete workflow definition
type Flow struct {
	XMLName     xml.Name       `xml:"flow"`
	Name        string         `xml:"name,attr"`
	Version     string         `xml:"version,attr"`
	Description string         `xml:"description"`
	Input       *InputBlock    `xml:"input"`
	Output      *OutputBlock   `xml:"output"`
	Context     *ContextBlock  `xml:"context"`
	Computed    *ComputedBlock `xml:"computed"`
	Agents      []Agent        `xml:"agents>agent"`
	States      []State        `xml:"states>state"`
}

// GetName implements shared.FlowInfo
func (f *Flow) GetName() string {
	return f.Name
}

// GetOutputFields implements shared.FlowInfo
func (f *Flow) GetOutputFields() []FieldDef {
	if f.Output == nil {
		return nil
	}

	return f.Output.GetAllFields()
}

// InputBlock defines the flow's input interface
type InputBlock struct {
	Strings []FieldDef  `xml:"string"`
	Ints    []FieldDef  `xml:"int"`
	Bools   []FieldDef  `xml:"bool"`
	Floats  []FieldDef  `xml:"float"`
	Arrays  []FieldDef  `xml:"array"`
	Maps    []FieldDef  `xml:"map"`
	Objects []ObjectDef `xml:"object"`
}

// GetAllFields returns all input fields as a slice
func (i *InputBlock) GetAllFields() []FieldDef {
	var fields []FieldDef
	for _, f := range i.Strings {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeString,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, f := range i.Ints {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeInt,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, f := range i.Bools {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeBool,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, f := range i.Floats {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeFloat,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, f := range i.Arrays {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeArray,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, f := range i.Maps {
		fields = append(fields, FieldDef{
			XMLName:  f.XMLName,
			Name:     f.Name,
			Type:     TypeMap,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	for _, obj := range i.Objects {
		fields = append(fields, FieldDef{
			XMLName:  obj.XMLName,
			Name:     obj.Name,
			Type:     TypeObject,
			Required: false,
			Default:  obj.Default,
		})
	}
	return fields
}

// OutputBlock defines the flow's output interface
type OutputBlock struct {
	Strings []FieldDef  `xml:"string"`
	Ints    []FieldDef  `xml:"int"`
	Bools   []FieldDef  `xml:"bool"`
	Floats  []FieldDef  `xml:"float"`
	Objects []ObjectDef `xml:"object"`
}

// GetAllFields returns all output fields as a slice
func (o *OutputBlock) GetAllFields() []FieldDef {
	var fields []FieldDef
	for _, f := range o.Strings {
		f.Type = TypeString
		fields = append(fields, f)
	}
	for _, f := range o.Ints {
		f.Type = TypeInt
		fields = append(fields, f)
	}
	for _, f := range o.Bools {
		f.Type = TypeBool
		fields = append(fields, f)
	}
	for _, f := range o.Floats {
		f.Type = TypeFloat
		fields = append(fields, f)
	}
	for _, obj := range o.Objects {
		fields = append(fields, FieldDef{
			XMLName:  obj.XMLName,
			Name:     obj.Name,
			Type:     TypeObject,
			Required: false,
		})
	}
	return fields
}

// GetDeclarative returns all output fields with a 'from' attribute
func (o *OutputBlock) GetDeclarative() []FieldDef {
	var fields []FieldDef
	for _, f := range o.GetAllFields() {
		if f.From != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// GetImperative returns all output fields without a 'from' attribute
func (o *OutputBlock) GetImperative() []FieldDef {
	var fields []FieldDef
	for _, f := range o.GetAllFields() {
		if f.From == "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// ContextBlock defines internal context fields (mutable by tools)
type ContextBlock struct {
	Strings []ContextField `xml:"string"`
	Ints    []ContextField `xml:"int"`
	Bools   []ContextField `xml:"bool"`
	Floats  []ContextField `xml:"float"`
	Objects []ObjectDef    `xml:"object"`
}

// GetAllFields returns all context fields as a slice
func (c *ContextBlock) GetAllFields() []ContextField {
	fields := make([]ContextField, 0)
	for _, f := range c.Strings {
		f.Type = TypeString
		fields = append(fields, f)
	}
	for _, f := range c.Ints {
		f.Type = TypeInt
		fields = append(fields, f)
	}
	for _, f := range c.Bools {
		f.Type = TypeBool
		fields = append(fields, f)
	}
	for _, f := range c.Floats {
		f.Type = TypeFloat
		fields = append(fields, f)
	}
	return fields
}

// ObjectDef represents nested object fields
type ObjectDef struct {
	XMLName xml.Name
	Name    string     `xml:"name,attr"`
	Type    ValueType  `xml:"type,attr"`
	Default string     `xml:"default,attr"`
	Fields  []FieldDef `xml:",any"`
}

// FieldDef is a base type for field definitions
type FieldDef struct {
	XMLName  xml.Name
	Name     string    `xml:"name,attr"`
	Type     ValueType `xml:"type,attr"`
	Required bool      `xml:"required,attr"`
	Default  string    `xml:"default,attr"`
	From     string    `xml:"from,attr,omitempty"` // Source reference for output bindings
}

// GetName returns the field name (implements variables.FieldDefinition interface)
func (f FieldDef) GetName() string {
	return f.Name
}

// GetType returns the field type (implements variables.FieldDefinition interface)
func (f FieldDef) GetType() ValueType {
	return f.Type
}

// GetDefault returns the default value (implements variables.FieldDefinition interface)
func (f FieldDef) GetDefault() string {
	return f.Default
}

// ContextField represents a regular context field
type ContextField struct {
	XMLName xml.Name
	Name    string    `xml:"name,attr"`
	Type    ValueType // Set programmatically, not from XML (element name defines type)
	Default string    `xml:"default,attr"`
}

// GetName returns the field name (implements variables.FieldDefinition interface)
func (c ContextField) GetName() string {
	return c.Name
}

// GetType returns the field type (implements variables.FieldDefinition interface)
func (c ContextField) GetType() ValueType {
	return c.Type
}

// GetDefault returns the default value (implements variables.FieldDefinition interface)
func (c ContextField) GetDefault() string {
	return c.Default
}

// ComputedField represents a computed context field (deprecated - use ComputedBlock)
type ComputedField struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Eval string `xml:"eval,attr"` // Expression to compute value
}

// ComputedBlock defines computed fields at the flow level
type ComputedBlock struct {
	Strings []ComputedFieldDef `xml:"string"`
	Ints    []ComputedFieldDef `xml:"int"`
	Bools   []ComputedFieldDef `xml:"bool"`
	Floats  []ComputedFieldDef `xml:"float"`
}

// GetAllFields returns all computed field definitions as a slice
func (c *ComputedBlock) GetAllFields() []ComputedFieldDef {
	var fields []ComputedFieldDef
	for _, f := range c.Strings {
		field := f
		if field.Type.IsEmpty() {
			field.Type = TypeString
		}
		fields = append(fields, field)
	}
	for _, f := range c.Ints {
		field := f
		if field.Type.IsEmpty() {
			field.Type = TypeInt
		}
		fields = append(fields, field)
	}
	for _, f := range c.Bools {
		field := f
		if field.Type.IsEmpty() {
			field.Type = TypeBool
		}
		fields = append(fields, field)
	}
	for _, f := range c.Floats {
		field := f
		if field.Type.IsEmpty() {
			field.Type = TypeFloat
		}
		fields = append(fields, field)
	}
	return fields
}

// ComputedFieldDef defines a computed field with name, type, and evaluation expression
type ComputedFieldDef struct {
	Name string    `xml:"name,attr"`
	Type ValueType // Set programmatically, not from XML (element name defines type)
	Eval string    `xml:"eval,attr"` // Expression to evaluate
}

// Agent defines an LLM agent
type Agent struct {
	Name        string  `xml:"name,attr"`
	Model       string  `xml:"model,attr"`
	Prompt      string  `xml:"prompt"`
	Temperature float64 `xml:"temperature"`
	TopP        float64 `xml:"top_p"`
	MaxTokens   int     `xml:"max_tokens"`
}

func (p Agent) ToClientConfig() *shared.LLMClientConfig {

	cnf := &shared.LLMClientConfig{
		Model: p.Model,
	}

	if p.MaxTokens != 0 {
		cnf.MaxTokens = &p.MaxTokens
	}

	if p.Temperature != 0.0 {
		cnf.Temperature = &p.Temperature
	}

	if p.TopP != 0.0 {
		cnf.TopP = &p.TopP
	}

	return cnf
}

// State represents a state in the workflow
type State struct {
	Name        string       `xml:"name,attr"`
	Initial     bool         `xml:"initial,attr"`
	Steps       []Step       `xml:"steps>step"`
	Calls       []Call       `xml:"steps>call"`
	Transitions []Transition `xml:"transitions>transition"`
}

// Step is a single execution step
type Step struct {
	XMLName  xml.Name
	Type     string             `xml:"type,attr"`
	Name     string             `xml:"name,attr"`
	Agent    string             `xml:"agent,attr"`
	Function string             `xml:"function,attr"`
	Tool     string             `xml:"tool,attr"`
	Prompt   string             `xml:"prompt"`
	Cmd      string             `xml:"cmd"`
	Tools    string             `xml:"tools"`
	Timeout  string             `xml:"timeout"`
	Params   []StepParam        `xml:"params>param"`
	OnError  *OnErrorTransition `xml:"on-error"`
	Retry    *Retry             `xml:"retry"`
	Output   *StepOutput        `xml:"output"`
}

// OnErrorTransition defines an error handler transition
type OnErrorTransition struct {
	State string `xml:"state,attr"`
}

// Retry defines retry logic
type Retry struct {
	Count   int    `xml:"count,attr"`
	Backoff string `xml:"backoff,attr"`
}

// StepOutput defines step output mapping
type StepOutput struct {
	Assign string       `xml:"assign,attr"`
	Paths  []OutputPath `xml:",any"`
}

// OutputPath maps a JSONPath to a field
type OutputPath struct {
	XMLName xml.Name
	Path    string `xml:"path,attr"`
	Assign  string `xml:"assign,attr"`
}

// StepParam defines a parameter for func steps
type StepParam struct {
	XMLName xml.Name
	Name    string `xml:"name,attr"`
	Value   string `xml:"value,attr"`
}

// Call invokes a sub-flow
type Call struct {
	Ref     string             `xml:"ref,attr"`
	When    string             `xml:"when,attr"`
	Timeout string             `xml:"timeout,attr"`
	OnError *OnErrorTransition `xml:"on-error"`
	Input   *CallInputBlock    `xml:"input"`
	Output  *CallOutputBlock   `xml:"output"`
}

// CallInputBlock defines typed input fields for a call
type CallInputBlock struct {
	Strings []CallTypedField `xml:"string"`
	Ints    []CallTypedField `xml:"int"`
	Bools   []CallTypedField `xml:"bool"`
	Floats  []CallTypedField `xml:"float"`
}

// GetFields returns all input fields as a slice of CallInputField
func (c *CallInputBlock) GetFields() []CallInputField {
	var fields []CallInputField
	for _, f := range c.Strings {
		fields = append(fields, CallInputField{String: &f})
	}
	for _, f := range c.Ints {
		fields = append(fields, CallInputField{Int: &f})
	}
	for _, f := range c.Bools {
		fields = append(fields, CallInputField{Bool: &f})
	}
	for _, f := range c.Floats {
		fields = append(fields, CallInputField{Float: &f})
	}
	return fields
}

// CallOutputBlock defines typed output fields for a call
type CallOutputBlock struct {
	Strings []CallTypedField `xml:"string"`
	Ints    []CallTypedField `xml:"int"`
	Bools   []CallTypedField `xml:"bool"`
	Floats  []CallTypedField `xml:"float"`
}

// GetFields returns all output fields as a slice of CallOutputField
func (c *CallOutputBlock) GetFields() []CallOutputField {
	var fields []CallOutputField
	for _, f := range c.Strings {
		fields = append(fields, CallOutputField{String: &f})
	}
	for _, f := range c.Ints {
		fields = append(fields, CallOutputField{Int: &f})
	}
	for _, f := range c.Bools {
		fields = append(fields, CallOutputField{Bool: &f})
	}
	for _, f := range c.Floats {
		fields = append(fields, CallOutputField{Float: &f})
	}
	return fields
}

// CallInputField represents a typed input field in a call (internal use)
type CallInputField struct {
	XMLName xml.Name        `xml:"-"`
	String  *CallTypedField `xml:"string"`
	Int     *CallTypedField `xml:"int"`
	Bool    *CallTypedField `xml:"bool"`
	Float   *CallTypedField `xml:"float"`
}

// GetTypedField returns the non-nil typed field
func (c *CallInputField) GetTypedField() *CallTypedField {
	if c.String != nil {
		return c.String
	}
	if c.Int != nil {
		return c.Int
	}
	if c.Bool != nil {
		return c.Bool
	}
	if c.Float != nil {
		return c.Float
	}
	return nil
}

// GetType returns the type name of this field
func (c *CallInputField) GetType() string {
	if c.String != nil {
		return string(TypeString)
	}
	if c.Int != nil {
		return string(TypeInt)
	}
	if c.Bool != nil {
		return string(TypeBool)
	}
	if c.Float != nil {
		return string(TypeFloat)
	}
	return ""
}

// CallOutputField represents a typed output field in a call
type CallOutputField struct {
	XMLName xml.Name        `xml:"-"`
	String  *CallTypedField `xml:"string"`
	Int     *CallTypedField `xml:"int"`
	Bool    *CallTypedField `xml:"bool"`
	Float   *CallTypedField `xml:"float"`
}

// GetTypedField returns the non-nil typed field
func (c *CallOutputField) GetTypedField() *CallTypedField {
	if c.String != nil {
		return c.String
	}
	if c.Int != nil {
		return c.Int
	}
	if c.Bool != nil {
		return c.Bool
	}
	if c.Float != nil {
		return c.Float
	}
	return nil
}

// GetType returns the type name of this field
func (c *CallOutputField) GetType() string {
	if c.String != nil {
		return string(TypeString)
	}
	if c.Int != nil {
		return string(TypeInt)
	}
	if c.Bool != nil {
		return string(TypeBool)
	}
	if c.Float != nil {
		return string(TypeFloat)
	}
	return ""
}

// CallTypedField represents a typed field reference in a call
type CallTypedField struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Transition defines state transition
type Transition struct {
	To        string `xml:"to,attr"`
	When      string `xml:"when,attr"`
	Otherwise bool   `xml:"otherwise,attr"`
}
