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
//  1. Initialize() - Protocol handshake, NO session created yet
//  2. First Prompt() call for a new session - Triggers NewSession callback
//  3. NewSession callback (in connection.go) creates the session BEFORE Prompt() executes
//  4. Subsequent Prompt() calls reuse existing session (identified by SessionID)
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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
)

// acpServiceImpl implements Service and channel.Channel (PRIVATE)
type acpServiceImpl struct {
	facade         channel.ChannelFacade
	logger         logger.LoggerService
	config         config.ConfigService
	client         acppkg.Client
	store          acppkg.SessionStore[*shared.Session] // ACP-internal store for protocol
	sessionManager session.SessionManager               // Gollum session manager for system integration
	id             uuid.UUID                            // Channel ID
	conn           Connection                           // ACP connection (created in Start)
	injector       do.Injector                          // For connection creation

	// Transport-related fields
	stdin         io.Reader     // For connection creation
	stdout        io.Writer     // For connection creation
	transportType TransportType // Transport type (stdio, http)
	host          string        // Host address for HTTP transport
	port          int           // Port number for HTTP transport
}

// Ensure acpServiceImpl implements required interfaces at compile time
var _ shared.ACPService = (*acpServiceImpl)(nil)
var _ channel.Channel = (*acpServiceImpl)(nil)
var _ acppkg.ExtMethodHandler = (*acpServiceImpl)(nil)

// NewACPService creates a new ACP service with DI
func NewACPService(injector do.Injector) (shared.ACPService, error) {
	logger := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[channel.ChannelFacade](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	// Use Invoke instead of MustInvoke to get error details
	sessionMgr, err := do.Invoke[session.SessionManager](injector)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke SessionManager: %w", err)
	}
	if sessionMgr == nil {
		return nil, fmt.Errorf("SessionManager is nil after DI invocation (provider bug?)")
	}

	logger.Debug("ACP service initialized",
		zap.Bool("sessionManager_set", sessionMgr != nil),
	)

	// Generate unique channel ID for this ACP service instance
	id := uuid.New()

	// Get ACP configuration with defaults
	acpConfig := cfg.GetACPConfig()

	// Parse transport type from config (default to stdio)
	transportType := TransportStdio
	if acpConfig.TransportType != "" {
		parsedType, err := ParseTransportType(acpConfig.TransportType)
		if err != nil {
			logger.Warn("invalid transport type in config, using stdio",
				zap.String("transport_type", acpConfig.TransportType),
				zap.Error(err))
		} else {
			transportType = parsedType
		}
	}

	// Use config defaults for host and port
	host := acpConfig.Host
	port := acpConfig.Port

	svc := &acpServiceImpl{
		logger:         logger,
		facade:         facade,
		config:         cfg,
		sessionManager: sessionMgr,
		id:             id,
		injector:       injector,
		transportType:  transportType,
		host:           host,
		port:           port,
	}

	logger.Debug("ACP service created",
		zap.String("channel_id", svc.id.String()),
		zap.Bool("sessionManager_set", svc.sessionManager != nil),
		zap.Bool("sessionMgr_equals_nil", sessionMgr == nil),
	)

	return svc, nil
}

// SetClient sets the ACP client (called by connection factory)
func (s *acpServiceImpl) SetClient(client acppkg.Client) {
	s.client = client
}

// SetSessionStore sets the session store (called by connection factory)
func (s *acpServiceImpl) SetSessionStore(store acppkg.SessionStore[*shared.Session]) {
	s.store = store
}

// =============================================================================
// Channel Interface Implementation
// =============================================================================

// ID returns the unique channel identifier for this ACP service
func (s *acpServiceImpl) ID() uuid.UUID {
	return s.id
}

// Start begins the ACP channel's lifecycle by creating and starting the ACP connection.
func (s *acpServiceImpl) Start(ctx context.Context) error {
	if s.transportType == TransportHTTP {
		return s.startHTTP(ctx)
	}
	return s.startStdio(ctx)
}

