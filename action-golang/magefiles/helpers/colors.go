package helpers

import "fmt"

// Color constants for terminal output
type Color string

const (
	ColorReset  Color = "\033[0m"
	ColorBlue   Color = "\033[0;34m"
	ColorGreen  Color = "\033[0;32m"
	ColorRed    Color = "\033[0;31m"
	ColorYellow Color = "\033[0;33m"
)

// Printer handles colored terminal output
type Printer struct {
	useColors bool
}

// NewPrinter creates a new printer with color support
func NewPrinter() *Printer {
	return &Printer{useColors: true}
}

// Printf prints formatted text with the specified color
func (p *Printer) Printf(color Color, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if p.useColors {
		fmt.Fprintf(stdout, "%s%s%s\n", color, msg, ColorReset)
	} else {
		fmt.Fprintln(stdout, msg)
	}
}

// Info prints an informational message (blue)
func (p *Printer) Info(format string, args ...interface{}) {
	p.Printf(ColorBlue, format, args...)
}

// Success prints a success message (green)
func (p *Printer) Success(format string, args ...interface{}) {
	p.Printf(ColorGreen, format, args...)
}

// Error prints an error message (red)
func (p *Printer) Error(format string, args ...interface{}) {
	p.Printf(ColorRed, format, args...)
}

// Warning prints a warning message (yellow)
func (p *Printer) Warning(format string, args ...interface{}) {
	p.Printf(ColorYellow, format, args...)
}

// Plain prints text without color
func (p *Printer) Plain(format string, args ...interface{}) {
	fmt.Fprintf(stdout, format+"\n", args...)
}

// Global printer instance
var print = NewPrinter()
