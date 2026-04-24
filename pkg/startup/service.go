// Package startup provides startup context management for flows.
package startup

import (
	"fmt"
	"strings"
	"sync"
)

// StartupContextService stores and provides startup flow context.
type StartupContextService interface {
	// SetContext stores the startup flow outputs.
	SetContext(outputs map[string]any)

	// GetContextText returns formatted context for prompts.
	GetContextText() string

	// HasContent returns true if startup context is available.
	HasContent() bool

	// Clear removes stored context (for testing).
	Clear()
}

type startupContextServiceImpl struct {
	outputs map[string]any
	mu      sync.RWMutex
}

// NewStartupContextService creates a new StartupContextService.
func NewStartupContextService() StartupContextService {
	return &startupContextServiceImpl{
		outputs: make(map[string]any),
	}
}

func (s *startupContextServiceImpl) SetContext(outputs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputs = outputs
}

func (s *startupContextServiceImpl) GetContextText() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.outputs) == 0 {
		return ""
	}

	// If there's a dedicated "context" or "text" field, use it directly
	if text, ok := s.outputs["context"].(string); ok && text != "" {
		return text
	}
	if text, ok := s.outputs["text"].(string); ok && text != "" {
		return text
	}

	// Otherwise, format all key-value pairs
	return s.formatOutputs()
}

func (s *startupContextServiceImpl) HasContent() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.outputs) == 0 {
		return false
	}

	// Check if there's actual content (not just empty values)
	for _, v := range s.outputs {
		if str, ok := v.(string); ok && str != "" {
			return true
		}
		if v != nil {
			return true
		}
	}
	return false
}

func (s *startupContextServiceImpl) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputs = make(map[string]any)
}

func (s *startupContextServiceImpl) formatOutputs() string {
	var result strings.Builder
	for key, value := range s.outputs {
		result.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}
	return result.String()
}
