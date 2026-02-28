package shared

// Tool name constants for type safety and to avoid magic strings
const (
	ToolNameSpawnAgent      = "spawn_agent"
	ToolNameResumeAgent     = "resume_agent"
	ToolNameAgentOutput     = "agent_output"
	ToolNameRemoveAgent     = "remove_agent"
	ToolNameListAgents      = "list_agents"
	ToolNameCurrentTime     = "current_time"
	ToolNameBash            = "bash"
	ToolNameWriteFile       = "write_file"
	ToolNameReadFile        = "read_file"
	ToolNameSessionLogs     = "session_logs"
	ToolNameChangeDirectory = "change_directory"
	ToolNameInvokeSkill     = "invoke_skill"

	// File operation tools (matching builtin tools)
	ToolNameEdit = "edit"
	ToolNameGlob = "glob"
	ToolNameGrep = "grep"
)
