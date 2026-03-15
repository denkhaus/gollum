package extensions

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestLoadFuncSteps_FromDirectory(t *testing.T) {
	// Create mock filesystem
	_ = fstest.MapFS{
		"functions/add.go":      {Data: []byte("package main\nfunc Add(a, b int) int { return a + b }")},
		"functions/double.go":   {Data: []byte("package main\nfunc Double(x int) int { return x * 2 }")},
		"functions/invalid.txt": {Data: []byte("not a go file")},
		"functions/.hidden.go":  {Data: []byte("package main\nfunc Hidden() {}")},
	}

	// Test loading
	// Implementation will verify:
	// 1. Only .go files are loaded
	// 2. Hidden files are skipped
	// 3. Functions are compiled successfully
	t.Skip("TODO: implement after filesystem loading")
}

func TestGetGlobalGollumDir(t *testing.T) {
	dir := getGlobalGollumDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, ".config")
	assert.Contains(t, dir, "gollum")
}
