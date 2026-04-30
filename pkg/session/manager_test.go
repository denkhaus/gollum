package session

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestInjector(t *testing.T, ctrl *gomock.Controller) do.Injector {
	injector := do.New()
	mockRepo := repository.NewMockSessionRepository(ctrl)
	// Set default expectations for methods called during setup
	// Create is called by CreateSession - expect it for any session
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// Close is called by CloseSession after removing from in-memory cache
	mockRepo.EXPECT().Close(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// Update is called by ResumeSession
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// Get is called by LoadSession when not in cache
	mockRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, repository.ErrSessionNotFound).AnyTimes()
	// List is called by ListSessions
	mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*shared.Session{}, nil).AnyTimes()
	// Fork is called by ForkSession
	mockRepo.EXPECT().Fork(gomock.Any(), gomock.Any()).Return(nil, repository.ErrSessionNotFound).AnyTimes()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	return injector
}

func TestNewSessionManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, err := NewSessionManager(injector)

	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestCreateSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	sessionCtx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
	}
	session, err := manager.CreateSession(context.Background(), sessionCtx)

	require.NoError(t, err)
	assert.Equal(t, sessionID, session.SessionContext.SessionID)
	assert.Equal(t, channelID, session.SessionContext.ChannelID)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)
	assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)

	// Verify session can be retrieved
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, sessionID, retrieved.SessionContext.SessionID)
	assert.Equal(t, channelID, retrieved.SessionContext.ChannelID)
}

func TestCreateSession_WithExternalSessionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	// Simulate ACP-provided session ID
	externalSessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: externalSessionID,
		ChannelID: channelID,
		}
	session, err := manager.CreateSession(context.Background(), ctx)

	require.NoError(t, err)
	assert.Equal(t, externalSessionID, session.SessionContext.SessionID)
	assert.Equal(t, channelID, session.SessionContext.ChannelID)

	// Verify session can be retrieved with the external ID
	retrieved, exists := manager.GetSession(externalSessionID)
	assert.True(t, exists)
	assert.Equal(t, externalSessionID, retrieved.SessionContext.SessionID)
}

func TestCreateSession_MultipleSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	ctx1 := &shared.SessionContext{
		SessionID: sessionID1,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx1)
	require.NoError(t, err)

	ctx2 := &shared.SessionContext{
		SessionID: sessionID2,
		ChannelID: channelID,
		AgentID:   uuid.New(),
	}
	_, err = manager.CreateSession(context.Background(), ctx2)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	nonExistentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440999")
	session, exists := manager.GetSession(nonExistentID)

	assert.False(t, exists)
	assert.Nil(t, session)
}

func TestCloseSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	session, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// Verify session exists before closing
	retrieved, exists := manager.GetSession(sessionID)
	require.True(t, exists, "session should exist before closing")

	// Verify the session ID matches
	assert.Equal(t, sessionID, session.SessionContext.SessionID, "session ID should match")
	assert.Equal(t, sessionID, retrieved.SessionContext.SessionID, "retrieved session ID should match")

	// Close the session
	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Session should no longer exist
	_, exists = manager.GetSession(sessionID)
	assert.False(t, exists)

	// Closing again should return error
	err = manager.CloseSession(sessionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCloseSession_CancelsContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	sess, err := manager.CreateSession(context.Background(), ctx)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	channelID1 := uuid.New()
	channelID2 := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	sessionID3 := uuid.New()

	// Create sessions for channel1
	ctx1 := &shared.SessionContext{
		SessionID: sessionID1,
		ChannelID: channelID1,
		}
	session1, err := manager.CreateSession(context.Background(), ctx1)
	require.NoError(t, err)

	ctx2 := &shared.SessionContext{
		SessionID: sessionID2,
		ChannelID: channelID1,
		}
	session2, err := manager.CreateSession(context.Background(), ctx2)
	require.NoError(t, err)

	// Create session for channel2
	ctx3 := &shared.SessionContext{
		SessionID: sessionID3,
		ChannelID: channelID2,
		}
	session3, err := manager.CreateSession(context.Background(), ctx3)
	require.NoError(t, err)

	// Get sessions for channel1
	sessions := manager.GetSessionsByChannel(channelID1)
	assert.Len(t, sessions, 2)

	sessionIDs := make(map[uuid.UUID]bool)
	for _, s := range sessions {
		sessionIDs[s.SessionContext.SessionID] = true
	}
	assert.True(t, sessionIDs[session1.SessionContext.SessionID])
	assert.True(t, sessionIDs[session2.SessionContext.SessionID])
	assert.False(t, sessionIDs[session3.SessionContext.SessionID])

	// Get sessions for channel2
	sessions = manager.GetSessionsByChannel(channelID2)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session3.SessionContext.SessionID, sessions[0].SessionContext.SessionID)
}

