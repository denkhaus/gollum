// Package profiling provides main entry point for profiling functionality
package profiling

import (
	"context"
	"flag"
	"fmt"
)

// Version of the profiling package
const Version = "1.0.0"

// Main runs profiling setup from command-line flags
func Main(ctx context.Context) error {
	// Parse flags
	config := ParseFlags()

	// If no flags provided, show help
	if flag.NArg() == 1 {
		fmt.Printf("Gollum Profiling Tool v%s\n", Version)
		fmt.Println("\nUsage:")
		fmt.Println("  gollum profiling [flags]")
		fmt.Println("\nFlags:")
		fmt.Printf("  --%-20s  %-10s  Enable profiling mode\n", "help", "Show this help message")
		fmt.Printf("      %-20s  %-10s  %s\n", "cpuprofile", "CPU", "Enable CPU profiling to file")
		fmt.Printf("      %-20s  %-10s  %s\n", "memprofile", "MEM", "Enable memory profiling to file")
		fmt.Printf("      %-20s  %-10s  %s (default: \"localhost:6060\")\n", "pprof-port", "PORT", "HTTP pprof server port")
		fmt.Printf("      %-20s  %-10s  %s (default: \"localhost:6060\")\n", "pprof-addr", "ADDR", "HTTP pprof server address")
		fmt.Println("\nProfiling Modes:")
		fmt.Println("  CPU profiling can be enabled with --cpuprofile flag")
		fmt.Println("  Memory profiling can be enabled with --memprofile flag")
		fmt.Println("  HTTP pprof server can be started with --pprof-port flag")
		fmt.Println("\nExamples:")
		fmt.Println("  # Enable CPU profiling and save to cpu.prof:")
		fmt.Println("  gollum profiling --cpuprofile=cpu.prof")
		fmt.Println("  # Enable memory profiling and save to mem.prof:")
		fmt.Println("  gollum profiling --memprofile=mem.prof")
		fmt.Println("  # Start HTTP pprof server on port 8080:")
		fmt.Println("  gollum profiling --pprof-port=:8080")
	}

	// If profiling is enabled, show configuration
	if config.Enable {
		fmt.Println("\nProfiling enabled:")
		fmt.Printf("  CPU Profile: %s\n", config.CPUProfile)
		fmt.Printf("  Memory Profile: %s\n", config.MemProfile)
		fmt.Printf("  HTTP Server: %s\n", config.ProfileAddr)
	}

	return nil
}
