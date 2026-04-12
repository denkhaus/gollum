package acp

import "fmt"

// TransportType represents the transport protocol for ACP communication
type TransportType int

const (
	// TransportStdio uses stdin/stdout for communication (default)
	TransportStdio TransportType = iota
	// TransportHTTP uses HTTP + Server-Sent Events for communication
	TransportHTTP
)

// String returns the string representation of the transport type
func (t TransportType) String() string {
	switch t {
	case TransportStdio:
		return "stdio"
	case TransportHTTP:
		return "http"
	default:
		return "unknown"
	}
}

// ParseTransportType parses a string into TransportType
func ParseTransportType(s string) (TransportType, error) {
	switch s {
	case "stdio":
		return TransportStdio, nil
	case "http":
		return TransportHTTP, nil
	default:
		return TransportStdio, fmt.Errorf("unsupported transport type: %s (supported: stdio, http)", s)
	}
}
