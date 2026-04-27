package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// ChannelMiddlewareProvider creates channel middleware instances via DI
	ChannelMiddlewareProvider interface {
		CreateChannelMiddleware(sessionCtx shared.SessionContext, agentRole string) *ChannelMiddleware
		SetChannelFacade(facade ChannelFacade)
	}

	channelMiddlewareProvider struct {
		injector do.Injector
		facade   ChannelFacade // Lazily set to break circular dependency
	}
)

// ChannelMiddleware sends agent outputs to channel facade.
// This middleware bridges the agent execution pipeline with the channel system.
type ChannelMiddleware struct {
	facade                ChannelFacade
	shared.SessionContext // Embedded session context
	agentRole             string
}

// NewChannelMiddleware creates a new channel middleware
func NewChannelMiddleware(facade ChannelFacade, sessionCtx shared.SessionContext, agentRole string) *ChannelMiddleware {
	return &ChannelMiddleware{
		facade:         facade,
		SessionContext: sessionCtx,
		agentRole:      agentRole,
	}
}

// NewChannelMiddlewareProvider creates a provider for channel middleware
func NewChannelMiddlewareProvider(injector do.Injector) (ChannelMiddlewareProvider, error) {
	return &channelMiddlewareProvider{
		injector: injector,
		facade:   nil, // Will be set later to break circular dependency
	}, nil
}

// SetChannelFacade sets the facade reference (called after ChannelFacade is constructed)
func (p *channelMiddlewareProvider) SetChannelFacade(facade ChannelFacade) {
	p.facade = facade
}

// CreateChannelMiddleware creates a new channel middleware for a specific agent
func (p *channelMiddlewareProvider) CreateChannelMiddleware(
	sessionCtx shared.SessionContext,
	agentRole string,
) *ChannelMiddleware {
	// Use stored facade reference to avoid circular dependency
	return NewChannelMiddleware(p.facade, sessionCtx, agentRole)
}

// ContentBlockMiddleware processes text content blocks and sends to channel
func (p *ChannelMiddleware) ContentBlockMiddleware(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
	return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		// Send text responses to channel
		if resp != nil && len(resp.Texts) > 0 {
			var nonEmptyTexts []string
			for _, text := range resp.Texts {
				if text != "" {
					nonEmptyTexts = append(nonEmptyTexts, text)
				}
			}
			if len(nonEmptyTexts) > 0 {
				combinedText := strings.Join(nonEmptyTexts, "")
				p.facade.DisplayMessage(shared.Message{
					ID:        uuid.New(),
					Role:      gollem.RoleAssistant,
					AgentRole: p.agentRole,
					SessionContext: shared.SessionContext{
						AgentID:   p.AgentID,
						SessionID: p.SessionID,
						ChannelID: p.ChannelID,
					},
					Content:   combinedText,
					Timestamp: time.Now(),
				})
			}
		}

		return resp, nil
	}
}

// ToolMiddleware processes tool execution and sends to channel
func (p *ChannelMiddleware) ToolMiddleware(next gollem.ToolHandler) gollem.ToolHandler {
	return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		// Send tool request to channel
		p.facade.DisplayMessage(shared.Message{
			ID:        uuid.New(),
			Role:      gollem.RoleTool,
			AgentRole: p.agentRole,
			SessionContext: shared.SessionContext{
				AgentID:   p.AgentID,
				SessionID: p.SessionID,
				ChannelID: p.ChannelID,
			},
			Content:   fmt.Sprintf("Tool use: %s", req.Tool.Name),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool_name": req.Tool.Name,
				"tool_id":   req.Tool.ID,
			},
		})

		// Execute the tool
		resp, err := next(ctx, req)

		// Send tool result to channel
		if err == nil && resp != nil && resp.Result != nil {
			p.facade.DisplayMessage(shared.Message{
				ID:        uuid.New(),
				Role:      gollem.RoleTool,
				AgentRole: p.agentRole,
				SessionContext: shared.SessionContext{
					AgentID:   p.AgentID,
					SessionID: p.SessionID,
					ChannelID: p.ChannelID,
				},
				Content:   fmt.Sprintf("Tool result: %s", req.Tool.Name),
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"tool_name": req.Tool.Name,
					"tool_id":   req.Tool.ID,
				},
			})
		}

		return resp, err
	}
}
