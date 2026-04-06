package session

import (
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	injector := do.New()
	manager, err := NewSessionManager(injector)

	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestCreateSession(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	session, err := manager.CreateSession(sessionID, channelID)

	require.NoError(t, err)
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, channelID, session.ChannelID)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)
	assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)

	// Verify session can be retrieved
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, sessionID, retrieved.ID)
	assert.Equal(t, channelID, retrieved.ChannelID)
}

func TestCreateSession_WithExternalSessionID(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	// Simulate ACP-provided session ID
	externalSessionID := "acp-session-12345"
	channelID := uuid.New()

	session, err := manager.CreateSession(externalSessionID, channelID)

	require.NoError(t, err)
	assert.Equal(t, externalSessionID, session.ID)
	assert.Equal(t, channelID, session.ChannelID)

	// Verify session can be retrieved with the external ID
	retrieved, exists := manager.GetSession(externalSessionID)
	assert.True(t, exists)
	assert.Equal(t, externalSessionID, retrieved.ID)
}

func TestCreateSession_MultipleSessions(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID1 := uuid.New().String()
	sessionID2 := uuid.New().String()

	_, err := manager.CreateSession(sessionID1, channelID)
	require.NoError(t, err)

	_, err = manager.CreateSession(sessionID2, channelID)
	require.NoError(t, err)

	// Sessions should have unique IDs
	assert.NotEqual(t, sessionID1, sessionID2)

	// Both should be retrievable
	_, exists := manager.GetSession(sessionID1)
	assert.True(t, exists)

	_, exists = manager.GetSession(sessionID2)
	assert.True(t, exists)
}

func TestGetSession_NotFound(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	session, exists := manager.GetSession("nonexistent")

	assert.False(t, exists)
	assert.Nil(t, session)
}

func TestCloseSession(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	_, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Close the session
	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Session should no longer exist
	_, exists := manager.GetSession(sessionID)
	assert.False(t, exists)

	// Closing again should return error
	err = manager.CloseSession(sessionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCloseSession_CancelsContext(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	sess, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Verify context is not cancelled initially
	select {
	case <-sess.Context.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	// Close the session
	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Context should now be cancelled
	select {
	case <-sess.Context.Done():
		// Expected
	default:
		t.Fatal("context should be cancelled after CloseSession")
	}
}

func TestGetSessionsByChannel(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	channelID1 := uuid.New()
	channelID2 := uuid.New()
	sessionID1 := uuid.New().String()
	sessionID2 := uuid.New().String()
	sessionID3 := uuid.New().String()

	// Create sessions for channel1
	session1, err := manager.CreateSession(sessionID1, channelID1)
	require.NoError(t, err)

	session2, err := manager.CreateSession(sessionID2, channelID1)
	require.NoError(t, err)

	// Create session for channel2
	session3, err := manager.CreateSession(sessionID3, channelID2)
	require.NoError(t, err)

	// Get sessions for channel1
	sessions := manager.GetSessionsByChannel(channelID1)
	assert.Len(t, sessions, 2)

	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}
	assert.True(t, sessionIDs[session1.ID])
	assert.True(t, sessionIDs[session2.ID])
	assert.False(t, sessionIDs[session3.ID])

	// Get sessions for channel2
	sessions = manager.GetSessionsByChannel(channelID2)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session3.ID, sessions[0].ID)
}

func TestGetSessionsByChannel_NoSessions(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	sessions := manager.GetSessionsByChannel(uuid.New())

	assert.Nil(t, sessions)
}

func TestGetSessionsByChannel_AfterClose(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID1 := uuid.New().String()
	sessionID2 := uuid.New().String()

	// Create two sessions
	_, err := manager.CreateSession(sessionID1, channelID)
	require.NoError(t, err)

	sess2, err := manager.CreateSession(sessionID2, channelID)
	require.NoError(t, err)

	// Close one session
	err = manager.CloseSession(sessionID1)
	require.NoError(t, err)

	// Should only return the remaining session
	sessions := manager.GetSessionsByChannel(channelID)
	assert.Len(t, sessions, 1)
	assert.Equal(t, sess2.ID, sessions[0].ID)
}

func TestGetSessionsByChannel_AfterCloseAll(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID := uuid.New().String()

	// Create and close a session
	_, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Should return nil (no sessions)
	sessions := manager.GetSessionsByChannel(channelID)
	assert.Nil(t, sessions)
}

func TestSetSupervisorID(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()
	supervisorID := uuid.New()

	session, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Set supervisor ID (this would typically be done by the caller)
	session.SupervisorID = supervisorID

	// Verify it's set
	retrieved, exists := manager.GetSession(sessionID)
	require.True(t, exists)
	assert.Equal(t, supervisorID, retrieved.SupervisorID)
}

func TestConcurrentAccess(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	// Create multiple sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sessionID := uuid.New().String()
			channelID := uuid.New()
			session, err := manager.CreateSession(sessionID, channelID)
			assert.NoError(t, err)
			assert.NotNil(t, session)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCreateSession_ContextCancellation(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	session, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Verify context is cancellable
	assert.NotNil(t, session.CancelFunc)

	// Verify we can cancel the context
	session.CancelFunc()

	// Context should be done
	select {
	case <-session.Context.Done():
		// Expected
	default:
		t.Fatal("context should be done after cancellation")
	}
}

func TestGetSessionsByChannel_MultipleChannels(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	// Create sessions across multiple channels
	channels := make([]uuid.UUID, 5)
	sessionsPerChannel := 3

	for c := 0; c < len(channels); c++ {
		channels[c] = uuid.New()
		for s := 0; s < sessionsPerChannel; s++ {
			sessionID := uuid.New().String()
			_, err := manager.CreateSession(sessionID, channels[c])
			require.NoError(t, err)
		}
	}

	// Verify each channel has correct number of sessions
	for _, channelID := range channels {
		sessions := manager.GetSessionsByChannel(channelID)
		assert.Len(t, sessions, sessionsPerChannel)

		// Verify all sessions belong to the correct channel
		for _, sess := range sessions {
			assert.Equal(t, channelID, sess.ChannelID)
		}
	}
}

func TestCloseSession_NonExistent(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	// Try to close a session that doesn't exist
	err := manager.CloseSession("non-existent-session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateSession_WithSameID(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := "duplicate-session-id"
	channelID1 := uuid.New()
	channelID2 := uuid.New()

	// Create first session
	session1, err := manager.CreateSession(sessionID, channelID1)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session1.ID)

	// Create second session with same ID (should overwrite)
	session2, err := manager.CreateSession(sessionID, channelID2)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session2.ID)

	// Verify we can retrieve the session and it has the new channel ID
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, channelID2, retrieved.ChannelID)
}

