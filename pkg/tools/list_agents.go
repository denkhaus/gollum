package tools

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// ListAgentsTool lists all subagents of the current agent with their IDs and roles
	ListAgentsTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
		senderID    uuid.UUID
	}

	// ListAgentsToolProvider creates ListAgentsTool instances via DI
	ListAgentsToolProvider interface {
		CreateTool(senderID uuid.UUID) *ListAgentsTool
	}

	listAgentsToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		registry    registry.AgentRegistry
	}
)

// NewListAgentsToolProvider creates a provider for ListAgents tools
func NewListAgentsToolProvider(injector do.Injector) (ListAgentsToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &listAgentsToolProvider{
		logService:  logService,
		hookManager: hookManager,
		registry:    registry,
	}, nil
}

// CreateListAgentsTool creates a new ListAgentsTool for a specific sender
func (p *listAgentsToolProvider) CreateTool(senderID uuid.UUID) *ListAgentsTool {
	return &ListAgentsTool{
		logService:  p.logService,
		hookManager: p.hookManager,
		registry:    p.registry,
		senderID:    senderID,
	}
}

// Spec returns the tool specification for the ListAgents tool
func (t *ListAgentsTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name: shared.ToolNameListAgents.String(),
		Description: `Shows your agent relationships with IDs, roles, descriptions, and types.

VISIBILITY (what you can see):
- By default: Shows your parent (if exists) and your DIRECT children only
- With --recursive flag: Shows your entire descendant subtree (children, grandchildren, etc.)

OPERATIONS (what you can do):
- CRITICAL: All tools work on DIRECT children ONLY
- ` + shared.ToolNameRemoveAgent.String() + `: Remove your direct children only
- ` + shared.ToolNameResumeAgent.String() + `: Resume your direct children only
- ` + shared.ToolNameAgentOutput.String() + `: Get output from your direct children only
- To operate on grandchildren: Must delegate through your direct child

SCOPE:
- You CANNOT see or operate on sibling agents
- You CANNOT see or operate on unrelated branches
- Each agent is responsible only for their own direct children

See also: ` + shared.ToolNameSpawnAgent.String() + `, ` + shared.ToolNameRemoveAgent.String() + `, ` + shared.ToolNameResumeAgent.String() + `, ` + shared.ToolNameAgentOutput.String(),
		Parameters: map[string]*gollem.Parameter{
			"recursive": {
				Type:        gollem.TypeBoolean,
				Description: "Show entire descendant subtree (children, grandchildren, etc.)",
			},
			"tree": {
				Type:        gollem.TypeBoolean,
				Description: "Format output as ASCII tree with indentation by depth",
			},
		},
	}
}

// descendantInfo holds information about a descendant agent with depth
type descendantInfo struct {
	agent     shared.Agent
	config    *shared.AgentConfig
	depth     int
	displayID string
	isLast    bool // for tree visualization
}

// Run executes the ListAgents tool to list all related agents
func (t *ListAgentsTool) Run(ctx context.Context, params map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.senderID, shared.ToolNameListAgents, params,
		func() (map[string]any, error) {
			return t.runListAgents(ctx, params)
		})
}