// startStdio creates and starts a stdio-based ACP connection
func (s *acpServiceImpl) startStdio(ctx context.Context) error {
	if s.stdin == nil || s.stdout == nil {
		return fmt.Errorf("ACP channel: stdin and stdout must be provided via options")
	}

	conn, err := s.newConnection(nil)
	if err != nil {
		return fmt.Errorf("failed to create ACP connection: %w", err)
	}

	s.conn = conn

	// Start the connection (this blocks until the ACP client disconnects)
	if err := s.conn.Start(ctx); err != nil {
		return fmt.Errorf("ACP connection failed: %w", err)
	}

	return nil
}

// startHTTP creates an HTTP-based ACP connection
func (s *acpServiceImpl) startHTTP(ctx context.Context) error {
	// If connection already exists, return success
	if s.conn != nil {
		return nil
	}

	// Create HTTP connection (will create its own transport internally)
	conn, err := s.newConnection(nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP connection: %w", err)
	}

	s.conn = conn

	// Note: We do NOT call conn.Start() here because for HTTP mode,
	// go-acp's conn.Start() blocks until HTTP server is running.
	// The ServeMux from the handler will automatically start processing
	// requests when the HTTP server begins serving.

	return nil
}

// GetHandler returns the HTTP handler for HTTP transport mode.
//
// Returns nil if:
//   - Start() has not been called yet
//   - Transport type is not HTTP (e.g., stdio mode)
//
// The handler is ready to use with http.Server after Start() completes
// successfully in HTTP transport mode.
func (s *acpServiceImpl) GetHandler() http.Handler {
	if s.conn == nil {
		return nil
	}
	return s.conn.Handler()
}

// OnMessage receives messages from the agent system and streams them to the ACP client
func (s *acpServiceImpl) OnMessage(msg shared.Message) {
	if s.store == nil {
		s.logger.Warn("store not initialized in OnMessage",
			zap.String("channel_id", s.id.String()),
			zap.String("session_id", msg.SessionID.String()),
		)
		return
	}

	// Convert uuid.UUID SessionID to acppkg.SessionID (string)
	sessionID := acppkg.SessionID(msg.SessionID.String())

	// Get session from store using SessionID from message
	session, ok := s.store.Get(sessionID)
	if !ok {
		s.logger.Warn("received message but session not found",
			zap.String("channel_id", s.id.String()),
			zap.String("session_id", msg.SessionID.String()),
			zap.String("message_content", msg.Content),
		)
		return
	}

	// Stream message to ACP client
	stream := acppkg.NewSessionStream(s.client, sessionID)
	if err := stream.SendText(session.Context, msg.Content); err != nil {
		s.logger.Error("failed to stream message to ACP client",
			zap.String("session_id", msg.SessionID.String()),
			zap.Error(err),
		)
	}
}

