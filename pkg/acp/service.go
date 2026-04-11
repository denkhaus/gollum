// Package acp provides the ACP (Agent Communication Protocol) service implementation.
//
// ARCHITECTURE OVERVIEW:
//
// The acpServiceImpl has a DUAL ROLE:
//
//  1. ACP AGENT INTERFACE (outward to ACP client):
//     Implements acppkg.Agent to communicate with ACP clients:
//     - Initialize()          - Protocol handshake, capabilities negotiation
//     - SetSessionMode()      - Configure session behavior
//     - SetSessionConfigOption() - Set session-specific options
//     - Prompt()              - Main execution loop: receive prompt, return final answer
//     - Cancel()              - Cancel in-flight prompt
//
//  2. CHANNEL INTERFACE (inward to Gollum system):
//     Implements channel.Channel to integrate with Gollum's channel system:
//     - ID()                  - Unique channel identifier
//     - OnMessage()           - Receive messages FROM agent system (stream to client)
//     - OnLog()               - Receive log entries
//     - OnAgentLifecycle()    - Receive agent registration/removal events
//
// DATA FLOW:
//
// PROMPT FLOW (TO agent system):
//
//	ACP Client → Prompt() → facade.SubmitInput(channelID, input) → Agents
//
// RESPONSE FLOW (FROM agent system):
//
//	Agents → facade.DisplayMessage() → OnMessage() → stream.SendText() → ACP Client
//
// SESSION MANAGEMENT:
//
// - Each Gollum-ACP process creates ONE connection via NewConnection()
// - The connection can create MULTIPLE sessions (multi-session support)
// - Session lifecycle follows the ACP protocol:
//
//   1. Initialize() - Protocol handshake, NO session created yet
//   2. First Prompt() call for a new session - Triggers NewSession callback
//   3. NewSession callback (in connection.go) creates the session BEFORE Prompt() executes
//   4. Subsequent Prompt() calls reuse existing session (identified by SessionID)
//
// - Sessions are stored in s.store and looked up by SessionID
// - Each OnMessage() call includes the target SessionID for routing to specific sessions
// - OnLog() broadcasts to ALL active sessions (logs are system-wide, not session-specific)
//
// PROTOCOL BEHAVIOR:
//
// - Prompt() method returns ONLY the final answer to the ACP client
// - All intermediate messages (logs, progress, etc.) must be streamed via OnMessage()
// - The Prompt() method blocks until the agent system completes the turn
package acp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
)

// acpServiceImpl implements Service and channel.Channel (PRIVATE)
type acpServiceImpl struct {
	facade channel.ChannelFacade
	logger logger.LoggerService
	client acppkg.Client
	store  acppkg.SessionStore[*shared.ACPSession]
	id     uuid.UUID // Channel ID
}

// Ensure acpServiceImpl implements Service and channel.Channel at compile time
var _ shared.ACPService = (*acpServiceImpl)(nil)
var _ channel.Channel = (*acpServiceImpl)(nil)

// NewAcpService creates a new ACP service with DI
func NewAcpService(injector do.Injector) (shared.ACPService, error) {
	logger := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	logger.Debug("startup ACP service")

	// Generate unique channel ID for this ACP service instance
	id := uuid.New()

	svc := &acpServiceImpl{
		logger: logger,
		facade: facade,
		id:     id,
	}

	// Register self with facade (breaks circular dependency)
	if err := facade.RegisterChannel(svc); err != nil {
		return nil, fmt.Errorf("failed to register ACP service as channel: %w", err)
	}

	logger.Debug("ACP service registered as channel", zap.String("channel_id", id.String()))

	return svc, nil
}

// SetClient sets the ACP client (called by connection factory)
func (s *acpServiceImpl) SetClient(client acppkg.Client) {
	s.client = client
}

// SetSessionStore sets the session store (called by connection factory)
func (s *acpServiceImpl) SetSessionStore(store acppkg.SessionStore[*shared.ACPSession]) {
	s.store = store
}

// =============================================================================
// Channel Interface Implementation
// =============================================================================

// ID returns the unique channel identifier for this ACP service
func (s *acpServiceImpl) ID() uuid.UUID {
	return s.id
}

