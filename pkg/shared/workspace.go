package shared

// SkillInfo holds information about a discovered skill.
// Used by workspace, skills, and prompt packages for consistent skill representation.
type SkillInfo struct {
	Name        string `json:"name"`        // Skill name
	Description string `json:"description"` // Brief description
	Location    string `json:"location"`    // File path to the skill
}

// WorkspaceContext holds workspace-specific information for prompt rendering.
// It provides the current working directory and available skills to the LLM.
type WorkspaceContext struct {
	CurrentPath string      `json:"current_path"` // Current working directory
	SkillsXML   string      `json:"skills_xml"`   // Skills in XML format for LLM prompts
	Skills      []SkillInfo `json:"skills"`       // List of discovered skills
}
