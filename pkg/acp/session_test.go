package acp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcpSession_NewSession_HasRequiredFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := NewAcpSession(ctx, cancel)

	assert.NotNil(t, session)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)
}
