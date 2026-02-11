// Package profiling provides profiling service interface
package profiling

// ProfilingService defines the interface for runtime profiling operations
type ProfilingService interface {
	// Enable activates profiling with optional HTTP server address
	Enable(addr string) error

	// Disable deactivates profiling and closes resources
	Disable() error

	// StartCPUProfiling starts CPU profiling
	StartCPUProfiling() error

	// StopCPUProfiling stops CPU profiling
	StopCPUProfiling() error

	// StartMemProfiling starts memory profiling
	StartMemProfiling() error

	// StopMemProfiling stops memory profiling
	StopMemProfiling() error

	// IsEnabled returns whether profiling is active
	IsEnabled() bool
}
