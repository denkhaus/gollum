package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommonMocks_CreatesAllMocks(t *testing.T) {
	mocks := NewCommonMocks(t)

	assert.NotNil(t, mocks.Ctrl)
	assert.NotNil(t, mocks.Logger)
	assert.NotNil(t, mocks.HookManager)
	assert.NotNil(t, mocks.SessionMgr)
}

func TestMocksWithController_SetupPassThrough(t *testing.T) {
	mocks := NewCommonMocks(t)
	mocks.SetupPassThrough()

	// Should not panic when calling methods
	mocks.Logger.Info("test")
	session, ok := mocks.SessionMgr.GetSession("test")

	require.True(t, ok)
	assert.NotNil(t, session)
}
