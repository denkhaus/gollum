package shared

// ToolName is a strongly-typed tool identifier
type ToolName string

func (p ToolName) String() string {
	return string(p)
}

// Tool name constants for type safety and to avoid magic strings
const (
	ToolNameSpawnAgent      ToolName = "spawn_agent"
	ToolNameResumeAgent     ToolName = "resume_agent"
	ToolNameAgentOutput     ToolName = "agent_output"
	ToolNameRemoveAgent     ToolName = "remove_agent"
	ToolNameListAgents      ToolName = "list_agents"
	ToolNameCurrentTime     ToolName = "current_time"
	ToolNameBash            ToolName = "bash"
	ToolNameWriteFile       ToolName = "write_file"
	ToolNameReadFile        ToolName = "read_file"
	ToolNameSessionLogs     ToolName = "session_logs"
	ToolNameChangeDirectory ToolName = "change_directory"
	ToolNameInvokeSkill     ToolName = "invoke_skill"
	ToolNameEdit            ToolName = "edit"
	ToolNameGlob            ToolName = "glob"
	ToolNameGrep            ToolName = "grep"
	// Flow executor tools - for use within flow executions
	ToolNameSetOutputField  ToolName = "set_output_field"
	ToolNameSetContextField ToolName = "set_context_field"
	ToolNameGetContext      ToolName = "get_context"
	ToolNameEmitLog         ToolName = "emit_log"
	ToolNameTransitionTo    ToolName = "transition_to"
	// Flow management tools
	ToolNameExecuteFlow ToolName = "execute_flow"
	ToolNameListFlows   ToolName = "list_flows"
)

// SupervisorBuiltinTools contains the built-in tools available to the supervisor agent.
// Flow executor tools (set_output_field, set_context_field, etc.) are excluded as they
// are only available within flow executions.
var SupervisorBuiltinTools = []ToolName{
	ToolNameSpawnAgent,
	ToolNameResumeAgent,
	ToolNameAgentOutput,
	ToolNameRemoveAgent,
	ToolNameListAgents,
	ToolNameCurrentTime,
	ToolNameBash,
	ToolNameWriteFile,
	ToolNameReadFile,
	ToolNameSessionLogs,
	ToolNameChangeDirectory,
	ToolNameInvokeSkill,
	ToolNameEdit,
	ToolNameGlob,
	ToolNameGrep,
}

// SubAgentBuiltinTools contains the built-in tools available to subagents by default.
// Excludes agent management tools (spawn_agent, resume_agent, agent_output, remove_agent, list_agents)
// to keep sub-agents focused on task execution. For orchestrator subagents that need to spawn
// additional subagents, these tools can be explicitly added via allowed_tools parameter.
var SubAgentBuiltinTools = []ToolName{
	ToolNameCurrentTime,
	ToolNameBash,
	ToolNameWriteFile,
	ToolNameReadFile,
	ToolNameSessionLogs,
	ToolNameChangeDirectory,
	ToolNameInvokeSkill,
	ToolNameEdit,
	ToolNameGlob,
	ToolNameGrep,
}

// ToolRegistry provides validation for tool names used in skills.
// It maintains a list of all valid Gollum tool names and allows
// the skills package to validate tool configurations
type ToolRegistry interface {
	// IsValidTool checks if a tool name is a valid registered tool
	IsValidTool(name ToolName) bool

	// GetValidToolNames returns all valid tool names
	GetValidToolNames() []ToolName
}
