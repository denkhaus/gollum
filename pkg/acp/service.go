package acp

import (
	"context"
	"fmt"

	acppkg "github.com/ironpark/go-acp"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
)

// Service defines the public interface for ACP agent operations
type Service interface {
	// acp.Agent interface methods
	Initialize(ctx context.Context, params *acppkg.InitializeRequest) (*acppkg.InitializeResponse, error)
	Authenticate(ctx context.Context, params *acppkg.AuthenticateRequest) (*acppkg.AuthenticateResponse, error)
	SetSessionMode(ctx context.Context, params *acppkg.SetSessionModeRequest) (*acppkg.SetSessionModeResponse, error)
	SetSessionConfigOption(ctx context.Context, params *acppkg.SetSessionConfigOptionRequest) (*acppkg.SetSessionConfigOptionResponse, error)
	Prompt(ctx context.Context, params *acppkg.PromptRequest) (*acppkg.PromptResponse, error)
	Cancel(ctx context.Context, params *acppkg.CancelNotification) error

	// Dependency injection setters (called by connection factory)
	SetClient(client acppkg.Client)
	SetSessionStore(store acppkg.SessionStore[*AcpSession])
}

// acpServiceImpl implements Service (PRIVATE)
type acpServiceImpl struct {
	facade channel.ChannelFacade
	logger logger.LoggerService
	client acppkg.Client
	store  acppkg.SessionStore[*AcpSession]
}

// Ensure acpServiceImpl implements Service at compile time
var _ Service = (*acpServiceImpl)(nil)

// NewAcpService creates a new ACP service with DI
func NewAcpService(injector do.Injector) (Service, error) {
	logger := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	logger.Debug("startup ACP service")

	return &acpServiceImpl{
		logger: logger,
		facade:  facade,
	}, nil
}

// SetClient sets the ACP client (called by connection factory)
func (s *acpServiceImpl) SetClient(client acppkg.Client) {
	s.client = client
}

// SetSessionStore sets the session store (called by connection factory)
func (s *acpServiceImpl) SetSessionStore(store acppkg.SessionStore[*AcpSession]) {
	s.store = store
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
	s.logger.Debug("set session mode", zap.String("mode_id", string(params.ModeID)))
	return nil, nil
}

// SetSessionConfigOption implements acp.Agent.SetSessionConfigOption
// TODO: Implement session config option handling
func (s *acpServiceImpl) SetSessionConfigOption(ctx context.Context, params *acppkg.SetSessionConfigOptionRequest) (*acppkg.SetSessionConfigOptionResponse, error) {
	s.logger.Debug("set session config",
		zap.String("config_id", string(params.ConfigID)),
		zap.String("value", string(params.Value)))
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

	// Submit to agent via channel facade
	result, err := s.facade.SubmitInput(sessionCtx, uuid.Nil, promptContent)
	if err != nil {
		if sessionCtx.Err() == context.Canceled {
			return &acppkg.PromptResponse{
				StopReason: acppkg.StopReasonCancelled,
			}, nil
		}
		return nil, err
	}

	// Send result back to client
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
// TODO: Implement prompt cancellation
func (s *acpServiceImpl) Cancel(ctx context.Context, params *acppkg.CancelNotification) error {
	s.logger.Debug("cancel request", zap.String("session_id", string(params.SessionID)))
	return nil
}