// runListAgents implements the core ListAgents logic
func (t *ListAgentsTool) runListAgents(_ context.Context, params map[string]any) (map[string]any, error) {
	// Parse flags
	recursive := false
	tree := false
	if val, ok := params["recursive"].(bool); ok {
		recursive = val
	}
	if val, ok := params["tree"].(bool); ok {
		tree = val
	}

	t.logService.Infof("Agent %s listing related agents (recursive=%v, tree=%v)", t.senderID, recursive, tree)

	// Try to get the parent agent
	parentAgent, hasParent := t.registry.GetParent(t.senderID)

	if tree {
		// Tree mode: hierarchical visualization
		var descendantCount int
		treeOutput := t.buildTreeOutput(recursive, hasParent, parentAgent, &descendantCount)

		return map[string]any{
			"success":          true,
			"message":          fmt.Sprintf("Agent tree (showing %d descendant(s))", descendantCount),
			"agents":           nil, // Tree output is formatted as string
			"tree":             treeOutput,
			"descendant_count": descendantCount,
		}, nil
	}

	// List mode: flat list
	var agentList []map[string]any

	// Add parent agent if it exists
	if hasParent {
		parentConfig := parentAgent.GetConfig()
		parentInfo := map[string]any{
			"id":          parentConfig.ID.String(),
			"role":        parentConfig.Role,
			"description": parentConfig.Description,
			"type":        "ParentAgent",
		}
		agentList = append(agentList, parentInfo)
		t.logService.Debugf("Found parent agent: %s (role: %s, description: %s)", parentConfig.ID, parentConfig.Role, parentConfig.Description)
	}

	var descendantCount int

	if recursive {
		// Get all descendants recursively
		descendants := t.getAllDescendants(t.senderID, 0)
		descendantCount = len(descendants)

		for _, desc := range descendants {
			agentInfo := map[string]any{
				"id":          desc.config.ID.String(),
				"role":        desc.config.Role,
				"description": desc.config.Description,
				"type":        "SubAgent",
				"depth":       desc.depth,
			}
			agentList = append(agentList, agentInfo)
			t.logService.Debugf("Found descendant: %s (depth: %d, role: %s)", desc.config.ID, desc.depth, desc.config.Role)
		}
	} else {
		// Default: direct children only
		children := t.registry.GetChildren(t.senderID)
		descendantCount = len(children)

		for _, child := range children {
			config := child.GetConfig()
			agentInfo := map[string]any{
				"id":          config.ID.String(),
				"role":        config.Role,
				"description": config.Description,
				"type":        "SubAgent",
			}
			agentList = append(agentList, agentInfo)
			t.logService.Debugf("Found subagent: %s (role: %s, description: %s)", config.ID, config.Role, config.Description)
		}
	}

	// Prepare message based on what was found
	var message string
	//nolint:gocritic // Nested conditions with recursive flag make switch less readable
	if hasParent && descendantCount > 0 {
		if recursive {
			message = fmt.Sprintf("Found 1 parent agent and %d descendant(s)", descendantCount)
		} else {
			message = fmt.Sprintf("Found 1 parent agent and %d subagent(s)", descendantCount)
		}
		t.logService.Infof("Agent %s found 1 parent and %d descendants", t.senderID, descendantCount)
	} else if hasParent {
		message = "Found 1 parent agent"
		t.logService.Infof("Agent %s found 1 parent (no descendants)", t.senderID)
	} else if descendantCount > 0 {
		if recursive {
			message = fmt.Sprintf("Found %d descendant(s) (no parent agent)", descendantCount)
		} else {
			message = fmt.Sprintf("Found %d subagent(s) (no parent agent)", descendantCount)
		}
		t.logService.Infof("Agent %s found %d descendants (no parent)", t.senderID, descendantCount)
	} else {
		message = "No related agents found (no parent and no subagents)"
		t.logService.Infof("Agent %s has no related agents", t.senderID)
	}

	return map[string]any{
		"success":          true,
		"message":          message,
		"agents":           agentList,
		"descendant_count": descendantCount,
	}, nil
}

// getAllDescendants recursively fetches all descendants of an agent
func (t *ListAgentsTool) getAllDescendants(agentID uuid.UUID, depth int) []descendantInfo {
	children := t.registry.GetChildren(agentID)
	result := make([]descendantInfo, 0, len(children))

	for _, child := range children {
		config := child.GetConfig()
		info := descendantInfo{
			agent:     child,
			config:    config,
			depth:     depth + 1,
			displayID: config.ID.String(),
		}
		result = append(result, info)

		// Recursively get descendants of this child
		childDescendants := t.getAllDescendants(config.ID, depth+1)
		result = append(result, childDescendants...)
	}

	return result
}

// buildTreeOutput creates ASCII tree visualization
func (t *ListAgentsTool) buildTreeOutput(recursive bool, hasParent bool, parentAgent shared.Agent, count *int) string {
	var builder string

	// Start with current agent (you are here)
	builder += fmt.Sprintf("├─ Agent %s (YOU)\n", t.senderID.String()[:8])

	// Add parent if exists (above YOU)
	if hasParent && parentAgent != nil {
		parentConfig := parentAgent.GetConfig()
		builder += fmt.Sprintf("│  └─ Parent: %s (%s)\n", parentConfig.Role, parentConfig.ID.String()[:8])
	}

	if recursive {
		// Recursive tree
		descendants := t.getAllDescendants(t.senderID, 0)
		*count = len(descendants)

		for i, desc := range descendants {
			desc.isLast = (i == len(descendants)-1)
			builder += t.buildTreeLine(desc, "")
		}
	} else {
		// Direct children only
		children := t.registry.GetChildren(t.senderID)
		*count = len(children)

		for i, child := range children {
			config := child.GetConfig()
			isLast := (i == len(children)-1)

			prefix := "│  "
			connector := "├─ "
			if isLast {
				connector = "└─ "
			}

			builder += fmt.Sprintf("%s%s%s (%s)\n", prefix, connector, config.Role, config.ID.String()[:8])
			if config.Description != "" {
				builder += fmt.Sprintf("%s   └─ %s\n", prefix, config.Description)
			}
		}
	}

	return builder
}

// buildTreeLine creates a single line in the tree with proper indentation
func (t *ListAgentsTool) buildTreeLine(desc descendantInfo, prefix string) string {
	var result string

	// Calculate indentation based on depth
	indent := prefix
	for i := 0; i < desc.depth; i++ {
		indent += "   "
	}

	// Choose connector
	connector := "├─ "
	if desc.isLast {
		connector = "└─ "
	}

	result = fmt.Sprintf("%s%s%s (%s)\n", indent, connector, desc.config.Role, desc.displayID[:8])

	// Add description if present
	if desc.config.Description != "" {
		result += fmt.Sprintf("%s   └─ %s\n", indent, desc.config.Description)
	}

	return result
}
