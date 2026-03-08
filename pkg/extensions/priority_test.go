package extensions

import (
	"testing"
)

func TestService_WorkspacePriority(t *testing.T) {
	// Workspace functions should override global functions
	// This test will create:
	// 1. ~/.config/gollum/functions/test.go with one implementation
	// 2. /workspace/.gollum/functions/test.go with different implementation
	// 3. Verify workspace version is loaded

	t.Skip("TODO: implement priority override test")
}
