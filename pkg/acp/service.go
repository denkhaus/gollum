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
// - The connection creates ONE session that lives for the entire process lifetime
// - Session is created by the ACP framework via NewSession callback (in connection.go)
// - TODO: Determine when NewSession callback fires (Initialize/SetSessionMode/Prompt?)
// - Session is tracked as "activeSession" since there's only one per connection
// - All messages received via OnMessage() are destined for this single active session
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
	"sync"

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

	// activeSession tracks the current ACP session (one per connection)
	activeSessionMu sync.RWMutex
	activeSession   *shared.ACPSession
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
	s.activeSessionMu.RLock()
	session := s.activeSession
	s.activeSessionMu.RUnlock()

	if session == nil {
		s.logger.Warn("received message but no active session",
			zap.String("channel_id", s.id.String()),
			zap.String("message_content", msg.Content),
		)
		return
	}

	// Stream message to ACP client
	stream := acppkg.NewSessionStream(s.client, session.SessionID)
	if err := stream.SendText(session.Context, msg.Content); err != nil {
		s.logger.Error("failed to stream message to ACP client",
			zap.String("session_id", string(session.SessionID)),
			zap.Error(err),
		)
	}
}

// OnLog receives log entries from the agent system
// Currently not forwarded to ACP client, but logged locally
func (s *acpServiceImpl) OnLog(entry channel.LogEntry) {
	s.logger.Debug("log entry from agent system",
		zap.String("level", entry.Level),
		zap.String("message", entry.Message),
	)
	// TODO: Optionally forward logs to ACP client if needed
}

// OnAgentLifecycle receives agent registration/removal events
func (s *acpServiceImpl) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
	s.logger.Debug("agent lifecycle event",
		zap.String("agent_id", event.AgentID.String()),
		zap.String("role", event.Role),
		zap.Bool("added", event.Added),
	)
	// TODO: Optionally notify ACP client of agent changes
}

// =============================================================================
// Session Management Helpers
// =============================================================================

// setActiveSession sets the given session as active for OnMessage callbacks
func (s *acpServiceImpl) setActiveSession(session *shared.ACPSession) {
	s.activeSessionMu.Lock()
	s.activeSession = session
	s.activeSessionMu.Unlock()
}

// clearActiveSession clears the active session
func (s *acpServiceImpl) clearActiveSession() {
	s.activeSessionMu.Lock()
	s.activeSession = nil
	s.activeSessionMu.Unlock()
}

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

	// Set as active session for OnMessage callbacks
	s.setActiveSession(session)
	defer s.clearActiveSession()

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

	// Also notify facade about the cancellation (for future tracking)
	if err := s.facade.CancelInput(s.id); err != nil {
		s.logger.Warn("failed to cancel input in facade",
			zap.String("channel_id", s.id.String()),
			zap.Error(err),
		)
	}

	return nil
}