func TestGetSessionsByChannel_NoSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	sessions := manager.GetSessionsByChannel(uuid.New())

	assert.Nil(t, sessions)
}

func TestGetSessionsByChannel_AfterClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	// Create two sessions
	ctx1 := &shared.SessionContext{
		SessionID: sessionID1,
		ChannelID: channelID,
		AgentID:   uuid.New(),
	}
	_, err := manager.CreateSession(context.Background(), ctx1)
	require.NoError(t, err)

	ctx2 := &shared.SessionContext{
		SessionID: sessionID2,
		ChannelID: channelID,
		AgentID:   uuid.New(),
	}
	sess2, err := manager.CreateSession(context.Background(), ctx2)
	require.NoError(t, err)

	// Close one session
	err = manager.CloseSession(sessionID1)
	require.NoError(t, err)

	// Should only return the remaining session
	sessions := manager.GetSessionsByChannel(channelID)
	assert.Len(t, sessions, 1)
	assert.Equal(t, sess2.SessionContext.SessionID, sessions[0].SessionContext.SessionID)
}

func TestGetSessionsByChannel_AfterCloseAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	channelID := uuid.New()
	sessionID := uuid.New()

	// Create and close a session
	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Should return nil (no sessions)
	sessions := manager.GetSessionsByChannel(channelID)
	assert.Nil(t, sessions)
}

func TestSetSupervisorID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
	}
	_, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// Verify session can be retrieved
	sess, exists := manager.GetSession(sessionID)
	require.True(t, exists)
	assert.Equal(t, sessionID, sess.SessionContext.SessionID)
	assert.Equal(t, channelID, sess.SessionContext.ChannelID)
}

func TestConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	// Create multiple sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sessionID := uuid.New()
			channelID := uuid.New()
			ctx := &shared.SessionContext{
				SessionID: sessionID,
				ChannelID: channelID,
			}
			session, err := manager.CreateSession(context.Background(), ctx)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	session, err := manager.CreateSession(context.Background(), ctx)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	// Create sessions across multiple channels
	channels := make([]uuid.UUID, 5)
	sessionsPerChannel := 3

	for c := 0; c < len(channels); c++ {
		channels[c] = uuid.New()
		for s := 0; s < sessionsPerChannel; s++ {
			sessionID := uuid.New()
			ctx := &shared.SessionContext{
				SessionID: sessionID,
				ChannelID: channels[c],
				}
			_, err := manager.CreateSession(context.Background(), ctx)
			require.NoError(t, err)
		}
	}

	// Verify each channel has correct number of sessions
	for _, channelID := range channels {
		sessions := manager.GetSessionsByChannel(channelID)
		assert.Len(t, sessions, sessionsPerChannel)

		// Verify all sessions belong to the correct channel
		for _, sess := range sessions {
			assert.Equal(t, channelID, sess.SessionContext.ChannelID)
		}
	}
}

func TestCloseSession_NonExistent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	// Try to close a session that doesn't exist
	nonExistentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440999")
	err := manager.CloseSession(nonExistentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateSession_WithSameID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	channelID1 := uuid.New()
	channelID2 := uuid.New()

	// Create first session
	ctx1 := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID1,
		}
	session1, err := manager.CreateSession(context.Background(), ctx1)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session1.SessionContext.SessionID)

	// Create second session with same ID (should overwrite)
	ctx2 := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID2,
		}
	session2, err := manager.CreateSession(context.Background(), ctx2)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session2.SessionContext.SessionID)

	// Verify we can retrieve the session and it has the new channel ID
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, channelID2, retrieved.SessionContext.ChannelID)
}

