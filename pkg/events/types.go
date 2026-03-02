// Package events provides a centralized event bus for decoupled communication
// between services in the Gollum application.
package events

// Event type constants - centrally defined for type safety
const (
	// Workspace events
	EventDirectoryChanged = "workspace.directory_changed"

	// Skill events
	EventSkillsUpdated    = "skills.updated"
	EventSkillsDiscovered = "skills.discovered"

	// Agent events
	EventAgentSpawned  = "agent.spawned"
	EventAgentRemoved  = "agent.removed"
	EventAgentIdle     = "agent.idle"
	EventAgentBusy     = "agent.busy"
	EventAgentResumed  = "agent.resumed"
	EventAgentPaused   = "agent.paused"

	// Configuration events
	EventConfigChanged = "config.changed"

	// Plugin events
	EventPluginLoaded   = "plugin.loaded"
	EventPluginUnloaded = "plugin.unloaded"
)

// DirectoryChangedPayload is sent when the working directory changes.
type DirectoryChangedPayload struct {
	OldPath string
	NewPath string
}

// SkillsUpdatedPayload is sent when skills are reloaded.
type SkillsUpdatedPayload struct {
	SkillPaths []string
}

// SkillsDiscoveredPayload is sent when new skills are discovered.
type SkillsDiscoveredPayload struct {
	SkillPaths []string
}

// AgentSpawnedPayload is sent when a new agent is created.
type AgentSpawnedPayload struct {
	AgentID   string
	ParentID  string
	AgentType string
}

// AgentRemovedPayload is sent when an agent is removed.
type AgentRemovedPayload struct {
	AgentID  string
	ParentID string
}

// AgentStatePayload is sent when an agent's state changes.
type AgentStatePayload struct {
	AgentID string
	State   string
}

// ConfigChangedPayload is sent when configuration changes.
type ConfigChangedPayload struct {
	Key   string
	Value any
}

// PluginLoadedPayload is sent when a plugin is loaded.
type PluginLoadedPayload struct {
	PluginName string
	PluginPath string
}

// PluginUnloadedPayload is sent when a plugin is unloaded.
type PluginUnloadedPayload struct {
	PluginName string
}
