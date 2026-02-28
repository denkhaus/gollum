package skills

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// SkillService defines the interface for skill management
type SkillService interface {
	// Discover scans for skills in the configured directories
	Discover(ctx context.Context) error
	// Get retrieves a skill by name
	Get(name string) (*Skill, error)
	// GetByPath retrieves a skill by its file path
	GetByPath(path string) (*Skill, error)
	// List returns all discovered skills
	List() Skills
	// ListByType returns skills of a specific type
	ListByType(skillType SkillType) Skills
	// ListUserInvocable returns skills that can be invoked by users
	ListUserInvocable() Skills
	// Validate validates a skill (including tool names)
	Validate(skill *Skill) error
	// ValidateTools validates only the tool names in a skill
	ValidateTools(skill *Skill) []string
	// Refresh rediscovery all skills
	Refresh(ctx context.Context) error
	// AddSearchPath adds a directory to search for skills
	AddSearchPath(path string)
	// AddSearchPathAndDiscover adds a directory to search for skills and triggers discovery
	AddSearchPathAndDiscover(ctx context.Context, path string) error
	// RemoveSearchPath removes a directory from search paths
	RemoveSearchPath(path string)
	// GetSearchPaths returns current search paths
	GetSearchPaths() []string
}

// skillServiceImpl implements SkillService
type skillServiceImpl struct {
	log             *zap.Logger
	config          *config.WorkspaceConfig
	toolNameValidator ToolNameValidator
	mu              sync.RWMutex
	skills          map[string]*Skill // name -> skill
	skillsByPath    map[string]*Skill // path -> skill
	searchPaths     []string
}

// Ensure skillServiceImpl implements SkillService
var _ SkillService = (*skillServiceImpl)(nil)

// NewService creates a new SkillService instance
func NewService(injector do.Injector) (SkillService, error) {
	log := do.MustInvoke[*zap.Logger](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	// Try to get ToolNameValidator from DI, but it's optional
	// If not available, tool validation will be skipped
	var toolNameValidator ToolNameValidator
	if tv, err := do.Invoke[ToolNameValidator](injector); err == nil {
		toolNameValidator = tv
	}

	service := &skillServiceImpl{
		log:                log,
		config:             cfg.GetWorkspaceConfig(),
		toolNameValidator:  toolNameValidator,
		skills:             make(map[string]*Skill),
		skillsByPath:       make(map[string]*Skill),
		searchPaths:        make([]string, 0),
	}

	// Add workspace directory as default search path
	if ws := service.config.GetCurrentWorkspace(); ws != "" {
		service.searchPaths = append(service.searchPaths, ws)
	}

	return service, nil
}

// Discover scans for skills in configured directories
func (s *skillServiceImpl) Discover(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing skills
	s.skills = make(map[string]*Skill)
	s.skillsByPath = make(map[string]*Skill)

	// Get search paths
	paths := s.getEffectiveSearchPaths()
	if len(paths) == 0 {
		s.log.Info("No search paths configured for skill discovery")
		return nil
	}

	// Discover skills in all paths
	result, err := DiscoverMultiple(ctx, paths, s.log)
	if err != nil {
		return err
	}

	// Store discovered skills
	for _, skill := range result.Skills {
		s.skills[skill.ID()] = skill
		s.skillsByPath[skill.FilePath] = skill
	}

	// Log any errors encountered
	for _, discoveryErr := range result.Errors {
		s.log.Warn("Skill discovery error", zap.Error(discoveryErr))
	}

	s.log.Info("Skill discovery complete",
		zap.Int("skills_found", len(s.skills)),
		zap.Int("errors", len(result.Errors)),
	)

	return nil
}

// Get retrieves a skill by name (case-insensitive)
func (s *skillServiceImpl) Get(name string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id := strings.ToLower(name)
	skill, exists := s.skills[id]
	if !exists {
		return nil, ErrSkillNotFound(name)
	}
	return skill, nil
}

// GetByPath retrieves a skill by its file path
func (s *skillServiceImpl) GetByPath(path string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, exists := s.skillsByPath[path]
	if !exists {
		return nil, ErrSkillNotFound(path)
	}
	return skill, nil
}

// List returns all discovered skills
func (s *skillServiceImpl) List() Skills {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(Skills, 0, len(s.skills))
	for _, skill := range s.skills {
		result = append(result, skill)
	}
	return result
}

// ListByType returns skills of a specific type
func (s *skillServiceImpl) ListByType(skillType SkillType) Skills {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result Skills
	for _, skill := range s.skills {
		if skill.Type == skillType {
			result = append(result, skill)
		}
	}
	return result
}

// ListUserInvocable returns skills that can be invoked by users
func (s *skillServiceImpl) ListUserInvocable() Skills {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result Skills
	for _, skill := range s.skills {
		if skill.UserInvocable {
			result = append(result, skill)
		}
	}
	return result
}

// Validate validates a skill (including tool names)
func (s *skillServiceImpl) Validate(skill *Skill) error {
	// First, validate the skill structure
	if err := skill.Validate(); err != nil {
		return err
	}

	// Then validate tool names
	invalidTools := s.ValidateTools(skill)
	if len(invalidTools) > 0 {
		return ErrInvalidToolNames(skill.FilePath, invalidTools)
	}

	return nil
}

// ValidateTools validates only the tool names in a skill
// Returns a list of invalid tool names (empty if all valid)
func (s *skillServiceImpl) ValidateTools(skill *Skill) []string {
	// If no validator is available, skip tool validation
	if s.toolNameValidator == nil {
		return nil
	}

	var invalid []string

	// Validate tools whitelist
	for _, tool := range skill.Tools {
		if !s.toolNameValidator.IsValidTool(tool) {
			invalid = append(invalid, tool)
		}
	}

	// Validate tool filter
	for _, tool := range skill.ToolFilter {
		if !s.toolNameValidator.IsValidTool(tool) {
			invalid = append(invalid, tool)
		}
	}

	return invalid
}

// Refresh rediscovery all skills
func (s *skillServiceImpl) Refresh(ctx context.Context) error {
	return s.Discover(ctx)
}

// AddSearchPath adds a directory to search for skills
func (s *skillServiceImpl) AddSearchPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	if slices.Contains(s.searchPaths, path) {
		return
	}

	s.searchPaths = append(s.searchPaths, path)
	s.log.Info("Added skill search path", zap.String("path", path))
}

