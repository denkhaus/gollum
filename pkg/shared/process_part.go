package shared

// PartType represents the type of content in a ProcessPart
type PartType int

const (
	// PartTypeText is a text content part
	PartTypeText PartType = iota
	// PartTypeToolUse is a tool use content part
	PartTypeToolUse
	// PartTypeToolResult is a tool result content part
	PartTypeToolResult
	// PartTypeThinking is a thinking/reasoning content part
	PartTypeThinking
	// PartTypeError is an error content part
	PartTypeError
)

// String returns a string representation of the PartType
func (pt PartType) String() string {
	switch pt {
	case PartTypeText:
		return "text"
	case PartTypeToolUse:
		return "tool_use"
	case PartTypeToolResult:
		return "tool_result"
	case PartTypeThinking:
		return "thinking"
	case PartTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// ProcessPart represents a part of the agent's output processing
// This is used by middleware to handle different types of agent outputs
type ProcessPart struct {
	Type PartType

	// Text content (for PartTypeText and PartTypeThinking)
	Text *string

	// Tool information (for PartTypeToolUse and PartTypeToolResult)
	ToolName string
	ToolID   string

	// Error message (for PartTypeError)
	ErrorMessage string
}
