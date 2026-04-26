package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/session/persistence/ent"
	"github.com/denkhaus/gollum/pkg/session/persistence/ent/message"
	"github.com/denkhaus/gollum/pkg/session/persistence/ent/session"
	"github.com/google/uuid"
)

// EntRepository implements SessionRepository using Ent ORM.
type EntRepository struct {
	client *ent.Client
	logger logger.LoggerService
}

// NewEntRepository creates a new EntRepository with the specified driver and DSN.
func NewEntRepository(driver string, dsn string, log logger.LoggerService) (*EntRepository, error) {
	client, err := ent.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrDatabaseConnection, err)
	}

	return &EntRepository{
		client: client,
		logger: log,
	}, nil
}

// NewEntRepositoryFromClient creates a new EntRepository from an existing Ent client.
// This is primarily useful for testing with enttest.
func NewEntRepositoryFromClient(client *ent.Client, log logger.LoggerService) *EntRepository {
	return &EntRepository{
		client: client,
		logger: log,
	}
}

// Shutdown closes the database connection.
// NOTE: Named Shutdown to avoid conflict with Close() session method from interface
func (r *EntRepository) Shutdown() error {
	return r.client.Close()
}

// Create persists a new session.
func (r *EntRepository) Create(ctx context.Context, session *shared.Session) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create session
	_, err = tx.Session.
		Create().
		SetSessionID(session.ID).
		SetChannelID(session.ChannelID).
		SetAgentID(session.SupervisorID).
		SetCwd(session.Cwd).
		SetState("active").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return tx.Commit()
}

// Get retrieves a session by ID.
func (r *EntRepository) Get(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
	sessionEnt, err := r.client.Session.
		Query().
		Where(session.SessionID(sessionID)).
		WithMessages().
		WithSupervisor().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
	}

	return r.entityToSession(sessionEnt)
}

// List retrieves sessions with optional filtering.
func (r *EntRepository) List(ctx context.Context, filter *repository.SessionFilter) ([]*shared.Session, error) {
	query := r.client.Session.Query()

	if filter != nil {
		if filter.ChannelID != uuid.Nil {
			query.Where(session.ChannelID(filter.ChannelID))
		}
		if filter.AgentID != uuid.Nil {
			query.Where(session.AgentID(filter.AgentID))
		}
		if filter.State != "" {
			query.Where(session.StateEQ(session.State(filter.State)))
		}
		if !filter.CreatedAfter.IsZero() {
			query.Where(session.CreatedAtGTE(filter.CreatedAfter))
		}
		if !filter.CreatedBefore.IsZero() {
			query.Where(session.CreatedAtLTE(filter.CreatedBefore))
		}
	}

	entities, err := query.
		Order(ent.Asc(session.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	sessions := make([]*shared.Session, len(entities))
	for i, ent := range entities {
		session, err := r.entityToSession(ent)
		if err != nil {
			return nil, err
		}
		sessions[i] = session
	}

	return sessions, nil
}

// Fork creates a copy of a session.
func (r *EntRepository) Fork(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get source session
	source, err := tx.Session.
		Query().
		Where(session.SessionID(sessionID)).
		WithSupervisor().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
	}

	// Create new session
	newID := uuid.New()
	_, err = tx.Session.
		Create().
		SetSessionID(newID).
		SetChannelID(source.ChannelID).
		SetAgentID(source.AgentID).
		SetCwd(source.Cwd).
		SetState("active").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create forked session: %w", err)
	}

	// Copy supervisor config if exists
	if source.Edges.Supervisor != nil {
		_, err = tx.SupervisorConfig.
			Create().
			SetSessionID(newID).
			SetModel(source.Edges.Supervisor.Model).
			SetTemperature(source.Edges.Supervisor.Temperature).
			SetMaxTokens(source.Edges.Supervisor.MaxTokens).
			SetSystemPrompt(source.Edges.Supervisor.SystemPrompt).
			SetConfigJSON(source.Edges.Supervisor.ConfigJSON).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to copy supervisor config: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.Get(ctx, newID)
}

// Close marks a session as closed.
func (r *EntRepository) Close(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	_, err := r.client.Session.
		Update().
		Where(session.SessionID(sessionID)).
		SetState("closed").
		SetClosedAt(now).
		Save(ctx)
	return err
}

// Exists checks if a session exists.
func (r *EntRepository) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	return r.client.Session.
		Query().
		Where(session.SessionID(sessionID)).
		Exist(ctx)
}

