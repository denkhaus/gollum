package parser

import (
	"encoding/xml"
	"os"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Parse reads and parses a flow XML file
func Parse(path string) (*flows.Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var flow flows.Flow
	if err := xml.Unmarshal(data, &flow); err != nil {
		return nil, err
	}

	return &flow, nil
}