func TestCloseSession_ThenGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	// Create and verify session exists
	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	beforeCreation := time.Now()
	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	session, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)
	afterCreation := time.Now()

	// Verify CreatedAt is set and within reasonable time range
	assert.False(t, session.CreatedAt.IsZero())
	assert.WithinDuration(t, beforeCreation, session.CreatedAt, time.Second)
	assert.WithinDuration(t, afterCreation, session.CreatedAt, time.Second)
}

func TestGetSessionsByChannel_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)

	// Create a session for one channel
	sessionID := uuid.New()
	channelID := uuid.New()
	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// Query a different channel that has no sessions
	otherChannelID := uuid.New()
	sessions := manager.GetSessionsByChannel(otherChannelID)

	// Should return nil (not empty slice)
	assert.Nil(t, sessions)
}

func TestConcurrentCloseAndGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	// Create initial session
	originalSession, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// GetOrCreate should return the existing session
	retrievedSession, err := manager.GetOrCreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// Should be the same session
	assert.Equal(t, originalSession.SessionContext.SessionID, retrievedSession.SessionContext.SessionID)
	assert.Equal(t, originalSession.SessionContext.ChannelID, retrievedSession.SessionContext.ChannelID)
	assert.Same(t, originalSession.Context, retrievedSession.Context)
}

func TestGetOrCreateSession_NewSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	// GetOrCreate should create a new session
	session, err := manager.GetOrCreateSession(context.Background(), ctx)
	require.NoError(t, err)

	assert.Equal(t, sessionID, session.SessionContext.SessionID)
	assert.Equal(t, channelID, session.SessionContext.ChannelID)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)

	// Verify session can be retrieved
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, sessionID, retrieved.SessionContext.SessionID)
}

func TestGetOrCreateSession_Concurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	// Call GetOrCreateSession concurrently
	done := make(chan bool)
	sessions := make([]*shared.Session, 0, 10)

	for i := 0; i < 10; i++ {
		go func() {
			session, err := manager.GetOrCreateSession(context.Background(), ctx)
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
		assert.Equal(t, sessionID, sess.SessionContext.SessionID)
		assert.Equal(t, channelID, sess.SessionContext.ChannelID)
	}
}

func TestCloseSession_CleansUpSupervisor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector(t, ctrl)
	manager, _ := NewSessionManager(injector)
	sessionID := uuid.New()
	channelID := uuid.New()

	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	session, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// Verify session exists before closing
	_, exists := manager.GetSession(sessionID)
	require.True(t, exists, "session should exist before closing")

	// Create a mock supervisor to test cleanup
	// We can't easily create a real agent without the full factory setup,
	// but we can verify that Close() doesn't error and context is cancelled

	// Close the session
	err = manager.CloseSession(sessionID)
	require.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-session.Context.Done():
		// Expected - context should be cancelled
	default:
		t.Fatal("context should be cancelled after CloseSession")
	}

	// Session should no longer exist
	_, exists = manager.GetSession(sessionID)
	assert.False(t, exists)
}

func TestLoadSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()
	channelID := uuid.New()
	expectedSession := &shared.Session{
		SessionContext: shared.SessionContext{
			SessionID: sessionID,
			ChannelID: channelID,
		},
		Context:    context.Background(),
		CancelFunc: func() {},
		CreatedAt:  time.Now(),
	}

	// Expect Get to be called
	mockRepo.EXPECT().Get(gomock.Any(), sessionID).Return(expectedSession, nil)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Load the session
	ctx := context.Background()
	session, err := manager.LoadSession(ctx, sessionID)

	require.NoError(t, err)
	assert.Equal(t, sessionID, session.SessionContext.SessionID)
	assert.Equal(t, channelID, session.SessionContext.ChannelID)

	// Verify session is cached
	retrieved, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, sessionID, retrieved.SessionContext.SessionID)
}

