package extensions

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDIGateway_NewGatewayService(t *testing.T) {
	injector := do.New()

	gateway, err := NewGatewayService(injector)

	require.NoError(t, err)
	assert.NotNil(t, gateway)
}

func TestDIGateway_Injector(t *testing.T) {
	injector := do.New()
	gateway, err := NewGatewayService(injector)
	require.NoError(t, err)

	returnedInjector := gateway.Injector()

	assert.Same(t, injector, returnedInjector, "should return the same injector instance")
}
