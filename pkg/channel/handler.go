// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// InputHandler handles user input processing including command execution and supervisor interaction.
type InputHandler interface {
	// HandleInput processes user input and returns the result.
	// It first checks if the input is a slash command, and if so, executes it.
	// Otherwise, it creates/gets a session and supervisor, then executes the input.
	HandleInput(ctx context.Context, sessionCtx *shared.SessionContext, input string) (*InputResult, error)

	// CancelInput cancels an in-flight input for the given session.
	CancelInput(sessionID uuid.UUID) error
}

// inputHandlerImpl implements InputHandler interface.
type inputHandlerImpl struct {
	commandManager command.Manager
	sessionManager session.SessionManager
	agentFactory   shared.AgentFactory
	logger         logger.LoggerService
}

// NewInputHandler creates a new InputHandler instance.
func NewInputHandler(cm command.Manager, sm session.SessionManager, af shared.AgentFactory, log logger.LoggerService) InputHandler {
	return &inputHandlerImpl{
		commandManager: cm,
		sessionManager: sm,
		agentFactory:   af,
		logger:         log,
	}
}

// HandleInput processes user input and returns the result.
func (h *inputHandlerImpl) HandleInput(ctx context.Context, sessionCtx *shared.SessionContext, input string) (*InputResult, error) {
	// First check if it's a slash command
	handled, response, err := h.commandManager.Execute(ctx, sessionCtx.SessionID, input)
	if handled {
		return &InputResult{
			Handled:   true,
			IsCommand: true,
			Response:  response,
			Error:     err,
		}, nil
	}

	// Get or create session for this interaction
	sess, err := h.sessionManager.GetOrCreateSession(sessionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create session: %w", err)
	}

	// Get or create supervisor for this session (lazy, thread-safe)
	supervisor, err := sess.GetOrCreateSupervisor(h.agentFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create supervisor: %w", err)
	}

	// Execute supervisor agent with session context
	resp, err := supervisor.Execute(sess.Context, gollem.Text(input))
	if err != nil {
		return &InputResult{
			Handled: true,
			Error:   err,
		}, nil
	}

	// Extract response
	var content string
	if resp != nil && len(resp.Texts) > 0 {
		content = strings.Join(resp.Texts, "\n")
	}

	return &InputResult{
		Handled:  true,
		Response: content,
	}, nil
}

// CancelInput cancels an in-flight input for the given session.
func (h *inputHandlerImpl) CancelInput(sessionID uuid.UUID) error {
	session, ok := h.sessionManager.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID.String())
	}
	session.CancelFunc()
	return nil
}

// Ensure inputHandlerImpl implements InputHandler at compile time
var _ InputHandler = (*inputHandlerImpl)(nil)
