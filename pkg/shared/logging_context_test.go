package shared

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoggingContext_IsValid_ValidContext(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.True(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_EmptySessionID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "",
		ChannelID: uuid.New(),
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_NilChannelID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.Nil,
		AgentID:   uuid.New(),
	}

	assert.False(t, ctx.IsValid())
}

func TestLoggingContext_IsValid_NilAgentID(t *testing.T) {
	ctx := LoggingContext{
		SessionID: "test-session",
		ChannelID: uuid.New(),
		AgentID:   uuid.Nil,
	}

	assert.False(t, ctx.IsValid())
}
