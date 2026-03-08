package parser

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Parse reads and parses a flow XML file
func Parse(path string) (*flows.Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Validate common XML syntax issues first for better error messages
	if err := validateCommonXMLErrors(data); err != nil {
		return nil, fmt.Errorf("XML syntax error: %w", err)
	}

	var flow flows.Flow
	if err := xml.Unmarshal(data, &flow); err != nil {
		return nil, err
	}

	return &flow, nil
}
