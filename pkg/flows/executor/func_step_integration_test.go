package executor

import (
	"testing"
)

func TestExecutor_FuncStep_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test requires:
	// 1. A Scriggo function loaded at ~/.gollum/functions/double.go
	// 2. The flow fixture to be available
	// Full implementation will be added after directory loading is complete
	t.Skip("TODO: implement after directory loading is complete")
}
