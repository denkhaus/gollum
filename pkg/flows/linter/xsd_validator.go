package linter

import (
	_ "embed"
	"sync"

	"github.com/denkhaus/gollum/pkg/flows"
	xsdvalidate "github.com/form3tech-oss/go-xsd-validate"
)

//go:embed schema/flow.xsd
var flowXSD []byte

// XSDValidator validates XML against the embedded XSD schema
type XSDValidator struct {
	handler *xsdvalidate.XsdHandler
	once    sync.Once
	initErr error
}

// xsdValidator is the global XSD validator instance
var xsdValidator = &XSDValidator{}

// InitXSD initializes the XSD validator. Must be called once at application startup.
func InitXSD() error {
	return xsdValidator.init()
}

// CleanupXSD releases XSD resources. Call at application shutdown.
func CleanupXSD() {
	xsdValidator.cleanup()
}

// ValidateXSD validates XML content against the flow XSD schema
func ValidateXSD(xmlContent string) []flows.LinterError {
	return xsdValidator.Validate(xmlContent)
}

// init initializes the XSD handler (called once)
func (v *XSDValidator) init() error {
	v.once.Do(func() {
		// Initialize libxml2
		if err := xsdvalidate.Init(); err != nil {
			v.initErr = err
			return
		}

		// Load the embedded XSD schema
		handler, err := xsdvalidate.NewXsdHandlerMem(flowXSD, xsdvalidate.ParsErrDefault)
		if err != nil {
			v.initErr = err
			return
		}
		v.handler = handler
	})

	return v.initErr
}

// cleanup releases resources
func (v *XSDValidator) cleanup() {
	if v.handler != nil {
		v.handler.Free()
		v.handler = nil
	}
	xsdvalidate.Cleanup()
}

// Validate validates XML content against the XSD schema
func (v *XSDValidator) Validate(xmlContent string) []flows.LinterError {
	if v.handler == nil {
		if err := v.init(); err != nil {
			return []flows.LinterError{{
				Code:    "XSD001",
				Message: "XSD validator not initialized: " + err.Error(),
			}}
		}
	}

	// Validate the XML
	err := v.handler.ValidateMem([]byte(xmlContent), xsdvalidate.ValidErrDefault)
	if err == nil {
		return nil
	}

	// Convert validation errors to linter errors
	var errors []flows.LinterError

	switch e := err.(type) {
	case xsdvalidate.ValidationError:
		for _, valErr := range e.Errors {
			errors = append(errors, flows.LinterError{
				Code:    "XSD002",
				Line:    valErr.Line,
				Column:  1, // XSD validator doesn't provide column
				Message: valErr.Message,
			})
		}
	default:
		errors = append(errors, flows.LinterError{
			Code:    "XSD003",
			Message: "XML validation error: " + err.Error(),
		})
	}

	return errors
}
