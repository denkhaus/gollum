package flows

import "encoding/xml"

// Flow represents a complete workflow definition
type Flow struct {
	XMLName     xml.Name      `xml:"flow"`
	Name        string        `xml:"name,attr"`
	Version     string        `xml:"version,attr"`
	Description string        `xml:"description"`
	Input       *InputBlock   `xml:"input"`
	Output      *OutputBlock  `xml:"output"`
	Context     *ContextBlock `xml:"context"`
	Agents      []Agent       `xml:"agents>agent"`
	States      []State       `xml:"states>state"`
}

// InputBlock defines the flow's input interface
type InputBlock struct {
	Fields []FieldDef `xml:",any"`
}

// OutputBlock defines the flow's output interface
type OutputBlock struct {
	Fields []FieldDef `xml:",any"`
}

// ContextBlock defines internal context fields
type ContextBlock struct {
	Fields       []ContextField `xml:",any"`
	Computeds    []ComputedField `xml:"computed"`
}

// FieldDef is a base type for field definitions
type FieldDef struct {
	XMLName  xml.Name
	Name     string `xml:"name,attr"`
	Type     string `xml:"type,attr"`
	Required bool   `xml:"required,attr"`
	Default  string `xml:"default,attr"`
}

// ContextField represents a regular context field
type ContextField struct {
	XMLName xml.Name
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr"`
	Default string `xml:"default,attr"`
}

// ComputedField represents a computed context field
type ComputedField struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	When  string `xml:"when,attr"`  // Expression
}

// Agent defines an LLM agent
type Agent struct {
	Name        string `xml:"name,attr"`
	Model       string `xml:"model,attr"`
	Prompt      string `xml:"prompt"`
	Temperature string `xml:"temperature"`
	MaxTokens   int    `xml:"max_tokens"`
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
	XMLName   xml.Name
	Type      string  `xml:"type,attr"`
	Name      string  `xml:"name,attr"`
	Agent     string  `xml:"agent,attr"`
	Function  string  `xml:"function,attr"`
	Tool      string  `xml:"tool,attr"`
	Prompt    string  `xml:"prompt"`
	Cmd       string  `xml:"cmd"`
	Tools     string  `xml:"tools"`
	Timeout   string  `xml:"timeout"`
	OnError   string  `xml:"on-error,attr"`
	Retry     *Retry  `xml:"retry"`
	Output    *StepOutput `xml:"output"`
}

// Retry defines retry logic
type Retry struct {
	Count   int    `xml:"count,attr"`
	Backoff string `xml:"backoff,attr"`
}

// StepOutput defines step output mapping
type StepOutput struct {
	Assign string     `xml:"assign,attr"`
	Paths  []OutputPath `xml:",any"`
}

// OutputPath maps a JSONPath to a field
type OutputPath struct {
	XMLName xml.Name
	Path    string `xml:"path,attr"`
	Assign  string `xml:"assign,attr"`
}

// Call invokes a sub-flow
type Call struct {
	Ref      string       `xml:"ref,attr"`
	When     string       `xml:"when,attr"`
	Timeout  string       `xml:"timeout,attr"`
	OnError  string       `xml:"on-error,attr"`
	Input    []CallField  `xml:"input>field"`
	Output   []CallField  `xml:"output>field"`
}

// CallField maps fields for call input/output
type CallField struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Transition defines state transition
type Transition struct {
	To        string `xml:"to,attr"`
	When      string `xml:"when,attr"`
	Otherwise bool   `xml:"otherwise,attr"`
}
