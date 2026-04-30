package acp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/denkhaus/gollum/pkg/shared"
)

func TestAcpSession_NewSession_HasRequiredFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := shared.NewSession(ctx, cancel, "/test/cwd")

	assert.NotNil(t, session)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)
	assert.Equal(t, "/test/cwd", session.Cwd)
}
