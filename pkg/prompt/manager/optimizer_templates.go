// Package manager provides optimizer template loading methods.
package manager

import (
	"fmt"
	"strings"
	"text/template"
)

// GetOptimizerGradientPrompt returns the gradient strategy reflection prompt template.
func (p *promptManager) GetOptimizerGradientPrompt() (string, error) {
	return p.loadOptimizerTemplate("optimizergradientprompt")
}

// GetOptimizerGradientMetaprompt returns the gradient strategy metaprompt template.
func (p *promptManager) GetOptimizerGradientMetaprompt() (string, error) {
	return p.loadOptimizerTemplate("optimizergradientmetaprompt")
}

// GetOptimizerMetaprompt returns the meta-prompt strategy template.
func (p *promptManager) GetOptimizerMetaprompt() (string, error) {
	return p.loadOptimizerTemplate("optimizermetapromptprompt")
}

// GetOptimizerPromptMemory returns the prompt memory strategy template.
func (p *promptManager) GetOptimizerPromptMemory() (string, error) {
	return p.loadOptimizerTemplate("optimizerpromptmemory")
}

// loadOptimizerTemplate loads an optimizer template from the embedded filesystem.
func (p *promptManager) loadOptimizerTemplate(templateName string) (string, error) {
	// Read all template files from embedded FS
	templateFiles, err := promptTemplates.ReadDir("templates")
	if err != nil {
		return "", fmt.Errorf("failed to read templates directory: %w", err)
	}

	// Build a combined template string from all .md files
	var templateContent strings.Builder
	for _, file := range templateFiles {
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		content, err := promptTemplates.ReadFile("templates/" + file.Name())
		if err != nil {
			return "", fmt.Errorf("failed to read template file %s: %w", file.Name(), err)
		}
		templateContent.Write(content)
	}

	// Parse the combined templates
	tmpl, err := template.New("optimizer").Parse(templateContent.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse optimizer templates: %w", err)
	}

	// Look up the specific named template
	namedTmpl := tmpl.Lookup(templateName)
	if namedTmpl == nil {
		return "", fmt.Errorf("optimizer template %q not found", templateName)
	}

	// Execute the template to get the raw content
	var buf strings.Builder
	if err := namedTmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute optimizer template %q: %w", templateName, err)
	}

	return buf.String(), nil
}
