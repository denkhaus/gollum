package extensions

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYaegiLoader_NewYaegiLoader(t *testing.T) {
	injector := do.New()
	gateway, _ := NewGatewayService(injector)
	do.ProvideValue(injector, gateway)

	loader, err := NewYaegiLoader(injector)

	require.NoError(t, err)
	assert.NotNil(t, loader)
}

func TestYaegiLoader_ListExtensions(t *testing.T) {
	loader := &yaegiLoaderImpl{
		gateway: nil,
		exts:    make(map[string]*Extension),
	}

	exts := loader.ListExtensions()
	assert.NotNil(t, exts)
	assert.Empty(t, exts)
}

func TestExtensionState_String(t *testing.T) {
	tests := []struct {
		state    ExtensionState
		expected string
	}{
		{StateLoaded, "loaded"},
		{StateInitializing, "initializing"},
		{StateReady, "ready"},
		{StateFailed, "failed"},
		{StateUnloading, "unloading"},
		{StateUnloaded, "unloaded"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}
