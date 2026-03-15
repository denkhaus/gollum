package cli

import (
	"log"

	"github.com/denkhaus/gollum/pkg/profiling"
	"github.com/samber/do/v2"
)

func setupProfiling(config *profiling.Config, injector do.Injector) func() {
	profilingService := do.MustInvoke[profiling.Service](injector)
	if err := profilingService.Enable(config.ProfileAddr); err != nil {
		log.Printf("Warning: failed to enable profiling: %v", err)
		return func() {}
	} else {
		log.Printf("Profiling enabled on %s", config.ProfileAddr)

		// Start CPU profiling if configured
		if config.CPUProfile != "" {
			if err := profilingService.StartCPUProfiling(); err != nil {
				log.Printf("Warning: failed to start CPU profiling: %v", err)
			} else {
				log.Printf("CPU profiling to %s", config.CPUProfile)
				defer func() {
					if err := profilingService.StopCPUProfiling(); err != nil {
						log.Printf("Warning: failed to stop CPU profiling: %v", err)
					}
				}()
			}
		}

		// Start memory profiling if configured
		if config.MemProfile != "" {
			if err := profilingService.StartMemProfiling(); err != nil {
				log.Printf("Warning: failed to start memory profiling: %v", err)
			} else {
				log.Printf("Memory profiling to %s", config.MemProfile)
				defer func() {
					if err := profilingService.StopMemProfiling(); err != nil {
						log.Printf("Warning: failed to stop memory profiling: %v", err)
					}
				}()
			}
		}

		return func() {
			if err := profilingService.Disable(); err != nil {
				log.Printf("Warning: failed to disable profiling: %v", err)
			}
		}
	}
}
