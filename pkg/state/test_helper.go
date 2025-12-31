package state

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

// setupTestInjector creates an injector with all required services for testing
func setupTestInjector() do.Injector {
	injector := do.New()

	// Register config service
	do.Provide(injector, config.NewService)

	// Register logger service
	do.Provide(injector, logger.NewService)

	return injector
}

// Test agent IDs for use in tests
var (
	TestAgent1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	TestAgent2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	TestAgent3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)
