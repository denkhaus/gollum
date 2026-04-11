package di

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestContainer_RegisterServices_Success(t *testing.T) {
	ctx := context.Background()
	container := NewContainer()
	injector := container.RegisterServices(ctx)

	// Should be able to invoke basic services without circular dependency
	cfg := do.MustInvoke[config.ConfigService](injector)
	assert.NotNil(t, cfg)

	log := do.MustInvoke[logger.LoggerService](injector)
	assert.NotNil(t, log)
}