// OnLog receives log entries from the agent system and forwards them to ACP sessions.
//
// Log routing behavior:
// - Session-specific logs (entry.SessionID set) → routed to that session only
// - System-wide logs (no session ID) → broadcast to all active sessions
//
// See pkg/channel package documentation for general log routing patterns.
func (s *acpServiceImpl) OnLog(entry shared.LogEntry) {
	// Log locally for debugging
	s.logger.Debug("log entry from agent system",
		zap.String("level", entry.Level),
		zap.String("message", entry.Message),
		zap.String("context", entry.String()),
	)

	// Format: [LEVEL] message
	logMsg := fmt.Sprintf("[%s] %s", entry.Level, entry.Message)

	// If session is specified, route to that specific session
	if entry.SessionID != uuid.Nil {
		sessionID := acppkg.SessionID(entry.SessionID.String())
		session, ok := s.store.Get(sessionID)
		if !ok {
			s.logger.Warn("log entry specifies session but session not found",
				zap.String("session_id", entry.SessionID.String()),
			)
			return
		}

		// Stream log to specific ACP session
		stream := acppkg.NewSessionStream(s.client, sessionID)
		if err := stream.SendText(session.Context, logMsg); err != nil {
			s.logger.Error("failed to stream log to ACP client",
				zap.String("session_id", entry.SessionID.String()),
				zap.Error(err),
			)
		}
		return
	}

	// If no session specified, broadcast to all active sessions
	if s.store == nil {
		s.logger.Warn("store not initialized in OnLog")
		return
	}

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
	acpConfig := s.config.GetACPConfig()

	// Build auth methods based on configuration
	// Note: Using basic AuthMethod since env_var type is still unstable in go-acp
	var authMethods []acppkg.AuthMethod
	if acpConfig.ExpectedAPIKey != "" {
		// Require API key authentication via environment variable
		authMethods = []acppkg.AuthMethod{
			{
				ID:          "gollum-acp",
				Name:        "Gollum ACP API Key",
				Description: "Set GOLLUM_ACP_API_KEY environment variable when starting the agent",
			},
		}
		s.logger.Debug("ACP authentication required: API key configured")
	} else {
		s.logger.Debug("ACP authentication not required: no API key configured")
	}

	return &acppkg.InitializeResponse{
		ProtocolVersion: acppkg.ProtocolVersion(acppkg.CurrentProtocolVersion),
		AgentCapabilities: &acppkg.AgentCapabilities{
			// Session management capabilities
			SessionCapabilities: &acppkg.SessionCapabilities{
				List: &acppkg.SessionListCapabilities{}, // We support listing sessions
				// TODO: Add Close, Fork, Resume when they move to stable schema
				// Currently implemented but not advertised:
				// - LoadSession: true (database-backed sessions)
				// - CloseSession, ResumeSession, ForkSession (via SessionManager)
			},
			// Session loading is fully implemented with database persistence
			LoadSession: true,
			// MCP support (Model Context Protocol - for external tools)
			MCPCapabilities: &acppkg.MCPCapabilities{
				HTTP: false, // Not yet implemented
				SSE:  false, // Not yet implemented
			},
			// Prompt capabilities
			PromptCapabilities: &acppkg.PromptCapabilities{
				Audio:           false, // Not yet implemented
				EmbeddedContext: false, // Not yet implemented
				Image:           false, // Not yet implemented
			},
		},
		AuthMethods: authMethods,
	}, nil
}

