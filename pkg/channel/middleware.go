package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// ChannelMiddleware sends agent outputs to channel facade.
// This middleware bridges the agent execution pipeline with the channel system.
type ChannelMiddleware struct {
	facade    ChannelFacade
	agentID   uuid.UUID
	agentRole string
}

// NewChannelMiddleware creates a new channel middleware
func NewChannelMiddleware(facade ChannelFacade, agentID uuid.UUID, agentRole string) *ChannelMiddleware {
	return &ChannelMiddleware{
		facade:    facade,
		agentID:   agentID,
		agentRole: agentRole,
	}
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
				p.facade.DisplayMessage(Message{
					ID:        uuid.New(),
					Type:      MessageTypeAgentChat,
					AgentID:   p.agentID,
					AgentRole: p.agentRole,
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
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeToolRequest,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
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
			p.facade.DisplayMessage(Message{
				ID:        uuid.New(),
				Type:      MessageTypeToolResponse,
				AgentID:   p.agentID,
				AgentRole: p.agentRole,
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
