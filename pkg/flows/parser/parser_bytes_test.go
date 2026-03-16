package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBytes_ValidFlow(t *testing.T) {
	// Test ParseBytes with valid flow XML
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-flow" version="1.0">
    <description>Test flow</description>
    <input>
        <string name="message" required="true" />
    </input>
    <output>
        <string name="result" />
    </output>
    <states>
        <state name="done" initial="true" />
    </states>
</flow>`)

	flow, err := ParseBytes(xmlData)

	assert.NoError(t, err)
	assert.NotNil(t, flow)
	assert.Equal(t, "test-flow", flow.Name)
	assert.Equal(t, "1.0", flow.Version)
	assert.Contains(t, flow.Description, "Test flow")
}

func TestParseBytes_InvalidXML_UnclosedComment(t *testing.T) {
	// Test that ParseBytes catches unclosed comments
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!-- This comment is never closed
<flow name="test" version="1.0">
</flow>`)

	flow, err := ParseBytes(xmlData)

	assert.Error(t, err)
	assert.Nil(t, flow)
	assert.Contains(t, err.Error(), "XML syntax error")
}

func TestParseBytes_InvalidXML_UnclosedTag(t *testing.T) {
	// Test that ParseBytes catches unclosed tags
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<flow name="test" version="1.0">
    <input>
</flow>`)

	flow, err := ParseBytes(xmlData)

	assert.Error(t, err)
	assert.Nil(t, flow)
}

func TestParseBytes_EmptyInput(t *testing.T) {
	// Test that ParseBytes handles empty input
	flow, err := ParseBytes([]byte{})

	assert.Error(t, err)
	assert.Nil(t, flow)
}
