package shared

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestSetAndGetInjector(t *testing.T) {
	cmd := &cli.Command{}
	injector := do.New()

	SetInjector(cmd, injector)

	retrieved, err := GetInjector(cmd)
	assert.NilError(t, err)
	assert.Equal(t, injector, retrieved)
}

func TestGetInjectorNilMetadata(t *testing.T) {
	cmd := &cli.Command{Metadata: nil}

	_, err := GetInjector(cmd)
	assert.ErrorContains(t, err, "metadata is nil")
}

func TestGetInjectorNotFound(t *testing.T) {
	cmd := &cli.Command{}
	cmd.Metadata = make(map[string]any)

	_, err := GetInjector(cmd)
	assert.ErrorContains(t, err, "injector not found")
}

func TestGetInjectorInvalidType(t *testing.T) {
	cmd := &cli.Command{}
	cmd.Metadata = make(map[string]any)
	cmd.Metadata[MetadataInjectorKey] = "not an injector"

	_, err := GetInjector(cmd)
	assert.ErrorContains(t, err, "invalid injector type")
}

func TestMustGetInjector(t *testing.T) {
	cmd := &cli.Command{}
	injector := do.New()
	SetInjector(cmd, injector)

	retrieved := MustGetInjector(cmd)
	assert.Equal(t, injector, retrieved)
}

func TestMustGetInjectorPanics(t *testing.T) {
	t.Run("nil metadata", func(t *testing.T) {
		cmd := &cli.Command{Metadata: nil}
		assert.Assert(t, cmp.Panics(func() { MustGetInjector(cmd) }))
	})

	t.Run("not found", func(t *testing.T) {
		cmd := &cli.Command{}
		cmd.Metadata = make(map[string]any)
		assert.Assert(t, cmp.Panics(func() { MustGetInjector(cmd) }))
	})
}
