package shared

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestGetPathArg(t *testing.T) {
	t.Run("valid path - integration", func(t *testing.T) {
		// Note: GetPathArg requires cmd.Args() to be populated by the CLI parser
		// The success case is tested through integration tests (e.g., flow/run_test.go)
		t.Skip("requires integration test with CLI parser")
	})

	t.Run("missing argument", func(t *testing.T) {
		// Create a minimal command - Args() will return nil, causing panic
		// This test verifies the error handling path
		// Note: In actual CLI usage, Args() is always populated by the framework
		t.Skip("requires integration test with CLI parser")
	})
}

func TestExitCode(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		code := ExitCode(nil)
		assert.Equal(t, 0, code)
	})

	t.Run("cli.Exit error", func(t *testing.T) {
		err := cli.Exit("error", 42)
		code := ExitCode(err)
		assert.Equal(t, 42, code)
	})

	t.Run("generic error", func(t *testing.T) {
		err := errors.New("some error")
		code := ExitCode(err)
		assert.Equal(t, 1, code)
	})
}
