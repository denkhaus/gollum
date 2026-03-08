package extensions

import (
	"context"
	"os"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
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
	loadedFuncs   map[string]string // funcName -> sourcePath
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
		loadedFuncs:   make(map[string]string),
	}, nil
}

func (p *extensionServiceImpl) LoadAll(ctx context.Context) error {
	p.logService.Info("Extension service: starting load",
		zap.String("workspace", p.workspaceDir),
	)

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		p.logService.Error("Extension service: context cancelled before load")
		return ctx.Err()
	default:
	}

	if err := p.loadFuncSteps(); err != nil {
		p.logService.Error("Extension service: failed to load func steps",
			zap.Error(err),
		)
		return err
	}

	if err := p.loadExtensions(); err != nil {
		p.logService.Error("Extension service: failed to load extensions",
			zap.Error(err),
		)
		return err
	}

	p.logService.Info("Extension service: load complete",
		zap.Int("functions", len(p.scriggoRunner.ListFuncs())),
		zap.Int("extensions", len(p.yaegiLoader.ListExtensions())),
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
	if p.loadedFuncs == nil {
		p.loadedFuncs = make(map[string]string)
	}

	dirs := p.getFunctionsDirs()
	for _, dir := range dirs {
		p.logService.Debug("Scanning for func steps",
			zap.String("directory", dir),
		)

		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			p.logService.Debug("Directory does not exist",
				zap.String("directory", dir),
			)
			continue
		}

		// Read directory entries
		entries, err := os.ReadDir(dir)
		if err != nil {
			p.logService.Warn("Failed to read directory",
				zap.String("directory", dir),
				zap.Error(err),
			)
			continue
		}

		// Load each .go file
		for _, entry := range entries {
			// Skip hidden files and non-.go files
			if entry.Name()[0] == '.' || filepath.Ext(entry.Name()) != ".go" {
				continue
			}

			// Extract function name from filename
			funcName := entry.Name()[:len(entry.Name())-3] // remove .go

			// Skip if already loaded from higher priority dir
			if _, exists := p.loadedFuncs[funcName]; exists {
				p.logService.Debug("Function already loaded, skipping",
					zap.String("function", funcName),
					zap.String("loaded_from", p.loadedFuncs[funcName]),
					zap.String("skipping", filepath.Join(dir, entry.Name())),
				)
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			p.logService.Debug("Loading func step",
				zap.String("name", funcName),
				zap.String("source", filePath),
			)

			// Read source
			source, err := os.ReadFile(filePath)
			if err != nil {
				p.logService.Warn("Failed to read file",
					zap.String("file", filePath),
					zap.Error(err),
				)
				continue
			}

			// Load via ScriggoRunner
			if err := p.scriggoRunner.LoadFunc(funcName, string(source)); err != nil {
				p.logService.Warn("Failed to compile function",
					zap.String("file", filePath),
					zap.Error(err),
				)
				continue
			}

			p.loadedFuncs[funcName] = filePath
			p.logService.Info("Loaded func step",
				zap.String("name", funcName),
				zap.String("source", filePath),
			)
		}
	}
	return nil
}

func (p *extensionServiceImpl) loadExtensions() error {
	dirs := p.getExtensionsDirs()
	for _, dir := range dirs {
		p.logService.Debug("Scanning for extensions",
			zap.String("directory", dir),
		)

		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			p.logService.Debug("Directory does not exist",
				zap.String("directory", dir),
			)
			continue
		}

		// Read directory entries
		entries, err := os.ReadDir(dir)
		if err != nil {
			p.logService.Warn("Failed to read directory",
				zap.String("directory", dir),
				zap.Error(err),
			)
			continue
		}

		// Load each subdirectory as an extension
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			extPath := filepath.Join(dir, entry.Name())
			p.logService.Info("Loading extension",
				zap.String("name", entry.Name()),
				zap.String("path", extPath),
			)

			// Load extension
			ext, err := p.yaegiLoader.LoadExtension(extPath)
			if err != nil {
				p.logService.Warn("Failed to load extension",
					zap.String("name", entry.Name()),
					zap.Error(err),
				)
				continue
			}

			// Initialize extension
			if err := p.yaegiLoader.InitExtension(ext); err != nil {
				p.logService.Warn("Failed to initialize extension",
					zap.String("name", ext.Name),
					zap.Error(err),
				)
				continue
			}

			p.logService.Info("Loaded extension",
				zap.String("name", ext.Name),
			)
		}
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
