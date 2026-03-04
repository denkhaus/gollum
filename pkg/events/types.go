// Package events provides a centralized event bus for decoupled communication
// between services in the Gollum application.
package events

import "github.com/denkhaus/gollum/pkg/shared"

type EventType string

func (p EventType) String() string {
	return string(p)
}

// Event type constants - centrally defined for type safety
const (
	// Workspace events
	EventDirectoryChanged EventType = "workspace.directory_changed"

	// Skill events
	EventSkillsUpdated    EventType = "skills.updated"
	EventSkillsDiscovered EventType = "skills.discovered"

	// Agent events
	EventAgentSpawned EventType = "agent.spawned"
	EventAgentRemoved EventType = "agent.removed"
	EventAgentIdle    EventType = "agent.idle"
	EventAgentBusy    EventType = "agent.busy"
	EventAgentResumed EventType = "agent.resumed"
	EventAgentPaused  EventType = "agent.paused"

	// Configuration events
	EventConfigChanged EventType = "config.changed"

	// Plugin events
	EventPluginLoaded   EventType = "plugin.loaded"
	EventPluginUnloaded EventType = "plugin.unloaded"
)

// DirectoryChangedPayload is sent when the working directory changes.
type DirectoryChangedPayload struct {
	OldPath string
	NewPath string
}

// SkillsUpdatedPayload is sent when skills are reloaded.
type SkillsUpdatedPayload struct {
	Skills    []shared.SkillInfo
	SkillsXML string // Skills in XML format for LLM prompts
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
