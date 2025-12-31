// Package middleware provides display and UI-related middleware for agent interactions.
package middleware

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/ui"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// DisplayMiddleware provides visual output for agent communication
	DisplayMiddleware struct {
		messenger ui.AgentMessenger
		registry  registry.AgentRegistry
		agentID   uuid.UUID
		agentRole string
	}

	// DisplayMiddlewareProvider creates display middleware instances via DI
	DisplayMiddlewareProvider interface {
		CreateDisplayMiddleware(agentID uuid.UUID, agentRole string) *DisplayMiddleware
	}

	displayMiddlewareProvider struct {
		messenger ui.AgentMessenger
		registry  registry.AgentRegistry
	}
)

// NewDisplayMiddlewareProvider creates a provider for display middleware
func NewDisplayMiddlewareProvider(injector do.Injector) (DisplayMiddlewareProvider, error) {
	messenger := do.MustInvoke[ui.AgentMessenger](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &displayMiddlewareProvider{
		messenger: messenger,
		registry:  registry,
	}, nil
}

// CreateDisplayMiddleware creates a new display middleware for a specific agent
func (p *displayMiddlewareProvider) CreateDisplayMiddleware(
	agentID uuid.UUID,
	agentRole string,
) *DisplayMiddleware {
	return &DisplayMiddleware{
		messenger: p.messenger,
		registry:  p.registry,
		agentID:   agentID,
		agentRole: agentRole,
	}
}

// ContentBlockMiddleware displays text content blocks
func (d *DisplayMiddleware) ContentBlockMiddleware(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
	return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		// Display text responses
		if resp != nil && len(resp.Texts) > 0 {
			for _, text := range resp.Texts {
				if text != "" {
					d.messenger.DisplayAgentMessage(d.agentID, d.agentRole, text, false)
				}
			}
		}

		return resp, nil
	}
}

// ToolMiddleware displays tool execution information
func (d *DisplayMiddleware) ToolMiddleware(next gollem.ToolHandler) gollem.ToolHandler {
	return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		// Show tool usage before execution
		toolInfo := fmt.Sprintf("Using tool: %s", req.Tool.Name)

		// Add arguments if available
		if req.Tool != nil && req.Tool.Arguments != nil {
			toolInfo += fmt.Sprintf("\nArguments: %v", req.Tool.Arguments)
		}

		d.messenger.DisplayAgentMessage(d.agentID, d.agentRole, toolInfo, true)

		// Execute the tool
		resp, err := next(ctx, req)

		// Display tool result if available
		if err == nil && resp != nil && resp.Result != nil {
			resultInfo := fmt.Sprintf("Tool result: %v", resp.Result)
			d.messenger.DisplayAgentMessage(d.agentID, d.agentRole, resultInfo, true)
		}

		return resp, err
	}
}

// DisplayWelcome shows the welcome message
func (d *DisplayMiddleware) DisplayWelcome() {
	d.messenger.DisplayWelcome()
}

// DisplayUserMessage shows a user's input message
func (d *DisplayMiddleware) DisplayUserMessage(message string) {
	d.messenger.DisplayUserMessage(message)
}

// DisplaySystemInfo shows system information
func (d *DisplayMiddleware) DisplaySystemInfo(message string) {
	d.messenger.DisplaySystemInfo(message)
}
