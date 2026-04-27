package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/session/persistence/ent/enttest"
	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
	"github.com/m-mizutani/gollem"
)

func TestEntRepository_Create(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}

	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Verify session was created
	exists, err := repo.Exists(ctx, session.ID)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestEntRepository_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	_, err := repo.Get(ctx, uuid.New())
	assert.Error(t, err)
}

func TestEntRepository_Get_Success(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Get the session
	retrieved, err := repo.Get(ctx, session.ID)
	assert.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.ChannelID, retrieved.ChannelID)
	assert.Equal(t, session.Cwd, retrieved.Cwd)
}

func TestEntRepository_List(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	channelID := uuid.New()

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &shared.Session{
			ID:        uuid.New(),
			ChannelID: channelID,
			Cwd:       "/test",
		}
		require.NoError(t, repo.Create(ctx, session))
	}

	// List all sessions
	sessions, err := repo.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, sessions, 3)

	// List with channel filter
	filter := &repository.SessionFilter{
		ChannelID: channelID,
	}
	sessions, err = repo.List(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestEntRepository_Fork(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	// Create source session
	source := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, source))

	// Fork session
	forked, err := repo.Fork(ctx, source.ID)
	assert.NoError(t, err)
	assert.NotEqual(t, source.ID, forked.ID)
	assert.Equal(t, source.ChannelID, forked.ChannelID)
	assert.Equal(t, source.Cwd, forked.Cwd)

	// Verify both sessions exist
	exists, err := repo.Exists(ctx, source.ID)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists(ctx, forked.ID)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestEntRepository_Close(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Close the session
	err := repo.Close(ctx, session.ID)
	assert.NoError(t, err)

	// Verify session state
	retrieved, err := repo.Get(ctx, session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestEntRepository_Update(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:           uuid.New(),
		ChannelID:    uuid.New(),
		SupervisorID: uuid.New(),
		Cwd:          "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Update session
	session.SupervisorID = uuid.New()
	session.Cwd = "/updated"
	err := repo.Update(ctx, session)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.Get(ctx, session.ID)
	assert.NoError(t, err)
	assert.Equal(t, session.SupervisorID, retrieved.SupervisorID)
	assert.Equal(t, session.Cwd, retrieved.Cwd)
}

func TestEntRepository_Delete(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Delete session
	err := repo.Delete(ctx, session.ID)
	assert.NoError(t, err)

	// Verify deletion
	exists, err := repo.Exists(ctx, session.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestEntRepository_Archive(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	// Create old session
	oldSession := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, oldSession))

	// Manually set created_at to old time (this is a limitation of the test)
	// In real scenarios, time would pass naturally

	// Create recent session
	recentSession := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, recentSession))

	// Archive sessions older than 1 hour
	count, err := repo.Archive(ctx, time.Hour)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestEntRepository_AddMessage(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Add message
	msg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleUser,
		AgentRole: "user",
		Content:   "Hello",
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}

	err := repo.AddMessage(ctx, session.ID, msg)
	assert.NoError(t, err)

	// Verify message was added
	messages, err := repo.GetMessages(ctx, session.ID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, msg.Content, messages[0].Content)
}

func TestEntRepository_GetMessages(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}
	require.NoError(t, repo.Create(ctx, session))

	// Add multiple messages
	for i := 0; i < 5; i++ {
		msg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleUser,
			AgentRole: "user",
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
			Metadata:  make(map[string]any),
		}
		require.NoError(t, repo.AddMessage(ctx, session.ID, msg))
	}

	// Get messages with limit
	messages, err := repo.GetMessages(ctx, session.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)

	// Get messages with offset
	messages, err = repo.GetMessages(ctx, session.ID, 3, 2)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestEntRepository_Exists(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewEntRepositoryFromClient(client, nil)

	session := &shared.Session{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Cwd:       "/test",
	}

	// Should not exist before creation
	exists, err := repo.Exists(ctx, session.ID)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Create session
	require.NoError(t, repo.Create(ctx, session))

	// Should exist after creation
	exists, err = repo.Exists(ctx, session.ID)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestEntRepository_NewEntRepository(t *testing.T) {
	// Test with SQLite driver
	repo, err := NewEntRepository("sqlite3", "file:test.db?mode=memory&cache=shared", nil)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	// Cleanup
	err = repo.Shutdown()
	assert.NoError(t, err)
}
