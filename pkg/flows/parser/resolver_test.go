package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver(t *testing.T) {
	resolver := NewResolver("/path/to/modules", "/another/path")

	assert.NotNil(t, resolver)
	assert.Equal(t, []string{"/path/to/modules", "/another/path"}, resolver.ModulePaths)
}

func TestNewResolver_EmptyPaths(t *testing.T) {
	resolver := NewResolver()

	assert.NotNil(t, resolver)
	assert.Empty(t, resolver.ModulePaths)
}

func TestDefaultResolver(t *testing.T) {
	resolver := DefaultResolver()

	assert.NotNil(t, resolver)
	assert.NotEmpty(t, resolver.ModulePaths)
	// Should contain path to .gollum/flows/modules
	assert.Contains(t, resolver.ModulePaths[0], ".gollum")
	assert.Contains(t, resolver.ModulePaths[0], "flows")
	assert.Contains(t, resolver.ModulePaths[0], "modules")
}

func TestResolver_ResolveCall_AbsolutePath(t *testing.T) {
	resolver := NewResolver("/tmp/modules")

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "flow-*.xml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_ = tmpFile.Close()

	// Absolute path should be returned as-is
	resolved, err := resolver.ResolveCall(tmpFile.Name(), "/some/flow.xml")

	assert.NoError(t, err)
	assert.Equal(t, tmpFile.Name(), resolved)
}

func TestResolver_ResolveCall_RelativePath_ParentDirectory(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "resolver-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory with a flow file
	subDir := filepath.Join(tmpDir, "subflows")
	require.NoError(t, os.Mkdir(subDir, 0755))

	flowFile := filepath.Join(subDir, "other-flow.xml")
	require.NoError(t, os.WriteFile(flowFile, []byte(`<flow name="other" version="1.0"></flow>`), 0644))

	// Current flow is in tmpDir
	currentFlow := filepath.Join(tmpDir, "main.xml")

	resolver := NewResolver(tmpDir)
	resolved, err := resolver.ResolveCall("subflows/other-flow", currentFlow)

	assert.NoError(t, err)
	assert.Equal(t, flowFile, resolved)
}

func TestResolver_ResolveCall_RelativePath_NotFound(t *testing.T) {
	resolver := NewResolver("/tmp/modules")

	resolved, err := resolver.ResolveCall("../non-existent", "/current/flow.xml")

	assert.Error(t, err)
	assert.Empty(t, resolved)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestResolver_ResolveCall_SlashFormat_ModuleNotFound(t *testing.T) {
	resolver := NewResolver("/nonexistent/modules")

	resolved, err := resolver.ResolveCall("my-module/sub-flow", "/current/flow.xml")

	assert.Error(t, err)
	assert.Empty(t, resolved)
	assert.Contains(t, err.Error(), "sub-flow not found")
}

func TestResolver_ResolveCall_SimpleReference_CurrentDirectory(t *testing.T) {
	// Create temporary directory with two flows
	tmpDir, err := os.MkdirTemp("", "resolver-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	mainFlow := filepath.Join(tmpDir, "main.xml")
	require.NoError(t, os.WriteFile(mainFlow, []byte(`<flow name="main" version="1.0"></flow>`), 0644))

	otherFlow := filepath.Join(tmpDir, "other.xml")
	require.NoError(t, os.WriteFile(otherFlow, []byte(`<flow name="other" version="1.0"></flow>`), 0644))

	resolver := NewResolver()
	resolved, err := resolver.ResolveCall("other", mainFlow)

	assert.NoError(t, err)
	assert.Equal(t, otherFlow, resolved)
}

func TestResolver_ResolveCall_SimpleReference_ModuleMain(t *testing.T) {
	// Create temporary module structure
	tmpDir, err := os.MkdirTemp("", "resolver-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	moduleDir := filepath.Join(tmpDir, "my-module")
	require.NoError(t, os.Mkdir(moduleDir, 0755))

	mainFile := filepath.Join(moduleDir, "main.xml")
	require.NoError(t, os.WriteFile(mainFile, []byte(`<flow name="my-module" version="1.0"></flow>`), 0644))

	resolver := NewResolver(tmpDir)
	resolved, err := resolver.ResolveCall("my-module", "/current/flow.xml")

	assert.NoError(t, err)
	assert.Equal(t, mainFile, resolved)
}

func TestResolver_ResolveCall_SimpleReference_NotFound(t *testing.T) {
	resolver := NewResolver("/nonexistent")

	resolved, err := resolver.ResolveCall("nonexistent", "/current/flow.xml")

	assert.Error(t, err)
	assert.Empty(t, resolved)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestResolver_ResolveAll_EmptyFlow(t *testing.T) {
	resolver := NewResolver("/nonexistent")

	// Test with empty flow (no states/calls)
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		States:  []flows.State{},
	}

	resolved, err := resolver.ResolveAll("/current/flow.xml", flow)

	assert.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Empty(t, resolved)
}

func TestResolver_StructFields(t *testing.T) {
	resolver := NewResolver("/path1", "/path2")

	assert.NotNil(t, resolver)
	assert.Equal(t, []string{"/path1", "/path2"}, resolver.ModulePaths)
}
