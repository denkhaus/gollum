package shared

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionContext_IsValid_ValidContext(t *testing.T) {
	ctx := SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.True(t, ctx.IsValid())
}

func TestSessionContext_IsValid_EmptySessionID(t *testing.T) {
	ctx := SessionContext{
		SessionID: uuid.Nil,
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestSessionContext_IsValid_NilChannelID(t *testing.T) {
	ctx := SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
		ChannelID: uuid.Nil,
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestSessionContext_IsValid_NilAgentID(t *testing.T) {
	ctx := SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
		ChannelID: uuid.New(),
		AgentID:   uuid.Nil,
	}

	assert.False(t, ctx.IsValid())
}
