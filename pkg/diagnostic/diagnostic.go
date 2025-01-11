package diagnostic

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// Diagnostic represents a diagnostic message
type Diagnostic struct {
	Message  string
	Location position.RawPosition
}

// GetDiagnostics returns diagnostic information for a template
func GetDiagnostics(ctx context.Context, template string, registry *ast.Registry) ([]*Diagnostic, error) {
	// Parse the template
	nodes, err := parser.Parse(ctx, []byte(template), "template.tmpl")
	if err != nil {
		return nil, errors.Errorf("parsing template: %w", err)
	}

	// Check each node
	var diagnostics []*Diagnostic
	for _, block := range nodes.Blocks {
		if block.TypeHint == nil {
			continue
		}

		// Get type information
		typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, block.TypeHint.TypePath, registry)
		if err != nil {
			return nil, errors.Errorf("validating type: %w", err)
		}

		for _, variable := range block.Variables {

			// Validate field access
			_, err = ast.GenerateFieldInfoFromPosition(ctx, typeInfo, variable.Position)
			if err != nil {
				diagnostics = append(diagnostics, &Diagnostic{
					Message:  err.Error(),
					Location: variable.Position,
				})
			}
		}
	}

	return diagnostics, nil
}
