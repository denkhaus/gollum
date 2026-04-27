package flows

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/strategy"
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

type FlowVariableScope string

func (p FlowVariableScope) String() string {
	return string(p)
}

// Validate checks if the scope is a valid FlowVariableScope
func (p FlowVariableScope) Validate() error {
	valid := map[FlowVariableScope]bool{
		FlowVariableScopeInput:    true,
		FlowVariableScopeContext:  true,
		FlowVariableScopeOutput:   true,
		FlowVariableScopeComputed: true,
		FlowVariableScopeSys:      true,
		FlowVariableScopeEnv:      true,
	}

	if !valid[p] {
		return fmt.Errorf("invalid flow variable scope: %s (expected one of: input, context, output, computed, sys, env)", p)
	}
	return nil
}

// ValueType represents the type of a field
type ValueType string

func (p ValueType) IsEmpty() bool {
	return p == ""
}

const (
	FlowVariableScopeInput    FlowVariableScope = "input"
	FlowVariableScopeContext  FlowVariableScope = "context"
	FlowVariableScopeOutput   FlowVariableScope = "output"
	FlowVariableScopeComputed FlowVariableScope = "computed"
	FlowVariableScopeSys      FlowVariableScope = "sys"
	FlowVariableScopeEnv      FlowVariableScope = "env"

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
	fields := make([]FieldDef, 0, len(i.Strings)+len(i.Ints)+len(i.Bools)+len(i.Floats)+len(i.Arrays)+len(i.Maps)+len(i.Objects))
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
	fields := make([]FieldDef, 0, len(o.Strings)+len(o.Ints)+len(o.Bools)+len(o.Floats)+len(o.Objects))
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
		if f.AssignFrom != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// GetImperative returns all output fields without a 'from' attribute
func (o *OutputBlock) GetImperative() []FieldDef {
	var fields []FieldDef
	for _, f := range o.GetAllFields() {
		if f.AssignFrom == "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// HasField checks if a field with the given name exists
func (o *OutputBlock) HasField(name string) bool {
	for _, f := range o.GetAllFields() {
		if f.Name == name {
			return true
		}
	}
	return false
}

// GetField returns the field with the given name, or nil if not found
func (o *OutputBlock) GetField(name string) *FieldDef {
	for _, f := range o.GetAllFields() {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

// HasField checks if a field with the given name exists in the input block
func (i *InputBlock) HasField(name string) bool {
	for _, f := range i.GetAllFields() {
		if f.Name == name {
			return true
		}
	}
	return false
}

// HasField checks if a field with the given name exists in the context block
func (c *ContextBlock) HasField(name string) bool {
	for _, f := range c.GetAllFields() {
		if f.Name == name {
			return true
		}
	}
	return false
}

// HasField checks if a field with the given name exists in the computed block
func (c *ComputedBlock) HasField(name string) bool {
	for _, f := range c.GetAllFields() {
		if f.Name == name {
			return true
		}
	}
	return false
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
	fields := make([]ContextField, 0, len(c.Strings)+len(c.Ints)+len(c.Bools)+len(c.Floats))
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
	XMLName     xml.Name
	Name        string     `xml:"name,attr"`
	Type        ValueType  `xml:"type,attr"`
	Default     string     `xml:"default,attr"`
	Description string     `xml:"description,attr,omitempty"`
	Fields      []FieldDef `xml:",any"`
}

// FieldDef is a base type for field definitions
type FieldDef struct {
	XMLName     xml.Name
	Name        string    `xml:"name,attr"`
	Type        ValueType `xml:"type,attr"`
	Required    bool      `xml:"required,attr"`
	Default     string    `xml:"default,attr"`
	AssignFrom  string    `xml:"assignFrom,attr,omitempty"` // Source reference for output bindings
	Description string    `xml:"description,attr,omitempty"`
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
	XMLName     xml.Name
	Name        string    `xml:"name,attr"`
	Type        ValueType // Set programmatically, not from XML (element name defines type)
	Default     string    `xml:"default,attr"`
	Description string    `xml:"description,attr,omitempty"`
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

// ComputedBlock defines computed fields at the flow level
type ComputedBlock struct {
	Strings []ComputedFieldDef `xml:"string"`
	Ints    []ComputedFieldDef `xml:"int"`
	Bools   []ComputedFieldDef `xml:"bool"`
	Floats  []ComputedFieldDef `xml:"float"`
}

// GetAllFields returns all computed field definitions as a slice
func (c *ComputedBlock) GetAllFields() []ComputedFieldDef {
	fields := make([]ComputedFieldDef, 0, len(c.Strings)+len(c.Ints)+len(c.Bools)+len(c.Floats))
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
	Name        string    `xml:"name,attr"`
	Type        ValueType // Set programmatically, not from XML (element name defines type)
	Eval        string    `xml:"eval,attr"` // Expression to evaluate
	Description string    `xml:"description,attr,omitempty"`
}

// AgentStrategy defines strategy parameters for flow agents
type AgentStrategy struct {
	MaxIterations      int                   `xml:"maxIterations,attr"`
	MaxRepeatedActions int                   `xml:"maxRepeatedActions,attr"`
	Type               strategy.StrategyType `xml:"type,attr,omitempty"` // "react" or "simple", defaults to "react"
}

// Agent defines an LLM agent
type Agent struct {
	Name        string         `xml:"name,attr"`
	Model       string         `xml:"model,attr"`
	Prompt      string         `xml:"prompt"`
	Temperature float64        `xml:"temperature"`
	TopP        float64        `xml:"top_p"`
	MaxTokens   int            `xml:"max_tokens"`
	Strategy    *AgentStrategy `xml:"strategy"`
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
	XMLName xml.Name
	// TODO: make Type a proper StepType type, not a string
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
	Result   *StepResult        `xml:"result"`
	Verbose  bool               `xml:"verbose,attr"` // Show LLM output in logs
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

// StepResult defines step result mapping
type StepResult struct {
	AssignTo string       `xml:"assignTo,attr"`
	Paths    []ResultPath `xml:",any"`
}

// ResultPath maps a JSONPath to a field
type ResultPath struct {
	XMLName  xml.Name
	Path     string `xml:"path,attr"`
	AssignTo string `xml:"assignTo,attr"`
}

// StepParam defines a parameter for func steps
type StepParam struct {
	XMLName    xml.Name
	Name       string `xml:"name,attr"`
	AssignFrom string `xml:"assignFrom,attr"` // Source reference (data comes FROM this value)
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
	Strings []CallInputParam `xml:"string"`
	Ints    []CallInputParam `xml:"int"`
	Bools   []CallInputParam `xml:"bool"`
	Floats  []CallInputParam `xml:"float"`
}

// GetFields returns all input fields as a slice of CallInputFieldRef
func (c *CallInputBlock) GetFields() []CallInputFieldRef {
	fields := make([]CallInputFieldRef, 0, len(c.Strings)+len(c.Ints)+len(c.Bools)+len(c.Floats))
	for _, f := range c.Strings {
		fields = append(fields, CallInputFieldRef{String: &f})
	}
	for _, f := range c.Ints {
		fields = append(fields, CallInputFieldRef{Int: &f})
	}
	for _, f := range c.Bools {
		fields = append(fields, CallInputFieldRef{Bool: &f})
	}
	for _, f := range c.Floats {
		fields = append(fields, CallInputFieldRef{Float: &f})
	}
	return fields
}

// CallOutputBlock defines typed output fields for a call
type CallOutputBlock struct {
	Strings []CallOutputParam `xml:"string"`
	Ints    []CallOutputParam `xml:"int"`
	Bools   []CallOutputParam `xml:"bool"`
	Floats  []CallOutputParam `xml:"float"`
}

// GetFields returns all output fields as a slice of CallOutputFieldRef
func (c *CallOutputBlock) GetFields() []CallOutputFieldRef {
	fields := make([]CallOutputFieldRef, 0, len(c.Strings)+len(c.Ints)+len(c.Bools)+len(c.Floats))
	for _, f := range c.Strings {
		fields = append(fields, CallOutputFieldRef{String: &f})
	}
	for _, f := range c.Ints {
		fields = append(fields, CallOutputFieldRef{Int: &f})
	}
	for _, f := range c.Bools {
		fields = append(fields, CallOutputFieldRef{Bool: &f})
	}
	for _, f := range c.Floats {
		fields = append(fields, CallOutputFieldRef{Float: &f})
	}
	return fields
}

// CallInputParam represents a parameter for call input (data from parent context)
type CallInputParam struct {
	Name        string `xml:"name,attr"`
	AssignFrom  string `xml:"assignFrom,attr"` // Source reference in parent context
	Description string `xml:"description,attr,omitempty"`
}

// CallOutputParam represents a parameter for call output (data to parent context)
type CallOutputParam struct {
	Name        string `xml:"name,attr"`
	AssignTo    string `xml:"assignTo,attr"` // Target in parent context
	Description string `xml:"description,attr,omitempty"`
}

// CallInputFieldRef is a union wrapper for input field variants (internal use)
type CallInputFieldRef struct {
	XMLName xml.Name        `xml:"-"`
	String  *CallInputParam `xml:"string"`
	Int     *CallInputParam `xml:"int"`
	Bool    *CallInputParam `xml:"bool"`
	Float   *CallInputParam `xml:"float"`
}

// GetParam returns the non-nil parameter
func (c *CallInputFieldRef) GetParam() *CallInputParam {
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
func (c *CallInputFieldRef) GetType() string {
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

// CallOutputFieldRef is a union wrapper for output field variants (internal use)
type CallOutputFieldRef struct {
	XMLName xml.Name         `xml:"-"`
	String  *CallOutputParam `xml:"string"`
	Int     *CallOutputParam `xml:"int"`
	Bool    *CallOutputParam `xml:"bool"`
	Float   *CallOutputParam `xml:"float"`
}

// GetParam returns the non-nil parameter
func (c *CallOutputFieldRef) GetParam() *CallOutputParam {
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
func (c *CallOutputFieldRef) GetType() string {
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

// Transition defines state transition
type Transition struct {
	To        string `xml:"to,attr"`
	When      string `xml:"when,attr"`
	Otherwise bool   `xml:"otherwise,attr"`
}

// LintRunner is an interface for linting flows (implemented by linter package to avoid circular imports)
type LintRunner interface {
	LintFlow(flow *Flow, flowPath, xmlContent string) *LinterResult
}

// lintRunner is set by the linter package to provide the actual implementation
var lintRunner LintRunner

// RegisterLintRunner registers the linter implementation (called by linter package init)
func RegisterLintRunner(runner LintRunner) {
	lintRunner = runner
}

// FlowExecutionResult holds the result of a flow execution
type FlowExecutionResult struct {
	Outputs map[string]any
}

// Executor is an interface for executing flows.
// Implementations should be provided via dependency injection.
type Executor interface {
	// Execute executes a flow with the given inputs and returns the result
	Execute(ctx context.Context, flow *Flow, inputs map[string]any) (*FlowExecutionResult, error)
}

// Lint runs all linter phases on a flow
func (f *Flow) Lint() *LinterResult {
	return f.LintPath("")
}

// LintPath runs all linter phases on a flow with a known file path
func (f *Flow) LintPath(flowPath string) *LinterResult {
	if lintRunner == nil {
		return &LinterResult{
			Errors: []LinterError{
				{Message: "linter not initialized - import linter package to enable linting"},
			},
		}
	}
	return lintRunner.LintFlow(f, flowPath, "")
}
