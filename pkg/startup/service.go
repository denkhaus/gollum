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

// Compile-time interface check
var _ StartupContextService = (*startupContextServiceImpl)(nil)

// NewStartupContextService creates a new StartupContextService.
func NewStartupContextService() StartupContextService {
	return &startupContextServiceImpl{
		outputs: make(map[string]any),
	}
}

func (p *startupContextServiceImpl) SetContext(outputs map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create defensive copy to prevent caller from modifying the map
	p.outputs = make(map[string]any, len(outputs))
	for k, v := range outputs {
		p.outputs[k] = v
	}
}

func (p *startupContextServiceImpl) GetContextText() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.outputs) == 0 {
		return ""
	}

	// If there's a dedicated "context" or "text" field, use it directly
	if text, ok := p.outputs["context"].(string); ok && text != "" {
		return text
	}
	if text, ok := p.outputs["text"].(string); ok && text != "" {
		return text
	}

	// Otherwise, format all key-value pairs
	return p.formatOutputs()
}

func (p *startupContextServiceImpl) HasContent() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.outputs) == 0 {
		return false
	}

	// Check if there's actual content (not just empty values)
	for _, v := range p.outputs {
		if str, ok := v.(string); ok && str != "" {
			return true
		}
		if v != nil {
			return true
		}
	}
	return false
}

func (p *startupContextServiceImpl) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputs = make(map[string]any)
}

func (p *startupContextServiceImpl) formatOutputs() string {
	var result strings.Builder
	for key, value := range p.outputs {
		result.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}
	return result.String()
}
