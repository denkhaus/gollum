// Package persistence provides database-backed implementations for session persistence.
// This file implements a bridge between Gollem's HistoryRepository and our SessionRepository.
package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

// Metadata key constants for Gollem message metadata
const (
	MetadataKeyMessageID = "message_id"
	MetadataKeySessionID = "session_id"
	MetadataKeyChannelID = "channel_id"
	MetadataKeyAgentID   = "agent_id"
	MetadataKeyAgentRole = "agent_role"
	MetadataKeyTimestamp = "timestamp"
	MetadataKeyLLMType   = "llm_type"
)

// gollumHistoryAdapter implements gollem.HistoryRepository using our SessionRepository.
// It bridges between Gollem's internal message format and our shared.Message format.
//
// This is a singleton service that handles history for ALL sessions. The sessionID
// parameter in Load/Save methods routes each operation to the correct session.
type gollumHistoryAdapter struct {
	repo repository.SessionRepository
}

// NewGollumHistoryAdapter creates a new HistoryRepository adapter (DI constructor).
// This is a singleton service that can handle history operations for any session.
// Returns the gollem.HistoryRepository interface for dependency injection.
func NewGollumHistoryAdapter(injector do.Injector) (gollem.HistoryRepository, error) {
	repo, err := do.Invoke[repository.SessionRepository](injector)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke SessionRepository: %w", err)
	}

	return &gollumHistoryAdapter{
		repo: repo,
	}, nil
}

// Load retrieves conversation history from the database for the given session ID.
// Returns nil history if the session has no messages (not an error per Gollem's contract).
func (a *gollumHistoryAdapter) Load(ctx context.Context, sessionID string) (*gollem.History, error) {
	// Parse session ID
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	// Check if session exists
	exists, err := a.repo.Exists(ctx, sid)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if !exists {
		// Session doesn't exist - return empty history (not an error for Gollem)
		return nil, nil
	}

	// Load messages from database
	messages, err := a.repo.GetMessages(ctx, sid, 0, 0) // No limit/offset for full history
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	// Convert shared.Messages to gollem.Messages
	gollemMessages := make([]gollem.Message, 0, len(messages))
	for _, msg := range messages {
		gMsg, err := sharedMessageToGollem(msg)
		if err != nil {
			// Skip invalid messages but continue loading others
			continue
		}
		gollemMessages = append(gollemMessages, gMsg)
	}

	// Determine LLM type from messages or default to OpenAI
	llmType := gollem.LLMTypeOpenAI
	if len(gollemMessages) > 0 {
		// Try to infer LLM type from message metadata
		if typeVal, ok := gollemMessages[0].Metadata[MetadataKeyLLMType].(string); ok {
			llmType = gollem.LLMType(typeVal)
		}
	}

	return &gollem.History{
		LLType:   llmType,
		Version:  gollem.HistoryVersion,
		Messages: gollemMessages,
	}, nil
}

