// Package middleware provides display and UI-related middleware for agent interactions.
package middleware

import (
	"context"
	"strings"

	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/ui"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// SummaryMiddleware provides only final output (no intermediate steps)
	SummaryMiddleware struct {
		messenger  ui.AgentMessenger
		registry   registry.AgentRegistry
		agentID    uuid.UUID
		agentRole  string
		buffer     strings.Builder
		hasContent bool
	}

	// SummaryMiddlewareProvider creates summary middleware instances via DI
	SummaryMiddlewareProvider interface {
		CreateSummaryMiddleware(agentID uuid.UUID, agentRole string) *SummaryMiddleware
	}

	summaryMiddlewareProvider struct {
		messenger ui.AgentMessenger
		registry  registry.AgentRegistry
	}
)

// NewSummaryMiddlewareProvider creates a provider for summary middleware
func NewSummaryMiddlewareProvider(injector do.Injector) (SummaryMiddlewareProvider, error) {
	messenger := do.MustInvoke[ui.AgentMessenger](injector)
	registry := do.MustInvoke[registry.AgentRegistry](injector)

	return &summaryMiddlewareProvider{
		messenger: messenger,
		registry:  registry,
	}, nil
}

// CreateSummaryMiddleware creates a new summary middleware for a specific agent
func (p *summaryMiddlewareProvider) CreateSummaryMiddleware(
	agentID uuid.UUID,
	agentRole string,
) *SummaryMiddleware {
	return &SummaryMiddleware{
		messenger: p.messenger,
		registry:  p.registry,
		agentID:   agentID,
		agentRole: agentRole,
	}
}

// ContentBlockMiddleware buffers all content and shows only the final result
func (s *SummaryMiddleware) ContentBlockMiddleware(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
	return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		// Buffer all text content (don't display yet)
		if resp != nil && len(resp.Texts) > 0 {
			for _, text := range resp.Texts {
				if text != "" {
					s.buffer.WriteString(text)
					s.hasContent = true
				}
			}
		}

		return resp, nil
	}
}

// Flush displays the buffered content (call this when agent completes)
func (s *SummaryMiddleware) Flush() {
	if s.hasContent {
		s.messenger.DisplayAgentMessage(s.agentID, s.agentRole, s.buffer.String(), false)
	}
}

// Reset clears the buffer
func (s *SummaryMiddleware) Reset() {
	s.buffer.Reset()
	s.hasContent = false
}

// ToolMiddleware is not implemented for summary mode (suppresses tool output)
func (s *SummaryMiddleware) ToolMiddleware(next gollem.ToolHandler) gollem.ToolHandler {
	// Pass through without displaying anything
	return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return next(ctx, req)
	}
}

// DisplayWelcome is not implemented for summary mode
func (s *SummaryMiddleware) DisplayWelcome() {
	// Suppress welcome in summary mode
}

// DisplaySystemInfo is not implemented for summary mode
func (s *SummaryMiddleware) DisplaySystemInfo(_ string) {
	// Suppress system info in summary mode
}
