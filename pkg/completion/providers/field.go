package providers

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// FieldProvider handles field completions
type FieldProvider struct {
	validator types.Validator
	registry  ast.PackageAnalyzer
}

// NewFieldProvider creates a new field completion provider
func NewFieldProvider(validator types.Validator, registry ast.PackageAnalyzer) *FieldProvider {
	return &FieldProvider{
		validator: validator,
		registry:  registry,
	}
}

// GetCompletions returns field completions for a given type
func (p *FieldProvider) GetCompletions(ctx context.Context, typePath string) ([]CompletionItem, error) {
	var completions []CompletionItem

	// Get type info from the validator
	typeInfo, err := p.validator.ValidateType(ctx, typePath, p.registry)
	if err != nil {
		return nil, errors.Errorf("failed to validate type: %w", err)
	}

	// Add all fields from the type
	for name, field := range typeInfo.Fields {
		completions = append(completions, CompletionItem{
			Label:         name,
			Kind:          "field",
			Detail:        field.Type.String(),
			Documentation: "Field of type: " + field.Type.String(),
		})
	}

	return completions, nil
}

// CompletionItem represents a single completion suggestion
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
}