// OnMessage receives messages from the agent system and streams them to the ACP client
func (s *acpServiceImpl) OnMessage(msg channel.Message) {
	// Convert string SessionID to acppkg.SessionID
	sessionID := acppkg.SessionID(msg.SessionID)

	// Get session from store using SessionID from message
	session, ok := s.store.Get(sessionID)
	if !ok {
		s.logger.Warn("received message but session not found",
			zap.String("channel_id", s.id.String()),
			zap.String("session_id", msg.SessionID),
			zap.String("message_content", msg.Content),
		)
		return
	}

	// Stream message to ACP client
	stream := acppkg.NewSessionStream(s.client, sessionID)
	if err := stream.SendText(session.Context, msg.Content); err != nil {
		s.logger.Error("failed to stream message to ACP client",
			zap.String("session_id", msg.SessionID),
			zap.Error(err),
		)
	}
}

// OnLog receives log entries from the agent system and forwards them to the appropriate ACP session
func (s *acpServiceImpl) OnLog(entry shared.LogEntry) {
	// Log locally for debugging
	s.logger.Debug("log entry from agent system",
		zap.String("level", entry.Level),
		zap.String("message", entry.Message),
		zap.String("session_id", entry.SessionID),
		zap.String("channel_id", entry.ChannelID.String()),
	)

	// Format: [LEVEL] message
	logMsg := fmt.Sprintf("[%s] %s", entry.Level, entry.Message)

	// If session is specified, route to that specific session
	if entry.SessionID != "" {
		sessionID := acppkg.SessionID(entry.SessionID)
		session, ok := s.store.Get(sessionID)
		if !ok {
			s.logger.Warn("log entry specifies session but session not found",
				zap.String("session_id", entry.SessionID),
			)
			return
		}

		// Stream log to specific ACP session
		stream := acppkg.NewSessionStream(s.client, sessionID)
		if err := stream.SendText(session.Context, logMsg); err != nil {
			s.logger.Error("failed to stream log to ACP client",
				zap.String("session_id", entry.SessionID),
				zap.Error(err),
			)
		}
		return
	}

	// If no session specified, broadcast to all active sessions
	sessionIDs := s.store.List()
	for _, sessionID := range sessionIDs {
		session, ok := s.store.Get(sessionID)
		if !ok {
			continue
		}

		// Stream log to ACP client
		stream := acppkg.NewSessionStream(s.client, sessionID)
		if err := stream.SendText(session.Context, logMsg); err != nil {
			s.logger.Error("failed to stream log to ACP client",
				zap.String("session_id", string(sessionID)),
				zap.Error(err),
			)
		}
	}
}

// OnAgentLifecycle receives agent registration/removal events and notifies all active ACP sessions
func (s *acpServiceImpl) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
	s.logger.Debug("agent lifecycle event",
		zap.String("agent_id", event.AgentID.String()),
		zap.String("role", event.Role),
		zap.Bool("added", event.Added),
	)

	// Notify all active ACP sessions of agent changes
	// Format: Agent {added|removed}: {role} (id: {agent_id})
	var action string
	if event.Added {
		action = "added"
	} else {
		action = "removed"
	}
	notification := fmt.Sprintf("Agent %s: %s (id: %s)", action, event.Role, event.AgentID.String())

	// Get all active sessions and send the notification
	sessionIDs := s.store.List()
	for _, sessionID := range sessionIDs {
		session, ok := s.store.Get(sessionID)
		if !ok {
			continue
		}

		// Stream notification to ACP client
		stream := acppkg.NewSessionStream(s.client, sessionID)
		if err := stream.SendText(session.Context, notification); err != nil {
			s.logger.Error("failed to send agent lifecycle notification to ACP client",
				zap.String("session_id", string(sessionID)),
				zap.Error(err),
			)
		}
	}
}

// =============================================================================
// ACP Protocol Implementation
// =============================================================================

// Initialize implements acp.Agent.Initialize
func (s *acpServiceImpl) Initialize(ctx context.Context, params *acppkg.InitializeRequest) (*acppkg.InitializeResponse, error) {
	return &acppkg.InitializeResponse{
		ProtocolVersion: acppkg.ProtocolVersion(acppkg.CurrentProtocolVersion),
		AgentCapabilities: &acppkg.AgentCapabilities{
			LoadSession: false,
			MCPCapabilities: &acppkg.MCPCapabilities{
				HTTP: false,
				SSE:  false,
			},
			PromptCapabilities: &acppkg.PromptCapabilities{
				Audio:           false,
				EmbeddedContext: false,
				Image:           false,
			},
		},
		AuthMethods: []acppkg.AuthMethod{},
	}, nil
}