// Authenticate implements acp.Agent.Authenticate
// TODO: The current implementation does only work with stdio mode. Make sure http mode is supported.
func (s *acpServiceImpl) Authenticate(ctx context.Context, params *acppkg.AuthenticateRequest) (*acppkg.AuthenticateResponse, error) {
	acpConfig := s.config.GetACPConfig()

	// If no expected key is configured, no authentication required
	if acpConfig.ExpectedAPIKey == "" {
		s.logger.Debug("ACP authentication skipped: no API key configured")
		return &acppkg.AuthenticateResponse{}, nil
	}

	// For env_var auth type per ACP RFC:
	// Client sets GOLLUM_ACP_API_KEY when starting the agent process.
	// Config service reads this into ProvidedAPIKey via envconfig.
	// We validate the provided key matches the expected key.
	if acpConfig.ProvidedAPIKey == "" {
		s.logger.Warn("ACP authentication failed: GOLLUM_ACP_API_KEY not set")
		return nil, fmt.Errorf("authentication failed: GOLLUM_ACP_API_KEY environment variable not set")
	}

	if acpConfig.ProvidedAPIKey != acpConfig.ExpectedAPIKey {
		s.logger.Warn("ACP authentication failed: invalid API key")
		return nil, fmt.Errorf("authentication failed: invalid API key")
	}

	s.logger.Debug("ACP authentication successful")
	return &acppkg.AuthenticateResponse{}, nil
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
func (s *acpServiceImpl) Prompt(ctx context.Context, params *acppkg.PromptRequest) (resp *acppkg.PromptResponse, err error) {
	// Declare session variable here so defer block can access it for nil checking
	var session *shared.Session

	// Add recover to catch panics and provide better error messages
	defer func() {
		if r := recover(); r != nil {
			// Capture stack trace
			stackTrace := debug.Stack()

			if s.logger != nil {
				s.logger.Error("ACP Prompt panic recovered",
					zap.Any("panic", r),
					zap.String("session_id", string(params.SessionID)),
					zap.String("stacktrace", string(stackTrace)),
				)
			}

			err = fmt.Errorf("panic in Prompt: %v\n%s", r, string(stackTrace))
			resp = &acppkg.PromptResponse{
				StopReason: acppkg.StopReasonEndTurn,
			}
		}
	}()

	// Validate required dependencies
	if s.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if s.facade == nil {
		return nil, fmt.Errorf("facade not initialized")
	}

	session, ok := s.store.Get(params.SessionID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", params.SessionID)
	}

	sessionCtx := session.NewTurn(ctx)

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

	// Submit to agent via channel facade using SessionContext
	if session.SessionID == uuid.Nil {
		return nil, fmt.Errorf("session.SessionID is nil/empty")
	}
	if s.id == uuid.Nil {
		return nil, fmt.Errorf("service id (channelID) is nil/empty")
	}

	// Check if Cwd is empty
	if session.Cwd == "" {
		return nil, fmt.Errorf("session.Cwd is empty")
	}

	s.logger.Info("ACP Prompt submitting to facade",
		zap.String("session_context", session.String()),
		zap.String("prompt_preview", promptContent[:min(100, len(promptContent))]+"..."),
	)

	result, err := s.facade.SubmitInput(session, promptContent)
	if err != nil {
		if sessionCtx.Err() == context.Canceled {
			return &acppkg.PromptResponse{
				StopReason: acppkg.StopReasonCancelled,
			}, nil
		}
		return nil, err
	}

	// Check result before accessing
	if result == nil {
		return &acppkg.PromptResponse{
			StopReason: acppkg.StopReasonEndTurn,
		}, nil
	}

	// Send final result back to client via stream
	stream := acppkg.NewSessionStream(s.client, params.SessionID)
	if stream == nil {
		return nil, fmt.Errorf("failed to create session stream")
	}
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
	if err := s.facade.CancelInput(session.SessionID); err != nil {
		s.logger.Warn("failed to cancel input in facade",
			zap.String("channel_id", s.id.String()),
			zap.String("session_id", session.SessionID.String()),
			zap.Error(err),
		)
	}

	return nil
}

// ListSessions implements ACP session/list protocol method.
func (s *acpServiceImpl) ListSessions(ctx context.Context) ([]*shared.Session, error) {
	return s.sessionManager.ListSessions(ctx)
}

// LoadSession loads an existing session (ACP: session/load).
func (s *acpServiceImpl) LoadSession(ctx context.Context, sessionID acppkg.SessionID) (*shared.Session, error) {
	uuidSessionID, err := uuid.Parse(string(sessionID))
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	session, err := s.sessionManager.LoadSession(ctx, uuidSessionID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// ResumeSession resumes a closed session (ACP: session/resume).
func (s *acpServiceImpl) ResumeSession(ctx context.Context, sessionID acppkg.SessionID) error {
	uuidSessionID, err := uuid.Parse(string(sessionID))
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	return s.sessionManager.ResumeSession(ctx, uuidSessionID)
}

// CloseSession closes a session (ACP: session/close).
func (s *acpServiceImpl) CloseSession(ctx context.Context, sessionID acppkg.SessionID) error {
	uuidSessionID, err := uuid.Parse(string(sessionID))
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	return s.sessionManager.CloseSession(uuidSessionID)
}

// ForkSession creates a copy of a session (ACP: session/fork).
func (s *acpServiceImpl) ForkSession(ctx context.Context, sessionID acppkg.SessionID) (acppkg.SessionID, error) {
	uuidSessionID, err := uuid.Parse(string(sessionID))
	if err != nil {
		return "", fmt.Errorf("invalid session ID: %w", err)
	}

	newSession, err := s.sessionManager.ForkSession(ctx, uuidSessionID)
	if err != nil {
		return "", err
	}

	return acppkg.SessionID(newSession.SessionID.String()), nil
}

// =============================================================================
// ACP Extension Method Handler
// =============================================================================

// ExtMethod implements acp.ExtMethodHandler to expose custom JSON-RPC methods
// for session lifecycle management.
//
// Extension methods are prefixed with underscore as per ACP convention.
//
// Supported methods:
//   - "_session/list"   - List all sessions
//   - "_session/load"   - Load a session by ID
//   - "_session/resume" - Resume a closed session
//   - "_session/close"  - Close a session
//   - "_session/fork"   - Fork a session
func (s *acpServiceImpl) ExtMethod(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "_session/list":
		return s.handleListSessions(ctx, params)

	case "_session/load":
		return s.handleLoadSession(ctx, params)

	case "_session/resume":
		return s.handleResumeSession(ctx, params)

	case "_session/close":
		return s.handleCloseSession(ctx, params)

	case "_session/fork":
		return s.handleForkSession(ctx, params)

	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

// handleListSessions handles the _session/list extension method.
func (s *acpServiceImpl) handleListSessions(ctx context.Context, params json.RawMessage) (any, error) {
	// Parse parameters (empty for list)
	var p struct{}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if s.sessionManager == nil {
		return nil, fmt.Errorf("sessionManager not initialized")
	}

	sessions, err := s.sessionManager.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Convert to response format
	type sessionInfo struct {
		ID        string `json:"id"`
		ChannelID string `json:"channel_id"`
		CreatedAt string `json:"created_at"`
		Cwd       string `json:"cwd"`
	}

	result := make([]any, len(sessions))
	for i, sess := range sessions {
		result[i] = sessionInfo{
			ID:        sess.SessionID.String(),
			ChannelID: sess.ChannelID.String(),
			CreatedAt: sess.CreatedAt.Format(time.RFC3339),
			Cwd:       sess.Cwd,
		}
	}

	return map[string]any{"sessions": result}, nil
}

// handleLoadSession handles the _session/load extension method.
func (s *acpServiceImpl) handleLoadSession(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	acpSession, err := s.LoadSession(ctx, acppkg.SessionID(p.SessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	return map[string]any{
		"session": map[string]any{
			"id":     acpSession.SessionID.String(),
			"cwd":    acpSession.Cwd,
			"loaded": true,
		},
	}, nil
}

// handleResumeSession handles the _session/resume extension method.
func (s *acpServiceImpl) handleResumeSession(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	if err := s.ResumeSession(ctx, acppkg.SessionID(p.SessionID)); err != nil {
		return nil, fmt.Errorf("failed to resume session: %w", err)
	}

	return map[string]any{
		"session_id": p.SessionID,
		"resumed":    true,
	}, nil
}

// handleCloseSession handles the _session/close extension method.
func (s *acpServiceImpl) handleCloseSession(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	if err := s.CloseSession(ctx, acppkg.SessionID(p.SessionID)); err != nil {
		return nil, fmt.Errorf("failed to close session: %w", err)
	}

	return map[string]any{
		"session_id": p.SessionID,
		"closed":     true,
	}, nil
}

// handleForkSession handles the _session/fork extension method.
func (s *acpServiceImpl) handleForkSession(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	newSessionID, err := s.ForkSession(ctx, acppkg.SessionID(p.SessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to fork session: %w", err)
	}

	return map[string]any{
		"original_session_id": p.SessionID,
		"new_session_id":      string(newSessionID),
		"forked":              true,
	}, nil
}

// =============================================================================
// ACP Response Builders - Helper functions for all possible callbacks
// =============================================================================

// buildNewSessionResponse creates a complete NewSessionResponse with all metadata.
// This prepares all possible session information even if not yet fully implemented.
func (s *acpServiceImpl) buildNewSessionResponse(sessionID acppkg.SessionID, cwd string) *acppkg.NewSessionResponse {
	return &acppkg.NewSessionResponse{
		SessionID: sessionID,
		// TODO: Implement models configuration (UNSTABLE in go-acp)
		// Models: s.buildSessionModels(),
		// TODO: Implement modes configuration
		Modes: s.buildSessionModes(),
		// TODO: Implement config options from Gollum configuration
		ConfigOptions: s.buildConfigOptions(),
	}
}

// buildSessionModes creates the session mode state with available permission modes.
// These map to Gollum's permission/execution modes.
func (s *acpServiceImpl) buildSessionModes() *acppkg.SessionModeState {
	modes := []acppkg.SessionMode{
		{
			ID:          acppkg.SessionModeID("default"),
			Name:        "Default",
			Description: "Standard behavior, prompts for dangerous operations",
		},
		{
			ID:          acppkg.SessionModeID("auto"),
			Name:        "Auto",
			Description: "Automatically approve safe operations",
		},
	}

	return &acppkg.SessionModeState{
		CurrentModeID:  acppkg.SessionModeID("default"),
		AvailableModes: modes,
	}
}

// buildConfigOptions creates configuration options that the client can modify.
// These map to Gollum's configuration system.
func (s *acpServiceImpl) buildConfigOptions() []acppkg.SessionConfigOption {
	options := []acppkg.SessionConfigOption{
		{
			ID:          acppkg.SessionConfigID("model"),
			Name:        "Model",
			Description: "AI model to use for responses",
			Category: func() *acppkg.SessionConfigOptionCategory {
				c := acppkg.SessionConfigOptionCategoryModel
				return &c
			}(),
			// TODO: Add select options with available models
		},
		{
			ID:          acppkg.SessionConfigID("timeout"),
			Name:        "Timeout",
			Description: "Request timeout in seconds",
			Category: func() *acppkg.SessionConfigOptionCategory {
				c := acppkg.SessionConfigOptionCategory("general")
				return &c
			}(),
		},
	}

	// TODO: Add debug mode option when configuration supports it
	// cfg := s.config.GetACPConfig()
	// if cfg.DebugLogging {
	//     options = append(options, acppkg.SessionConfigOption{
	//         ID:          acppkg.SessionConfigID("debug"),
	//         Name:        "Debug Mode",
	//         Description: "Enable verbose logging for this session",
	//         Category: func() *acppkg.SessionConfigOptionCategory {
	//             c := acppkg.SessionConfigOptionCategory("debug")
	//             return &c
	//         }(),
	//     })
	// }

	return options
}

// sendAvailableCommands sends available commands/skills to the client.
// This is called after session/new to inform the client about available slash commands.
func (s *acpServiceImpl) sendAvailableCommands(ctx context.Context, sessionID acppkg.SessionID) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}

	stream := acppkg.NewSessionStream(s.client, sessionID)

	// TODO: Load available commands from Gollum's command/skill system
	// For now, sending a minimal set to demonstrate the capability
	commands := []acppkg.AvailableCommand{
		{
			Name:        "debug",
			Description: "Enable debug logging for this session",
		},
		{
			Name:        "compact",
			Description: "Free up context by summarizing the conversation",
		},
		{
			Name:        "clear",
			Description: "Start a new session with empty context",
		},
	}

	return stream.SendCommands(ctx, commands)
}

// sendModeUpdate sends a mode change notification to the client.
// Use this when the session mode has changed.
func (s *acpServiceImpl) sendModeUpdate(ctx context.Context, sessionID acppkg.SessionID, modeID acppkg.SessionModeID) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}

	stream := acppkg.NewSessionStream(s.client, sessionID)
	return stream.SendModeUpdate(ctx, modeID)
}

// sendConfigUpdate sends a configuration update notification to the client.
// Use this when configuration options have changed.
func (s *acpServiceImpl) sendConfigUpdate(ctx context.Context, sessionID acppkg.SessionID, options []acppkg.SessionConfigOption) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}

	stream := acppkg.NewSessionStream(s.client, sessionID)
	return stream.SendConfigUpdate(ctx, options)
}

// sendSessionInfo sends session metadata update to the client.
// Use this when session title or other metadata has changed.
func (s *acpServiceImpl) sendSessionInfo(ctx context.Context, sessionID acppkg.SessionID, title, updatedAt string) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}

	stream := acppkg.NewSessionStream(s.client, sessionID)
	return stream.SendSessionInfo(ctx, title, updatedAt)
}