// Save persists conversation history to the database for the given session ID.
// It appends new messages to the session rather than overwriting.
func (a *gollumHistoryAdapter) Save(ctx context.Context, sessionID string, history *gollem.History) error {
	// Parse session ID
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	// Check if session exists
	exists, err := a.repo.Exists(ctx, sid)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	if !exists {
		// Session doesn't exist - create it with minimal info
		sess := &shared.Session{
			SessionContext: shared.SessionContext{
				SessionID:        sid,
				ChannelID:        uuid.Nil, // Unknown at this point
				AgentID:          uuid.Nil,
				StartupDirectory: "",
			},
			CreatedAt: time.Now(),
		}
		if err := a.repo.Create(ctx, sess); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Convert gollem.Messages to shared.Messages
	sharedMessages := make([]shared.Message, 0, len(history.Messages))
	for _, gMsg := range history.Messages {
		sMsg, err := gollemMessageToShared(gMsg, sid)
		if err != nil {
			// Skip invalid messages but continue saving others
			continue
		}
		sharedMessages = append(sharedMessages, sMsg)
	}

	// Save each message to the database
	// Note: This is a simple implementation. For production, consider:
	// - Batch insert for better performance
	// - Upsert semantics to handle duplicate messages gracefully
	// - Transaction wrapping for atomicity
	for _, msg := range sharedMessages {
		if err := a.repo.AddMessage(ctx, sid, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
	}

	return nil
}

// sharedMessageToGollem converts a shared.Message to a gollem.Message.
func sharedMessageToGollem(msg shared.Message) (gollem.Message, error) {
	// Use the Role directly (already gollem.MessageRole)
	role := msg.Role

	// Convert content
	contentBytes, err := json.Marshal(gollem.TextContent{Text: msg.Content})
	if err != nil {
		return gollem.Message{}, err
	}

	gMsg := gollem.Message{
		Role: role,
		Contents: []gollem.MessageContent{
			{
				Type: gollem.MessageContentTypeText,
				Data: contentBytes,
			},
		},
		Metadata: make(map[string]interface{}),
	}

	// Copy metadata using constants
	if msg.Metadata != nil {
		for k, v := range msg.Metadata {
			gMsg.Metadata[k] = v
		}
	}

	// Store our internal IDs using constants
	gMsg.Metadata[MetadataKeyMessageID] = msg.SessionID
	if msg.SessionID != uuid.Nil {
		gMsg.Metadata[MetadataKeySessionID] = msg.SessionID
	}
	if msg.ChannelID != uuid.Nil {
		gMsg.Metadata[MetadataKeyChannelID] = msg.ChannelID
	}
	if msg.AgentID != uuid.Nil {
		gMsg.Metadata[MetadataKeyAgentID] = msg.AgentID
	}
	gMsg.Metadata[MetadataKeyAgentRole] = msg.AgentRole
	gMsg.Metadata[MetadataKeyTimestamp] = msg.Timestamp

	return gMsg, nil
}

// gollemMessageToShared converts a gollem.Message to a shared.Message.
func gollemMessageToShared(gMsg gollem.Message, sessionID uuid.UUID) (shared.Message, error) {
	// Extract text content
	var content string
	if len(gMsg.Contents) > 0 {
		for _, c := range gMsg.Contents {
			if c.Type == gollem.MessageContentTypeText {
				var textContent gollem.TextContent
				if err := json.Unmarshal(c.Data, &textContent); err == nil {
					content = textContent.Text
				}
			}
		}
	}

	// Extract message ID from metadata or generate new one
	msgID := uuid.New()
	if idVal, ok := gMsg.Metadata[MetadataKeyMessageID]; ok {
		if idStr, ok := idVal.(string); ok {
			if parsed, err := uuid.Parse(idStr); err == nil {
				msgID = parsed
			}
		}
	}

	// Extract timestamp from metadata or use current time
	var timestamp time.Time
	if tsVal, ok := gMsg.Metadata[MetadataKeyTimestamp]; ok {
		if ts, ok := tsVal.(time.Time); ok {
			timestamp = ts
		}
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Extract agent ID from metadata using constant
	agentID := uuid.Nil
	if aidVal, ok := gMsg.Metadata[MetadataKeyAgentID]; ok {
		if aidStr, ok := aidVal.(string); ok {
			if parsed, err := uuid.Parse(aidStr); err == nil {
				agentID = parsed
			}
		}
	}

	// Extract channel ID from metadata using constant
	channelID := uuid.Nil
	if cidVal, ok := gMsg.Metadata[MetadataKeyChannelID]; ok {
		if cidStr, ok := cidVal.(string); ok {
			if parsed, err := uuid.Parse(cidStr); err == nil {
				channelID = parsed
			}
		}
	}

	// Extract agent role from metadata for UI purposes
	agentRole := string(gMsg.Role)
	if arVal, ok := gMsg.Metadata[MetadataKeyAgentRole]; ok {
		if arStr, ok := arVal.(string); ok {
			agentRole = arStr
		}
	}

	// Copy remaining metadata (excluding our internal keys)
	metadata := make(map[string]interface{})
	internalKeys := map[string]bool{
		MetadataKeyMessageID: true,
		MetadataKeySessionID: true,
		MetadataKeyChannelID: true,
		MetadataKeyAgentID:   true,
		MetadataKeyAgentRole: true,
		MetadataKeyTimestamp: true,
		MetadataKeyLLMType:   true,
	}
	for k, v := range gMsg.Metadata {
		if !internalKeys[k] {
			metadata[k] = v
		}
	}

	return shared.Message{
		ID:        msgID,
		Role:      gMsg.Role,
		AgentRole: agentRole,
		SessionContext: shared.SessionContext{
			SessionID: sessionID,
			ChannelID: channelID,
			AgentID:   agentID,
		},
		Content:   content,
		Timestamp: timestamp,
		Metadata:  metadata,
	}, nil
}
