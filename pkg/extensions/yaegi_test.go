package extensions

import (
	"errors"
	"os"
	"path/filepath"
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

func TestYaegiLoader_LoadExtension_NameExtraction(t *testing.T) {
	injector := do.New()
	gateway, _ := NewGatewayService(injector)
	do.ProvideValue(injector, gateway)

	loader, err := NewYaegiLoader(injector)
	require.NoError(t, err)

	yaegiLoader := loader.(*yaegiLoaderImpl)

	// Create temporary directories for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		expectedExt string
	}{
		{
			name:        "simple directory",
			path:        filepath.Join(tempDir, "myextension"),
			expectedExt: "myextension",
		},
		{
			name:        "nested path",
			path:        filepath.Join(tempDir, "testext"),
			expectedExt: "testext",
		},
		{
			name:        "relative path",
			path:        filepath.Join(tempDir, "sample"),
			expectedExt: "sample",
		},
	}

	// Create extension main.go files with a valid Init function
	// Note: Must be package main for Yaegi to load properly
	mainGoContent := `
package main

import "fmt"

func Init() error {
	fmt.Println("Extension initialized")
	return nil
}
`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create directory
			err := os.Mkdir(tt.path, 0755)
			require.NoError(t, err)

			// Create main.go
			mainPath := filepath.Join(tt.path, "main.go")
			err = os.WriteFile(mainPath, []byte(mainGoContent), 0644)
			require.NoError(t, err)

			// Clean up directory after test
			defer os.RemoveAll(tt.path)

			ext, err := yaegiLoader.LoadExtension(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedExt, ext.Name)
			assert.Equal(t, tt.path, ext.Path)
			assert.Equal(t, StateLoaded, ext.State)
			assert.NotNil(t, ext.Interpreter)
			assert.NotNil(t, ext.InitFunc)
		})
	}
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

func TestYaegiLoader_InitExtension(t *testing.T) {
	loader := &yaegiLoaderImpl{
		exts: make(map[string]*Extension),
	}

	ext := &Extension{
		Name:     "test",
		State:    StateLoaded,
		InitFunc: func() error { return nil },
	}

	err := loader.InitExtension(ext)
	require.NoError(t, err)
	assert.Equal(t, StateReady, ext.State)
}

func TestYaegiLoader_InitExtension_Error(t *testing.T) {
	loader := &yaegiLoaderImpl{
		exts: make(map[string]*Extension),
	}

	expectedErr := errors.New("init failed")
	ext := &Extension{
		Name:     "test",
		State:    StateLoaded,
		InitFunc: func() error { return expectedErr },
	}

	err := loader.InitExtension(ext)
	assert.Error(t, err)
	assert.Equal(t, StateFailed, ext.State)
	assert.Equal(t, expectedErr, ext.Error)
}

func TestYaegiLoader_UnloadExtension(t *testing.T) {
	loader := &yaegiLoaderImpl{
		exts: make(map[string]*Extension),
	}

	ext := &Extension{
		Name:  "test",
		State: StateReady,
	}
	loader.exts["test"] = ext

	err := loader.UnloadExtension(ext)
	require.NoError(t, err)
	assert.Equal(t, StateUnloaded, ext.State)
	assert.NotContains(t, loader.exts, "test")
}

func TestYaegiLoader_GetExtension(t *testing.T) {
	loader := &yaegiLoaderImpl{
		exts: make(map[string]*Extension),
	}

	ext := &Extension{Name: "test"}
	loader.exts["test"] = ext

	// Success case
	found, err := loader.GetExtension("test")
	require.NoError(t, err)
	assert.Same(t, ext, found)

	// Not found case
	_, err = loader.GetExtension("nonexistent")
	assert.Error(t, err)
}