func TestCloseSession_ThenGet(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	// Create and verify session exists
	_, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	_, exists := manager.GetSession(sessionID)
	assert.True(t, exists)

	// Close session
	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Verify session no longer exists
	_, exists = manager.GetSession(sessionID)
	assert.False(t, exists)
}

func TestSession_TimeFields(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	beforeCreation := time.Now()
	session, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)
	afterCreation := time.Now()

	// Verify CreatedAt is set and within reasonable time range
	assert.False(t, session.CreatedAt.IsZero())
	assert.WithinDuration(t, beforeCreation, session.CreatedAt, time.Second)
	assert.WithinDuration(t, afterCreation, session.CreatedAt, time.Second)
}

func TestGetSessionsByChannel_EmptyResult(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)

	// Create a session for one channel
	sessionID := uuid.New().String()
	channelID := uuid.New()
	_, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Query a different channel that has no sessions
	otherChannelID := uuid.New()
	sessions := manager.GetSessionsByChannel(otherChannelID)

	// Should return nil (not empty slice)
	assert.Nil(t, sessions)
}

func TestConcurrentCloseAndGet(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	_, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Perform concurrent operations
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			manager.GetSession(sessionID)
			done <- true
		}()
		go func() {
			manager.GetSessionsByChannel(channelID)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGetOrCreateSession_ExistingSession(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	// Create initial session
	originalSession, err := manager.CreateSession(sessionID, channelID)
	require.NoError(t, err)

	// GetOrCreate should return the existing session
	retrievedSession, err := manager.GetOrCreateSession(sessionID, channelID)
	require.NoError(t, err)

	// Should be the same session
	assert.Equal(t, originalSession.ID, retrievedSession.ID)
	assert.Equal(t, originalSession.ChannelID, retrievedSession.ChannelID)
	assert.Same(t, originalSession.Context, retrievedSession.Context)
}

func TestGetOrCreateSession_NewSession(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	// GetOrCreate should create a new session
	session, err := manager.GetOrCreateSession(sessionID, channelID)
	require.NoError(t, err)

	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, channelID, session.ChannelID)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)

	// Verify session can be retrieved
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, sessionID, retrieved.ID)
}

func TestGetOrCreateSession_Concurrent(t *testing.T) {
	injector := do.New()
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New().String()
	channelID := uuid.New()

	// Call GetOrCreateSession concurrently
	done := make(chan bool)
	sessions := make([]*shared.Session, 0, 10)

	for i := 0; i < 10; i++ {
		go func() {
			session, err := manager.GetOrCreateSession(sessionID, channelID)
			assert.NoError(t, err)
			sessions = append(sessions, session)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All sessions should have the same ID
	for _, sess := range sessions {
		assert.Equal(t, sessionID, sess.ID)
		assert.Equal(t, channelID, sess.ChannelID)
	}
}

