package completion

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

// CompletionKind represents the type of completion item
type CompletionKind string

const (
	CompletionKindField    CompletionKind = "field"
	CompletionKindVariable CompletionKind = "variable"
	CompletionKindMethod   CompletionKind = "method"
)

// CompletionItem represents a single completion suggestion
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
}

// GetCompletions returns completion items for a given position in a template
func GetCompletions(ctx context.Context, registry *ast.TypeRegistry, info *parser.TemplateInfo, line, character int, content string) ([]CompletionItem, error) {
	// Create completion context to determine what kind of completion we need
	completionCtx := NewCompletionContext(content, line, character)

	// If we're after a dot and in a template action, we need field completions
	if completionCtx.IsDotCompletion() {
		// Get the type hint for the current position
		hint := findTypeHintForPosition(info, line, character)
		if hint == nil {
			// If we don't have a type hint, return variable completions
			return getVariableCompletions(info), nil
		}

		// Get field completions for the type
		return getFieldCompletions(ctx, registry, hint.TypePath)
	}

	// Otherwise, return variable completions
	return getVariableCompletions(info), nil
}

// getFieldCompletions returns field completions for a given type path
func getFieldCompletions(ctx context.Context, registry *ast.TypeRegistry, typePath string) ([]CompletionItem, error) {
	// Get type info from registry
	typeInfo, err := registry.ValidateType(ctx, typePath)
	if err != nil {
		return nil, errors.Errorf("getting type info: %w", err)
	}

	// Create completion item for the type itself
	return []CompletionItem{
		{
			Label:         typePath,
			Kind:          string(CompletionKindField),
			Detail:        typeInfo.Name,
			Documentation: "Field of type: " + typeInfo.Name,
		},
	}, nil
}

// getVariableCompletions returns variable completions from template info
func getVariableCompletions(info *parser.TemplateInfo) []CompletionItem {
	var completions []CompletionItem

	// Add all variables from the template info
	for _, v := range info.Variables {
		completions = append(completions, CompletionItem{
			Label:         v.Name(),
			Kind:          string(CompletionKindVariable),
			Detail:        "Template variable",
			Documentation: "Variable from template scope: " + v.Scope,
		})
	}

	return completions
}

// findTypeHintForPosition finds the type hint that applies to the given position
func findTypeHintForPosition(info *parser.TemplateInfo, line, character int) *parser.TypeHint {
	if len(info.TypeHints) == 0 {
		return nil
	}

	// Find the type hint that matches the current position
	for _, hint := range info.TypeHints {
		// For now, just check if the type path matches the position text
		// TODO: Implement proper position-based type hint lookup
		if hint.Position.Text() == hint.TypePath {
			return &hint
		}
	}

	return nil
}
