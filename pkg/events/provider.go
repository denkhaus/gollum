package events

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
)

// NewBusProvider creates a new event bus instance for DI.
func NewBusProvider(injector do.Injector) (Bus, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	cfgService := do.MustInvoke[config.ConfigService](injector)
	return newMemoryBus(logService, cfgService.GetEventsConfig()), nil
}
