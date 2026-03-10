package skills

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/workspace"
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

	// Workspace context methods
	// GetSkillsXML returns all discovered skills in XML format for LLM prompts
	GetSkillsXML() string
	// GetSkillInfos returns skill information for all discovered skills
	GetSkillInfos() []shared.SkillInfo
}

// skillServiceImpl implements SkillService
type skillServiceImpl struct {
	log               *zap.Logger
	workspace         workspace.Service
	toolNameValidator shared.ToolRegistry
	eventBus          events.Bus
	mu                sync.RWMutex
	skills            map[string]*Skill // name -> skill
	skillsByPath      map[string]*Skill // path -> skill
	searchPaths       []string
}

// Ensure skillServiceImpl implements SkillService
var _ SkillService = (*skillServiceImpl)(nil)

// NewService creates a new SkillService instance.
// The service self-initializes by discovering skills during construction.
func NewService(injector do.Injector) (SkillService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	log := logService.GetLogger()
	ws := do.MustInvoke[workspace.Service](injector)
	bus := do.MustInvoke[events.Bus](injector)

	// Try to get ToolRegistry from DI, but it's optional
	// If not available, tool validation will be skipped
	var toolNameValidator shared.ToolRegistry
	if tv, err := do.Invoke[shared.ToolRegistry](injector); err == nil {
		toolNameValidator = tv
	}

	service := &skillServiceImpl{
		log:               log,
		workspace:         ws,
		toolNameValidator: toolNameValidator,
		eventBus:          bus,
		skills:            make(map[string]*Skill),
		skillsByPath:      make(map[string]*Skill),
		searchPaths:       make([]string, 0),
	}

	// Subscribe to events
	if err := service.subscribeToEvents(); err != nil {
		return nil, err
	}

	// Self-initialize: discover skills during construction
	ctx := context.Background()
	if err := service.discoverInternal(ctx); err != nil {
		log.Warn("Skill discovery failed during initialization", zap.Error(err))
		// Non-fatal - service works but has no skills
	} else {
		log.Info("SkillService initialized",
			zap.Int("skills_found", len(service.skills)),
			zap.Strings("search_paths", service.searchPaths))
	}

	return service, nil
}

// subscribeToEvents registers event handlers for the skill service
func (s *skillServiceImpl) subscribeToEvents() error {
	// Subscribe to directory changes - sync with normal priority
	_, err := events.SubscribeTyped[events.DirectoryChangedPayload](s.eventBus, events.EventDirectoryChanged,
		s.handleDirectoryChanged,
		events.WithSync(),
		events.WithPriority(50),
	)
	if err != nil {
		return err
	}

	return nil
}

// handleDirectoryChanged handles directory change events
func (s *skillServiceImpl) handleDirectoryChanged(ctx context.Context, payload events.DirectoryChangedPayload) error {
	s.log.Debug("skill service received directory changed event",
		zap.String("old_path", payload.OldPath),
		zap.String("new_path", payload.NewPath))

	// Trigger skill discovery in the new directory
	if err := s.AddSearchPathAndDiscover(ctx, payload.NewPath); err != nil {
		s.log.Warn("failed to discover skills in new directory",
			zap.String("path", payload.NewPath),
			zap.Error(err))
		// Don't return error - skill discovery failure is non-critical
	}

	return nil
}

// publishSkillsUpdated publishes an event when skills are updated
func (s *skillServiceImpl) publishSkillsUpdated(ctx context.Context, skills []shared.SkillInfo, skillsXML string) {
	if s.eventBus == nil {
		return
	}

	if err := events.PublishTyped(s.eventBus, ctx,
		events.EventSkillsUpdated,
		"skill_service",
		events.SkillsUpdatedPayload{Skills: skills, SkillsXML: skillsXML},
	); err != nil {
		s.log.Warn("failed to publish skills updated event", zap.Error(err))
	}
}