// Authenticate implements acp.Agent.Authenticate
// TODO: Implement proper authentication in Task 6
func (s *acpServiceImpl) Authenticate(ctx context.Context, params *acppkg.AuthenticateRequest) (*acppkg.AuthenticateResponse, error) {
	s.logger.Debug("authenticate request", zap.String("method_id", string(params.MethodID)))
	// Return nil to indicate no authentication required
	return nil, nil
}

// SetSessionMode implements acp.Agent.SetSessionMode
// TODO: Implement session mode handling
func (s *acpServiceImpl) SetSessionMode(ctx context.Context, params *acppkg.SetSessionModeRequest) (*acppkg.SetSessionModeResponse, error) {
	s.logger.Debug("ACP set session mode",
		zap.String("channel_id", s.id.String()),
		zap.String("mode_id", string(params.ModeID)),
		zap.Any("parameters", params.Meta),
	)
	return nil, nil
}

// SetSessionConfigOption implements acp.Agent.SetSessionConfigOption
// TODO: Implement session config option handling
func (s *acpServiceImpl) SetSessionConfigOption(ctx context.Context, params *acppkg.SetSessionConfigOptionRequest) (*acppkg.SetSessionConfigOptionResponse, error) {
	s.logger.Debug("ACP set session config option",
		zap.String("channel_id", s.id.String()),
		zap.String("config_id", string(params.ConfigID)),
		zap.String("value", string(params.Value)),
	)
	return nil, nil
}

// Prompt implements acp.Agent.Prompt - core agent execution loop
func (s *acpServiceImpl) Prompt(ctx context.Context, params *acppkg.PromptRequest) (*acppkg.PromptResponse, error) {
	session, ok := s.store.Get(params.SessionID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", params.SessionID)
	}

	// Cancel previous turn and create new context
	session.CancelFunc()
	sessionCtx, cancelFunc := context.WithCancel(context.Background())
	session.Context = sessionCtx
	session.CancelFunc = cancelFunc

	// Extract prompt content from params
	if len(params.Prompt) == 0 {
		return nil, fmt.Errorf("prompt content is empty")
	}
	contentBlock := params.Prompt[0]
	textContent, ok := contentBlock.AsText()
	if !ok {
		return nil, fmt.Errorf("prompt content is not text")
	}
	promptContent := textContent.Text

	// Submit to agent via channel facade using our channel ID
	result, err := s.facade.SubmitInput(sessionCtx, s.id, string(params.SessionID), promptContent)
	if err != nil {
		if sessionCtx.Err() == context.Canceled {
			return &acppkg.PromptResponse{
				StopReason: acppkg.StopReasonCancelled,
			}, nil
		}
		return nil, err
	}

	// Send final result back to client via stream
	stream := acppkg.NewSessionStream(s.client, params.SessionID)
	if result.Response != "" {
		if err := stream.SendText(sessionCtx, result.Response); err != nil {
			return nil, fmt.Errorf("failed to send response: %w", err)
		}
	}

	return &acppkg.PromptResponse{
		StopReason: acppkg.StopReasonEndTurn,
	}, nil
}

// Cancel implements acp.Agent.Cancel
// Cancels the in-flight prompt for the given session
func (s *acpServiceImpl) Cancel(ctx context.Context, params *acppkg.CancelNotification) error {
	s.logger.Debug("ACP cancel request",
		zap.String("channel_id", s.id.String()),
		zap.String("session_id", string(params.SessionID)),
	)

	// Get session from store
	session, ok := s.store.Get(params.SessionID)
	if !ok {
		return fmt.Errorf("session %s not found", params.SessionID)
	}

	// Cancel the session context (this will cancel the in-flight prompt)
	session.CancelFunc()

	// Also notify facade about the cancellation via session ID
	if err := s.facade.CancelInput(string(params.SessionID)); err != nil {
		s.logger.Warn("failed to cancel input in facade",
			zap.String("channel_id", s.id.String()),
			zap.String("session_id", string(params.SessionID)),
			zap.Error(err),
		)
	}

	return nil
}