// AddSearchPathAndDiscover adds a directory to search for skills and triggers discovery
func (s *skillServiceImpl) AddSearchPathAndDiscover(ctx context.Context, path string) error {
	s.mu.Lock()

	// Check for duplicates
	if slices.Contains(s.searchPaths, path) {
		s.mu.Unlock()
		return nil // Already exists, no need to discover
	}

	s.searchPaths = append(s.searchPaths, path)
	s.log.Info("Added skill search path", zap.String("path", path))

	s.mu.Unlock()

	// Trigger discovery with the new path included
	return s.Discover(ctx)
}

// RemoveSearchPath removes a directory from search paths
func (s *skillServiceImpl) RemoveSearchPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.searchPaths {
		if p == path {
			s.searchPaths = append(s.searchPaths[:i], s.searchPaths[i+1:]...)
			s.log.Info("Removed skill search path", zap.String("path", path))
			return
		}
	}
}

// GetSearchPaths returns current search paths
func (s *skillServiceImpl) GetSearchPaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.searchPaths))
	copy(result, s.searchPaths)
	return result
}

// getEffectiveSearchPaths returns search paths including workspace history
func (s *skillServiceImpl) getEffectiveSearchPaths() []string {
	paths := make(map[string]bool)

	// Add configured search paths
	for _, p := range s.searchPaths {
		paths[p] = true
	}

	// Add current workspace
	if ws := s.config.GetCurrentWorkspace(); ws != "" {
		paths[ws] = true
	}

	// Add workspace history
	for _, p := range s.config.GetWorkspaceHistory() {
		paths[p] = true
	}

	// Convert to slice
	result := make([]string, 0, len(paths))
	for p := range paths {
		result = append(result, p)
	}

	return result
}
