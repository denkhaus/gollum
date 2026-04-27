// Package persistence provides session persistence implementation using Ent ORM.
package persistence

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // Register SQLite driver
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// NewSessionRepository creates a SessionRepository with DI.
func NewSessionRepository(injector do.Injector) (repository.SessionRepository, error) {
	cfg := do.MustInvoke[config.ConfigService](injector)
	log := do.MustInvoke[logger.LoggerService](injector)
	workspaceSvc := do.MustInvoke[workspace.Service](injector)

	dbConfig := cfg.GetDatabaseConfig()

	// Resolve database path against workspace directory for SQLite
	dsn := dbConfig.DSN
	if dbConfig.Driver == "sqlite3" {
		// Resolve relative path against workspace
		if !filepath.IsAbs(dsn) {
			workspacePath := workspaceSvc.GetCurrentWorkspace()
			dsn = filepath.Join(workspacePath, dsn)
		}

		// Ensure parent directory exists for SQLite
		dbDir := filepath.Dir(dsn)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Error("Failed to create database directory",
				zap.String("directory", dbDir),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
		}

		// Enable foreign key support (required by Ent)
		dsn += "?_fk=1"
	}

	repo, err := NewEntRepository(dbConfig.Driver, dsn, log)
	if err != nil {
		return nil, err
	}

	log.Info("Session repository initialized",
		zap.String("driver", dbConfig.Driver),
		zap.String("dsn", dsn),
	)

	return repo, nil
}
