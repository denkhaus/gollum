// Package profiling provides HTTP pprof endpoint for runtime profiling
package profiling

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
)

// PProfProvider serves runtime profiling data via HTTP
type PProfProvider struct {
	enabled    bool
	cpuProfile *os.File
	memProfile *os.File
	server     *http.Server
}

// NewPProfProvider creates a new pprof HTTP provider
func NewPProfProvider() *PProfProvider {
	return &PProfProvider{
		enabled: false,
	}
}

// Enable activates profiling and starts the HTTP server
func (p *PProfProvider) Enable(addr string) error {
	if p.enabled {
		return fmt.Errorf("profiling already enabled")
	}

	// Create profile files
	cpuFile, err := os.CreateTemp("", "cpu.prof")
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	memFile, err := os.CreateTemp("", "mem.prof")
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}

	p.cpuProfile = cpuFile
	p.memProfile = memFile

	// Start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", p.pprofHandler)

	p.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash - server might be closed intentionally
		}
	}()

	p.enabled = true
	return nil
}

// Disable stops profiling and closes the HTTP server
func (p *PProfProvider) Disable() error {
	if !p.enabled {
		return fmt.Errorf("profiling not enabled")
	}

	// Stop HTTP server
	if p.server != nil {
		if err := p.server.Close(); err != nil {
			return fmt.Errorf("failed to close pprof server: %w", err)
		}
	}

	// Close profile files
	if p.cpuProfile != nil {
		p.cpuProfile.Close()
	}
	if p.memProfile != nil {
		p.memProfile.Close()
	}

	p.enabled = false
	p.cpuProfile = nil
	p.memProfile = nil
	p.server = nil

	return nil
}

// StartCPUProfiling starts CPU profiling
func (p *PProfProvider) StartCPUProfiling() error {
	if !p.enabled {
		return fmt.Errorf("profiling not enabled")
	}
	if p.cpuProfile == nil {
		return fmt.Errorf("CPU profile not initialized")
	}

	return pprof.StartCPUProfile(p.cpuProfile)
}

// StopCPUProfiling stops CPU profiling
func (p *PProfProvider) StopCPUProfiling() error {
	if !p.enabled || p.cpuProfile == nil {
		return fmt.Errorf("CPU profiling not enabled")
	}

	pprof.StopCPUProfile()
	return nil
}

// StartMemProfiling starts memory profiling
func (p *PProfProvider) StartMemProfiling() error {
	if !p.enabled {
		return fmt.Errorf("profiling not enabled")
	}
	if p.memProfile == nil {
		return fmt.Errorf("memory profile not initialized")
	}

	return pprof.WriteHeapProfile(p.memProfile)
}

// StopMemProfiling stops memory profiling
func (p *PProfProvider) StopMemProfiling() error {
	if !p.enabled || p.memProfile == nil {
		return fmt.Errorf("memory profiling not enabled")
	}

	// No explicit stop for heap profile - pprof handles cleanup
	return nil
}

// pprofHandler serves pprof data from runtime/pprof
func (p *PProfProvider) pprofHandler(w http.ResponseWriter, r *http.Request) {
	// Delegate to standard pprof handlers
	// The net/http/pprof import already registers handlers
	// We just need to use the default ServeMux or explicitly handle
	http.DefaultServeMux.ServeHTTP(w, r)
}

// IsEnabled returns whether profiling is currently enabled
func (p *PProfProvider) IsEnabled() bool {
	return p.enabled
}
