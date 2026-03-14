package flow

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// MigrateCommand returns the migrate command
func MigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate flow files to latest schema version",
		Description: `Migrate flow files from deprecated syntax to current schema.

Currently supports:
- Migrating <context><computed> to top-level <computed>
- Converting 'when' attribute to 'eval' attribute

Example:
  gollum flow migrate .gollum/flows/examples/old-flow.xml`,
		ArgsUsage: "<flow-file>",
		Action:     runMigrate,
	}
}

func runMigrate(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("expected exactly one argument: <flow-file>")
	}

	flowPath := cmd.Args().First()

	// Read the flow file
	data, err := os.ReadFile(flowPath)
	if err != nil {
		return fmt.Errorf("failed to read flow file: %w", err)
	}

	// Parse XML
	var flow Flow
	if err := xml.Unmarshal(data, &flow); err != nil {
		return fmt.Errorf("failed to parse flow XML: %w", err)
	}

	// Check if migration is needed
	migrated := false

	// Migrate context computed fields to top-level computed
	if flow.Context != nil && len(flow.Context.Computed) > 0 {
		fmt.Printf("Migrating %d computed fields from context to top-level\n", len(flow.Context.Computed))

		if flow.Computed == nil {
			flow.Computed = &ComputedBlock{}
		}

		for _, oldComputed := range flow.Context.Computed {
			newField := ComputedFieldDef{
				Name: oldComputed.Name,
				Type: oldComputed.Type,
				Eval: oldComputed.When, // Convert 'when' to 'eval'
			}
			flow.Computed.Fields = append(flow.Computed.Fields, newField)
			fmt.Printf("  - Migrated: %s (type: %s, eval: %s)\n", oldComputed.Name, oldComputed.Type, oldComputed.When)
		}

		// Clear context computed fields
		flow.Context.Computed = nil
		migrated = true
	}

	if !migrated {
		fmt.Println("Flow is already up-to-date, no migration needed.")
		return nil
	}

	// Marshal back to XML with proper indentation
	output, err := xml.MarshalIndent(flow, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal flow XML: %w", err)
	}

	// Add XML header
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	output = append([]byte(xmlHeader), output...)

	// Create backup
	backupPath := flowPath + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup at %s: %w", backupPath, err)
	}
	fmt.Printf("\nCreated backup: %s\n", backupPath)

	// Write migrated flow
	if err := os.WriteFile(flowPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write migrated flow: %w", err)
	}

	fmt.Printf("\nSuccessfully migrated flow: %s\n", flowPath)

	// Show relative paths for display
	if relPath, err := filepath.Rel(".", flowPath); err == nil {
		flowPath = relPath
	}
	if relBackup, err := filepath.Rel(".", backupPath); err == nil {
		backupPath = relBackup
	}

	fmt.Printf("\nFiles modified:\n")
	fmt.Printf("  - Original backed up to: %s\n", backupPath)
	fmt.Printf("  - Migrated flow written to: %s\n", flowPath)

	return nil
}

// Flow represents the XML structure for migration
type Flow struct {
	XMLName  xml.Name       `xml:"flow"`
	Name     string         `xml:"name,attr"`
	Version  string         `xml:"version,attr"`
	Context  *ContextBlock  `xml:"context"`
	Computed *ComputedBlock `xml:"computed"`
	States   []interface{}  `xml:"states>state"`
}

// ContextBlock represents the context section
type ContextBlock struct {
	Computed []OldComputedField `xml:"computed"`
}

// ComputedBlock represents the computed section
type ComputedBlock struct {
	Fields []ComputedFieldDef `xml:"field"`
}

// OldComputedField represents deprecated context computed field
type OldComputedField struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	When string `xml:"when,attr"`
}

// ComputedFieldDef represents a computed field with eval attribute
type ComputedFieldDef struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Eval string `xml:"eval,attr"`
}