// Update updates an existing session.
func (r *EntRepository) Update(ctx context.Context, sess *shared.Session) error {
	_, err := r.client.Session.
		Update().
		Where(session.SessionID(sess.ID)).
		SetAgentID(sess.SupervisorID).
		SetCwd(sess.Cwd).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// Delete removes a session.
func (r *EntRepository) Delete(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.client.Session.
		Delete().
		Where(session.SessionID(sessionID)).
		Exec(ctx)
	return err
}

// Archive removes old sessions.
func (r *EntRepository) Archive(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	// First mark sessions as archived
	updated, err := r.client.Session.
		Update().
		Where(
			session.CreatedAtLTE(cutoff),
			session.StateEQ(session.StateActive),
		).
		SetState(session.StateArchived).
		Save(ctx)

	if err != nil {
		return 0, err
	}

	// Then delete archived sessions
	deleted, err := r.client.Session.
		Delete().
		Where(session.StateEQ(session.StateArchived)).
		Exec(ctx)

	if err != nil {
		return int64(updated), err
	}

	return int64(updated) + int64(deleted), nil
}

// AddMessage adds a message to a session.
func (r *EntRepository) AddMessage(ctx context.Context, sessionID uuid.UUID, msg channel.Message) error {
	// Get session by session_id
	sessionEnt, err := r.client.Session.
		Query().
		Where(session.SessionID(sessionID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
	}

	// Store agent role in metadata
	metadata := msg.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["agent_role"] = msg.AgentRole
	metadata["message_id"] = msg.ID

	// Convert channel.MessageType to message.Type
	msgType := message.Type(msg.Type.String())

	// Create message
	_, err = r.client.Message.
		Create().
		SetSession(sessionEnt).
		SetType(msgType).
		SetContent(msg.Content).
		SetTimestamp(msg.Timestamp).
		SetMetadata(metadata).
		Save(ctx)

	return err
}

// GetMessages retrieves messages from a session.
func (r *EntRepository) GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]channel.Message, error) {
	// Get session by session_id
	sessionEnt, err := r.client.Session.
		Query().
		Where(session.SessionID(sessionID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
	}

	// Query messages using the session edge
	entities, err := sessionEnt.QueryMessages().
		Order(ent.Asc(message.FieldTimestamp)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, err
	}

	messages := make([]channel.Message, len(entities))
	for i, ent := range entities {
		messages[i] = r.entityToMessage(ent)
	}

	return messages, nil
}

// entityToSession converts an Ent Session entity to a shared.Session.
func (r *EntRepository) entityToSession(sessionEnt *ent.Session) (*shared.Session, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &shared.Session{
		ID:           sessionEnt.SessionID,
		ChannelID:    sessionEnt.ChannelID,
		SupervisorID: sessionEnt.AgentID,
		Cwd:          sessionEnt.Cwd,
		Context:      ctx,
		CancelFunc:   cancel,
		CreatedAt:    sessionEnt.CreatedAt,
	}, nil
}

// entityToMessage converts an Ent Message entity to a channel.Message.
func (r *EntRepository) entityToMessage(msgEnt *ent.Message) channel.Message {
	metadata := msgEnt.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Extract agent role from metadata
	agentRole := ""
	if ar, ok := metadata["agent_role"].(string); ok {
		agentRole = ar
	}

	// Extract message ID from metadata
	msgID := uuid.Nil
	if id, ok := metadata["message_id"].(uuid.UUID); ok {
		msgID = id
	} else if idStr, ok := metadata["message_id"].(string); ok {
		msgID, _ = uuid.Parse(idStr)
	}

	// Convert message.Type string to channel.MessageType
	msgType := channel.MessageTypeUserChat // default
	switch string(msgEnt.Type) {
	case "user_chat":
		msgType = channel.MessageTypeUserChat
	case "agent_chat":
		msgType = channel.MessageTypeAgentChat
	case "tool_request":
		msgType = channel.MessageTypeToolRequest
	case "tool_response":
		msgType = channel.MessageTypeToolResponse
	case "thinking":
		msgType = channel.MessageTypeThinking
	case "system_info":
		msgType = channel.MessageTypeSystemInfo
	case "error":
		msgType = channel.MessageTypeError
	}

	return channel.Message{
		ID:        msgID,
		Type:      msgType,
		AgentRole: agentRole,
		Content:   msgEnt.Content,
		Timestamp: msgEnt.Timestamp,
		Metadata:  metadata,
	}
}
