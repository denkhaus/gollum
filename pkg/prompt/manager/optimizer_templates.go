// Package manager provides optimizer template loading and rendering methods.
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

// RenderOptimizerGradientPrompt renders the gradient reflection prompt with the provided data.
func (p *promptManager) RenderOptimizerGradientPrompt(data map[string]string) (string, error) {
	return p.renderOptimizerTemplate("optimizergradientprompt", data)
}

// RenderOptimizerGradientMetaprompt renders the gradient metaprompt with the provided data.
func (p *promptManager) RenderOptimizerGradientMetaprompt(data map[string]string) (string, error) {
	return p.renderOptimizerTemplate("optimizergradientmetaprompt", data)
}

// RenderOptimizerMetaprompt renders the meta-prompt strategy template with the provided data.
func (p *promptManager) RenderOptimizerMetaprompt(data map[string]string) (string, error) {
	return p.renderOptimizerTemplate("optimizermetapromptprompt", data)
}

// RenderOptimizerPromptMemory renders the prompt memory strategy template with the provided data.
func (p *promptManager) RenderOptimizerPromptMemory(data map[string]string) (string, error) {
	return p.renderOptimizerTemplate("optimizerpromptmemory", data)
}

// renderOptimizerTemplate loads and renders an optimizer template with the provided data.
func (p *promptManager) renderOptimizerTemplate(templateName string, data map[string]string) (string, error) {
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

	// Execute the template with data
	var buf strings.Builder
	if err := namedTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute optimizer template %q: %w", templateName, err)
	}

	return buf.String(), nil
}

// loadOptimizerTemplate loads an optimizer template from the embedded filesystem without rendering.
// This returns the raw template content for compatibility.
func (p *promptManager) loadOptimizerTemplate(templateName string) (string, error) {
	return p.renderOptimizerTemplate(templateName, nil)
}
