// Package persistence provides session persistence implementation using Ent ORM.
package persistence

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// NewSessionRepository creates a SessionRepository with DI.
func NewSessionRepository(injector do.Injector) (repository.SessionRepository, error) {
	cfg := do.MustInvoke[config.ConfigService](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	dbConfig := cfg.GetDatabaseConfig()

	repo, err := NewEntRepository(dbConfig.Driver, dbConfig.DSN, log)
	if err != nil {
		return nil, err
	}

	log.Info("Session repository initialized",
		zap.String("driver", dbConfig.Driver),
		zap.String("dsn", dbConfig.DSN),
	)

	return repo, nil
}
