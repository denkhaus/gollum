package linter

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// XMLStructureChecker validates raw XML structure for invalid elements
type XMLStructureChecker struct{}

// NewXMLStructureChecker creates a new XML structure checker
func NewXMLStructureChecker() *XMLStructureChecker {
	return &XMLStructureChecker{}
}

// CheckRawXML validates raw XML content for structural issues
func (x *XMLStructureChecker) CheckRawXML(xmlContent string, result *flows.LinterResult) {
	// Check for <computed> elements inside <context> blocks
	x.checkComputedInBlock(xmlContent, "context", result)
	// Check for <computed> elements inside <output> blocks
	x.checkComputedInBlock(xmlContent, "output", result)
}

// checkComputedInBlock checks for <computed> elements nested inside a block
func (x *XMLStructureChecker) checkComputedInBlock(xmlContent, blockName string, result *flows.LinterResult) {
	// Find all occurrences of <block>...</block>
	blockStart := 0
	for {
		startTag := fmt.Sprintf("<%s", blockName)
		endTag := fmt.Sprintf("</%s>", blockName)

		// Find next block start
		startIdx := strings.Index(xmlContent[blockStart:], startTag)
		if startIdx == -1 {
			break
		}
		startIdx += blockStart

		// Skip if it's a closing tag or self-closing
		if strings.HasPrefix(xmlContent[startIdx:], "</") || strings.HasPrefix(xmlContent[startIdx:], "<"+blockName+" />") {
			blockStart = startIdx + len(blockName)
			continue
		}

		// Find the matching end tag (naive approach - assumes no nested blocks with same name)
		endIdx := strings.Index(xmlContent[startIdx:], endTag)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + len(endTag)

		// Extract block content
		blockContent := xmlContent[startIdx:endIdx]

		// Check for <computed> elements inside
		if strings.Contains(blockContent, "<computed") {
			// Extract computed element names for better error messages
			computedStart := 0
			for {
				compStart := strings.Index(blockContent[computedStart:], "<computed")
				if compStart == -1 {
					break
				}
				compStart += computedStart

				// Find the end of this computed element
				compEnd := strings.Index(blockContent[compStart:], "/>")
				if compEnd == -1 {
					compEnd = strings.Index(blockContent[compStart:], ">")
				}
				if compEnd == -1 {
					break
				}
				compEnd += compStart + 2

				computedElement := blockContent[compStart:compEnd]

				// Extract name attribute
				name := x.extractAttribute(computedElement, "name")
				if name == "" {
					name = "(unknown)"
				}

				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrComputedInContainer,
					Message: fmt.Sprintf("invalid element: <computed> must be at top-level, not nested inside <%s>. Found field '%s' - move it to <computed> block at flow level", blockName, name),
				})

				computedStart = compEnd
			}
		}

		blockStart = endIdx
	}
}

// extractAttribute extracts an attribute value from an XML element string
func (x *XMLStructureChecker) extractAttribute(element, attrName string) string {
	prefix := attrName + `="`
	startIdx := strings.Index(element, prefix)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(prefix)

	endIdx := strings.Index(element[startIdx:], `"`)
	if endIdx == -1 {
		return ""
	}
	endIdx += startIdx

	return element[startIdx:endIdx]
}

// rawFlow represents the raw XML structure for validation
type rawFlow struct {
	XMLName  xml.Name `xml:"flow"`
	Context  []byte   `xml:",innerxml"`
	Output   []byte   `xml:",innerxml"`
	Computed []byte   `xml:",innerxml"`
}

// DecodeRawForCheck decodes XML just for structural checking
func (x *XMLStructureChecker) DecodeRawForCheck(reader io.Reader) (*rawFlow, error) {
	var raw rawFlow
	decoder := xml.NewDecoder(reader)
	err := decoder.Decode(&raw)
	if err != nil {
		return nil, err
	}
	return &raw, nil
}
