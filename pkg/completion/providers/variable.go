package providers

import (
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

// VariableProvider handles variable completions
type VariableProvider struct{}

// NewVariableProvider creates a new variable completion provider
func NewVariableProvider() *VariableProvider {
	return &VariableProvider{}
}

// GetCompletions returns variable completions from template info
func (p *VariableProvider) GetCompletions(info *parser.TemplateInfo) []CompletionItem {
	var completions []CompletionItem

	// Add all variables from the template info
	for _, v := range info.Variables {
		completions = append(completions, CompletionItem{
			Label:         v.Name,
			Kind:          "variable",
			Detail:        "Template variable",
			Documentation: "Variable from template scope: " + v.Scope,
		})
	}

	return completions
}
