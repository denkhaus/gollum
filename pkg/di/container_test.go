package di

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestContainer_ACPService_IsRegistered(t *testing.T) {
	ctx := context.Background()
	container := NewContainer()
	_ = container.RegisterServices(ctx)

	injector := container.GetInjector()

	// Should be able to invoke ACP Service
	service := do.MustInvoke[acp.Service](injector)
	assert.NotNil(t, service)
}
