package shared

type ToolName string

// Tool name constants for type safety and to avoid magic strings
const (
	ToolNameSpawnAgent  ToolName = "spawn_agent"
	ToolNameResumeAgent ToolName = "resume_agent"
	ToolNameAgentOutput ToolName = "agent_output"
	ToolNameRemoveAgent ToolName = "remove_agent"
	ToolNameListAgents  ToolName = "list_agents"
	ToolNameCurrentTime ToolName = "current_time"
	ToolNameBash        ToolName = "bash"
	ToolNameWriteFile   ToolName = "write_file"
	ToolNameReadFile    ToolName = "read_file"
	ToolNameSessionLogs ToolName = "session_logs"

	// File operation tools (matching builtin tools)
	ToolNameEdit = "edit"
	ToolNameGlob = "glob"
	ToolNameGrep = "grep"
)
