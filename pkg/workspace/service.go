// Package workspace provides workspace state management
// for the Gollum agent system.
package workspace

import (
	"context"
	"os"
	"sync"

	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// SourceName is the event source identifier for this service
const SourceName = "workspace_service"

// Service manages workspace state during runtime.
// It manages workspace paths and history.
type Service interface {
	// Workspace path management

	GetCurrentWorkspace() string
	GetWorkspaceHistory() []string
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	mu             sync.RWMutex
	currentPath    string
	history        []string
	maxHistorySize int
	eventBus       events.Bus
	logger         logger.LoggerService
}

// Ensure serviceImpl implements Service
var _ Service = (*serviceImpl)(nil)

// NewServiceProvider creates a new workspace service for DI
func NewServiceProvider(injector do.Injector) (Service, error) {
	bus := do.MustInvoke[events.Bus](injector)
	logService := do.MustInvoke[logger.LoggerService](injector)

	service := &serviceImpl{
		currentPath:    "",
		history:        make([]string, 0),
		maxHistorySize: 10, // Keep last 10 workspaces
		eventBus:       bus,
		logger:         logService,
	}

	// Initialize with current working directory
	if wd, err := os.Getwd(); err == nil && wd != "" {
		service.setCurrentWorkspace(wd)
	}

	// Subscribe to events
	if err := service.subscribeToEvents(); err != nil {
		return nil, err
	}

	return service, nil
}

// subscribeToEvents registers event handlers for the workspace service
func (s *serviceImpl) subscribeToEvents() error {
	// Subscribe to directory changes - sync with high priority
	_, err := events.SubscribeTyped[events.DirectoryChangedPayload](s.eventBus, events.EventDirectoryChanged,
		s.handleDirectoryChanged,
		events.WithSync(),
		events.WithPriority(100),
	)
	if err != nil {
		return err
	}

	return nil
}

// handleDirectoryChanged handles directory change events
func (s *serviceImpl) handleDirectoryChanged(ctx context.Context, payload events.DirectoryChangedPayload) error {
	s.logger.Debug("workspace service received directory changed event",
		zap.String("old_path", payload.OldPath),
		zap.String("new_path", payload.NewPath))

	// Update the current workspace path
	if err := s.setCurrentWorkspace(payload.NewPath); err != nil {
		s.logger.Error("failed to set current workspace",
			zap.String("path", payload.NewPath),
			zap.Error(err))
		return err
	}

	s.logger.Info("workspace updated",
		zap.String("old_path", payload.OldPath),
		zap.String("new_path", payload.NewPath))

	return nil
}

// setCurrentWorkspace sets the current workspace path
// only used for internal purposes. The current workspace path
// gets updated by subscription
func (s *serviceImpl) setCurrentWorkspace(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentPath = path
	s.addToHistoryLocked(path)

	return nil
}

// GetCurrentWorkspace returns the current workspace path
func (s *serviceImpl) GetCurrentWorkspace() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentPath
}

// addToHistoryLocked adds to history without locking (internal use)
func (s *serviceImpl) addToHistoryLocked(path string) {
	if path == "" {
		return
	}

	// Remove if already exists
	for i, h := range s.history {
		if h == path {
			s.history = append(s.history[:i], s.history[i+1:]...)
			break
		}
	}

	// Add to front
	s.history = append([]string{path}, s.history...)

	// Trim to max size
	if len(s.history) > s.maxHistorySize {
		s.history = s.history[:s.maxHistorySize]
	}
}

// GetWorkspaceHistory returns the workspace history
func (s *serviceImpl) GetWorkspaceHistory() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.history))
	copy(result, s.history)
	return result
}
