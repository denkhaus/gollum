package shared

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

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