// discoverInternal performs skill discovery without locking (called from constructor)
func (s *skillServiceImpl) discoverInternal(ctx context.Context) error {
	// Store old skill count to detect changes
	oldSkillCount := len(s.skills)

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

	// Remove search paths that yielded no skills
	// Only remove from explicitly added searchPaths, not from workspace history
	var removedPaths []string
	newSearchPaths := make([]string, 0, len(s.searchPaths))
	for _, path := range s.searchPaths {
		if hasSkills, exists := result.PathsWithSkills[path]; exists && !hasSkills {
			removedPaths = append(removedPaths, path)
			s.log.Info("Removing skill search path (no skills found)",
				zap.String("path", path))
		} else {
			newSearchPaths = append(newSearchPaths, path)
		}
	}
	s.searchPaths = newSearchPaths

	s.log.Info("Skill discovery complete",
		zap.Int("skills_found", len(s.skills)),
		zap.Int("errors", len(result.Errors)),
		zap.Int("search_paths_removed", len(removedPaths)),
	)

	// Publish SkillsUpdated event if skills changed
	if len(s.skills) != oldSkillCount || len(s.skills) > 0 {
		skillInfos := make([]shared.SkillInfo, 0, len(s.skills))
		skills := make(Skills, 0, len(s.skills))
		for _, skill := range s.skills {
			skillInfos = append(skillInfos, shared.SkillInfo{
				Name:        skill.Name,
				Description: skill.Description,
				Location:    skill.FilePath,
			})
			skills = append(skills, skill)
		}
		s.publishSkillsUpdated(ctx, skillInfos, skills.ToPromptXML())
	}

	return nil
}

// Discover scans for skills in configured directories (public API for re-discovery)
func (s *skillServiceImpl) Discover(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.discoverInternal(ctx)
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
		if !s.toolNameValidator.IsValidTool(shared.ToolName(tool)) {
			invalid = append(invalid, tool)
		}
	}

	// Validate tool filter
	for _, tool := range skill.ToolFilter {
		if !s.toolNameValidator.IsValidTool(shared.ToolName(tool)) {
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

// GetSkillsXML returns all discovered skills in XML format for LLM prompts
func (s *skillServiceImpl) GetSkillsXML() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make(Skills, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills.ToPromptXML()
}

// GetSkillInfos returns skill information for all discovered skills
func (s *skillServiceImpl) GetSkillInfos() []shared.SkillInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]shared.SkillInfo, 0, len(s.skills))
	for _, skill := range s.skills {
		result = append(result, shared.SkillInfo{
			Name:        skill.Name,
			Description: skill.Description,
			Location:    skill.FilePath,
		})
	}
	return result
}

// getEffectiveSearchPaths returns search paths for skill discovery.
// Searches in:
// 1. ~/.config/gollum/skills/ (global skills)
// 2. <workspace>/.gollum/skills/ (workspace-specific skills)
func (s *skillServiceImpl) getEffectiveSearchPaths() []string {
	paths := make(map[string]bool)

	// Add global skills directory: ~/.config/gollum/skills/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalSkillsDir := filepath.Join(homeDir, ".config", "gollum", "skills")
		paths[globalSkillsDir] = true
	}

	// Add configured search paths
	for _, p := range s.searchPaths {
		paths[p] = true
	}

	// Helper to add .gollum/skills subdirectory from a workspace path
	addWorkspaceSkillsDir := func(workspacePath string) {
		if workspacePath == "" {
			return
		}
		workspaceSkillsDir := filepath.Join(workspacePath, ".gollum", "skills")
		paths[workspaceSkillsDir] = true
	}

	// Add current workspace's .gollum/skills/
	if ws := s.workspace.GetCurrentWorkspace(); ws != "" {
		addWorkspaceSkillsDir(ws)
	}

	// Add workspace history's .gollum/skills/ directories
	for _, p := range s.workspace.GetWorkspaceHistory() {
		addWorkspaceSkillsDir(p)
	}

	// Convert to slice
	result := make([]string, 0, len(paths))
	for p := range paths {
		result = append(result, p)
	}

	return result
}