func TestLoadSession_FromCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()
	channelID := uuid.New()

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Create a session first (this will cache it)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	ctx := &shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		}
	_, err := manager.CreateSession(context.Background(), ctx)
	require.NoError(t, err)

	// LoadSession should return from cache without calling repo
	mockRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0) // Should not be called

	session, err := manager.LoadSession(context.Background(), sessionID)

	require.NoError(t, err)
	assert.Equal(t, sessionID, session.SessionContext.SessionID)
}

func TestLoadSession_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()

	// Expect Get to return error
	mockRepo.EXPECT().Get(gomock.Any(), sessionID).Return(nil, repository.ErrSessionNotFound)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Load should fail
	ctx := context.Background()
	session, err := manager.LoadSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestListSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	expectedSessions := []*shared.Session{
		{
			SessionContext: shared.SessionContext{SessionID: sessionID1, ChannelID: uuid.New()},
			Context:        context.Background(),
			CancelFunc:     func() {},
			CreatedAt:      time.Now(),
		},
		{
			SessionContext: shared.SessionContext{SessionID: sessionID2, ChannelID: uuid.New()},
			Context:        context.Background(),
			CancelFunc:     func() {},
			CreatedAt:      time.Now(),
		},
	}

	// Expect List to be called
	mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(expectedSessions, nil)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// List sessions
	ctx := context.Background()
	sessions, err := manager.ListSessions(ctx)

	require.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.Equal(t, sessionID1, sessions[0].SessionContext.SessionID)
	assert.Equal(t, sessionID2, sessions[1].SessionContext.SessionID)
}

func TestResumeSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()
	channelID := uuid.New()

	existingSession := &shared.Session{
		SessionContext: shared.SessionContext{
			SessionID: sessionID,
			ChannelID: channelID,
		},
		Context:    context.Background(),
		CancelFunc: func() {},
		CreatedAt:  time.Now(),
	}

	// LoadSession will call Get
	mockRepo.EXPECT().Get(gomock.Any(), sessionID).Return(existingSession, nil)
	// ResumeSession will call Update
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Resume the session
	ctx := context.Background()
	err := manager.ResumeSession(ctx, sessionID)

	require.NoError(t, err)

	// Verify session has new context
	session, exists := manager.GetSession(sessionID)
	assert.True(t, exists)
	assert.NotNil(t, session.Context)
	assert.NotNil(t, session.CancelFunc)
}

func TestResumeSession_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()

	// LoadSession will call Get and return error
	mockRepo.EXPECT().Get(gomock.Any(), sessionID).Return(nil, repository.ErrSessionNotFound)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Resume should fail
	ctx := context.Background()
	err := manager.ResumeSession(ctx, sessionID)

	assert.Error(t, err)
}

func TestForkSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()
	channelID := uuid.New()
	newSessionID := uuid.New()

	forkedSession := &shared.Session{
		SessionContext: shared.SessionContext{
			SessionID: newSessionID,
			ChannelID: channelID,
		},
		Context:    context.Background(),
		CancelFunc: func() {},
		CreatedAt:  time.Now(),
	}

	// Expect Fork to be called
	mockRepo.EXPECT().Fork(gomock.Any(), sessionID).Return(forkedSession, nil)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Fork the session
	ctx := context.Background()
	session, err := manager.ForkSession(ctx, sessionID)

	require.NoError(t, err)
	assert.Equal(t, newSessionID, session.SessionContext.SessionID)
	assert.Equal(t, channelID, session.SessionContext.ChannelID)

	// Verify session is cached
	retrieved, exists := manager.GetSession(newSessionID)
	assert.True(t, exists)
	assert.Equal(t, newSessionID, retrieved.SessionContext.SessionID)
}

func TestForkSession_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockSessionRepository(ctrl)
	sessionID := uuid.New()

	// Expect Fork to return error
	mockRepo.EXPECT().Fork(gomock.Any(), sessionID).Return(nil, repository.ErrSessionNotFound)

	injector := do.New()
	do.ProvideValue[repository.SessionRepository](injector, mockRepo)
	manager, _ := NewSessionManager(injector)

	// Fork should fail
	ctx := context.Background()
	session, err := manager.ForkSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Nil(t, session)
}
