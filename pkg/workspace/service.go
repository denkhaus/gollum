// Package workspace provides workspace state management
// for the Gollum agent system.
package workspace

import (
	"os"
	"sync"

	"github.com/samber/do/v2"
)

// SkillInfo holds information about a discovered skill
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

// WorkspaceContext holds the current workspace state
type WorkspaceContext struct {
	CurrentPath      string      `json:"current_path"`
	WorkspaceHistory []string    `json:"workspace_history"`
	Skills           []SkillInfo `json:"skills"`
	SkillsXML        string      `json:"skills_xml"`
}

// Service manages workspace state during runtime.
// It manages workspace paths and skills context.
type Service interface {
	// Workspace path management
	SetCurrentWorkspace(path string) error
	GetCurrentWorkspace() string
	AddToHistory(path string)
	GetWorkspaceHistory() []string
	ClearHistory()

	// Skills context (populated by SkillService)
	SetSkillsContext(skills []SkillInfo, skillsXML string)
	GetSkillsContext() []SkillInfo
	GetSkillsXML() string

	// Context export
	GetWorkspaceContext() *WorkspaceContext
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	mu             sync.RWMutex
	currentPath    string
	history        []string
	maxHistorySize int
	skills         []SkillInfo
	skillsXML      string
}

// Ensure serviceImpl implements Service
var _ Service = (*serviceImpl)(nil)

// NewServiceProvider creates a new workspace service for DI
func NewServiceProvider(injector do.Injector) (Service, error) {
	service := &serviceImpl{
		currentPath:    "",
		history:        make([]string, 0),
		maxHistorySize: 10, // Keep last 10 workspaces
		skills:         make([]SkillInfo, 0),
		skillsXML:      "",
	}

	// Initialize with current working directory
	if wd, err := os.Getwd(); err == nil && wd != "" {
		service.SetCurrentWorkspace(wd)
	}

	return service, nil
}

// SetCurrentWorkspace sets the current workspace path
func (s *serviceImpl) SetCurrentWorkspace(path string) error {
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

// AddToHistory adds a path to the workspace history
func (s *serviceImpl) AddToHistory(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addToHistoryLocked(path)
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

// ClearHistory clears the workspace history
func (s *serviceImpl) ClearHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = make([]string, 0)
}

// GetWorkspaceContext returns the workspace context (for prompt rendering)
func (s *serviceImpl) GetWorkspaceContext() *WorkspaceContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]string, len(s.history))
	copy(history, s.history)

	skills := make([]SkillInfo, len(s.skills))
	copy(skills, s.skills)

	return &WorkspaceContext{
		CurrentPath:      s.currentPath,
		WorkspaceHistory: history,
		Skills:           skills,
		SkillsXML:        s.skillsXML,
	}
}

// SetSkillsContext updates the skills context
func (s *serviceImpl) SetSkillsContext(skills []SkillInfo, skillsXML string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.skills = make([]SkillInfo, len(skills))
	copy(s.skills, skills)
	s.skillsXML = skillsXML
}

// GetSkillsContext returns the current skills
func (s *serviceImpl) GetSkillsContext() []SkillInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SkillInfo, len(s.skills))
	copy(result, s.skills)
	return result
}

// GetSkillsXML returns the skills in XML format
func (s *serviceImpl) GetSkillsXML() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.skillsXML
}
