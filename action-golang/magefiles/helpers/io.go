package helpers

import "os"

// Global I/O streams
var (
	stdout = os.Stdout
	stderr = os.Stderr
	stdin  = os.Stdin
)
