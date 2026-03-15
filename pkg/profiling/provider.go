// Package profiling provides DI provider for profiling service
package profiling

import "github.com/samber/do/v2"

// NewProfilingServiceProvider creates a new profiling service for DI
func NewProfilingServiceProvider(_ do.Injector) (Service, error) {
	// Create pprof provider implementation
	// In a real implementation, this would read config for address, etc.
	// For now, we create it with defaults
	pprofProvider := NewPProfProvider()

	return &pProfProvider{
		PProfProvider: pprofProvider,
	}, nil
}

// pProfProvider wraps the actual profiling implementation
type pProfProvider struct {
	*PProfProvider
}
