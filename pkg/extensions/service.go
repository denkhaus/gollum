package extensions

import (
	"context"
	"os"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
)

// ExtensionService manages extension and function loading
type ExtensionService interface {
	// LoadAll discovers and loads all extensions and functions
	LoadAll(ctx context.Context) error

	// GetFuncRunner returns the Scriggo function runner
	GetFuncRunner() ScriggoRunner

	// GetExtension returns a loaded extension by name
	GetExtension(name string) (*Extension, error)

	// ListExtensions returns all loaded extension names
	ListExtensions() []string
}

// extensionServiceImpl is the private implementation
type extensionServiceImpl struct {
	logService    logger.LoggerService
	gateway       DIGateway
	yaegiLoader   YaegiLoader
	scriggoRunner ScriggoRunner
	workspaceDir  string
}

// Ensure extensionServiceImpl implements ExtensionService
var _ ExtensionService = (*extensionServiceImpl)(nil)

// NewExtensionServiceWithWorkspace creates the extension service
func NewExtensionServiceWithWorkspace(injector do.Injector) (ExtensionService, error) {
	logService, err := do.Invoke[logger.LoggerService](injector)
	if err != nil {
		return nil, err
	}

	gateway, err := do.Invoke[DIGateway](injector)
	if err != nil {
		return nil, err
	}

	yaegiLoader, err := do.Invoke[YaegiLoader](injector)
	if err != nil {
		return nil, err
	}

	scriggoRunner, err := do.Invoke[ScriggoRunner](injector)
	if err != nil {
		return nil, err
	}

	workspaceService, err := do.Invoke[workspace.Service](injector)
	if err != nil {
		return nil, err
	}

	return &extensionServiceImpl{
		logService:    logService,
		gateway:       gateway,
		yaegiLoader:   yaegiLoader,
		scriggoRunner: scriggoRunner,
		workspaceDir:  workspaceService.GetCurrentWorkspace(),
	}, nil
}

func (p *extensionServiceImpl) LoadAll(ctx context.Context) error {
	p.logService.Info("Extension service: loading extensions and functions...")

	if err := p.loadFuncSteps(); err != nil {
		return err
	}

	if err := p.loadExtensions(); err != nil {
		return err
	}

	p.logService.Infof("Extension service: loaded %d functions, %d extensions",
		len(p.scriggoRunner.ListFuncs()),
		len(p.yaegiLoader.ListExtensions()),
	)

	return nil
}

func (p *extensionServiceImpl) GetFuncRunner() ScriggoRunner {
	return p.scriggoRunner
}

func (p *extensionServiceImpl) GetExtension(name string) (*Extension, error) {
	return p.yaegiLoader.GetExtension(name)
}

func (p *extensionServiceImpl) ListExtensions() []string {
	return p.yaegiLoader.ListExtensions()
}

func (p *extensionServiceImpl) getFunctionsDirs() []string {
	var dirs []string

	// Priority 1: Workspace-local (PRIMARY)
	if p.workspaceDir != "" {
		dirs = append(dirs, filepath.Join(p.workspaceDir, ".gollum", "functions"))
	}

	// Priority 2: User space (for testing/reusable)
	dirs = append(dirs, filepath.Join(getGlobalGollumDir(), "functions"))

	return dirs
}

func (p *extensionServiceImpl) getExtensionsDirs() []string {
	var dirs []string

	// Priority 1: Workspace-local (PRIMARY)
	if p.workspaceDir != "" {
		dirs = append(dirs, filepath.Join(p.workspaceDir, ".gollum", "extensions"))
	}

	// Priority 2: User space (for testing/reusable)
	dirs = append(dirs, filepath.Join(getGlobalGollumDir(), "extensions"))

	return dirs
}

func (p *extensionServiceImpl) loadFuncSteps() error {
	dirs := p.getFunctionsDirs()
	for _, dir := range dirs {
		// Load .go files from directory
		// Implementation will be added in next tasks
		p.logService.Debugf("Scanning for func steps in: %s", dir)
	}
	return nil
}

func (p *extensionServiceImpl) loadExtensions() error {
	dirs := p.getExtensionsDirs()
	for _, dir := range dirs {
		// Load extension subdirectories
		// Implementation will be added in next tasks
		p.logService.Debugf("Scanning for extensions in: %s", dir)
	}
	return nil
}

func getGlobalGollumDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "gollum")
}
