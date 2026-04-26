package testutil

import (
	"testing"

	"github.com/google/uuid"
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
	testSessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	session, ok := mocks.SessionMgr.GetSession(testSessionID)

	require.True(t, ok)
	assert.NotNil(t, session)
}
