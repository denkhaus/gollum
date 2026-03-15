// Package profiling provides CLI flags and configuration for profiling
package profiling

import (
	"flag"
)

// Flags for enabling different profiling modes
const (
	FlagCPUProfile  = "cpuprofile"
	FlagMemProfile  = "memprofile"
	FlagProfilePort = "pprof-port"
	FlagProfileAddr = "pprof-addr"
)

// Config holds profiling configuration
type Config struct {
	// Enable indicates whether profiling is enabled
	Enable bool

	// CPUProfile enables CPU profiling output
	CPUProfile string

	// MemProfile enables memory profiling output
	MemProfile string

	// ProfilePort specifies the HTTP port for pprof endpoint (default: 6060)
	ProfilePort string

	// ProfileAddr specifies the HTTP address for pprof endpoint (default: "localhost:6060")
	ProfileAddr string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Enable:      false,
		CPUProfile:  "",
		MemProfile:  "",
		ProfilePort: "6060",
		ProfileAddr: "localhost:6060",
	}
}

// DefineFlags registers profiling flags with the flag package
func DefineFlags() {
	flag.StringVar(&profilingCPUProfile, FlagCPUProfile, "", "Enable CPU profiling to file")
	flag.StringVar(&profilingMemProfile, FlagMemProfile, "", "Enable memory profiling to file")
	flag.StringVar(&profilingProfilePort, FlagProfilePort, "6060", "HTTP pprof server port")
	flag.StringVar(&profilingProfileAddr, FlagProfileAddr, "localhost:6060", "HTTP pprof server address")
}

// Package-level variables for flag storage
var (
	profilingCPUProfile  string
	profilingMemProfile  string
	profilingProfilePort string
	profilingProfileAddr string
)

// ParseFlags parses command-line flags and returns config
func ParseFlags() *Config {
	config := DefaultConfig()

	// Check if any profiling flags were set
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case FlagCPUProfile:
			if f.Value.String() != "" {
				config.Enable = true
				config.CPUProfile = f.Value.String()
			}
		case FlagMemProfile:
			if f.Value.String() != "" {
				config.Enable = true
				config.MemProfile = f.Value.String()
			}
		case FlagProfilePort:
			config.ProfilePort = f.Value.String()
		case FlagProfileAddr:
			config.ProfileAddr = f.Value.String()
		}
	})

	return config
}
