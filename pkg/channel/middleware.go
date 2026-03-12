package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/shared"
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

// Process handles ProcessPart output and sends to channel
func (p *ChannelMiddleware) Process(_ context.Context, part shared.ProcessPart) error {
	if part.Text == nil && part.Type != shared.PartTypeError && part.ToolName == "" {
		return nil
	}

	switch part.Type {
	case shared.PartTypeText:
		if part.Text != nil {
			p.facade.DisplayMessage(Message{
				ID:        uuid.New(),
				Type:      MessageTypeAgentChat,
				AgentID:   p.agentID,
				AgentRole: p.agentRole,
				Content:   *part.Text,
				Timestamp: time.Now(),
			})
		}

	case shared.PartTypeToolUse:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeToolRequest,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   fmt.Sprintf("Tool use: %s", part.ToolName),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool_name": part.ToolName,
				"tool_id":   part.ToolID,
			},
		})

	case shared.PartTypeToolResult:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeToolResponse,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   fmt.Sprintf("Tool result: %s", part.ToolName),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool_name": part.ToolName,
				"tool_id":   part.ToolID,
			},
		})

	case shared.PartTypeThinking:
		if part.Text != nil {
			p.facade.DisplayMessage(Message{
				ID:        uuid.New(),
				Type:      MessageTypeThinking,
				AgentID:   p.agentID,
				AgentRole: p.agentRole,
				Content:   *part.Text,
				Timestamp: time.Now(),
			})
		}

	case shared.PartTypeError:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   part.ErrorMessage,
			Timestamp: time.Now(),
		})
	}

	return nil
}
