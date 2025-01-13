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
	Severity int
}

// Severity levels for diagnostics using bit flags
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// Example usage:
// Severity: SeverityInformation | SeverityHint  // Combines both severities

// GetDiagnosticsFromParsed returns diagnostic information for a parsed template
func GetDiagnosticsFromParsed(ctx context.Context, nodes *parser.ParsedTemplateFile, registry *ast.Registry) ([]*Diagnostic, error) {

	var diagnostics []*Diagnostic
	for _, block := range nodes.Blocks {
		if block.TypeHint == nil {
			continue
		} else {
			// green happy underline for successful load
			diagnostics = append(diagnostics, &Diagnostic{
				Message:  "type hint successfully loaded: " + block.TypeHint.TypePath,
				Location: block.TypeHint.Position,
				Severity: SeverityInformation,
			})
		}

		// Get type information
		typeInfo, err := ast.BuildTypeHintDefinitionFromRegistry(ctx, block.TypeHint.TypePath, registry)
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
					Severity: SeverityError,
				})
			}
		}

		// Validate function calls
		for _, functionCall := range block.Functions {
			_, err := ast.GenerateFunctionCallInfoFromPosition(ctx, functionCall.Position)
			if err != nil {
				diagnostics = append(diagnostics, &Diagnostic{
					Message:  err.Error(),
					Location: functionCall.Position,
					Severity: SeverityError,
				})
			}
		}
	}

	return diagnostics, nil

}

// GetDiagnostics returns diagnostic information for a template
func GetDiagnostics(ctx context.Context, template string, registry *ast.Registry) ([]*Diagnostic, error) {
	// Parse the template
	nodes, err := parser.Parse(ctx, "template.tmpl", []byte(template))
	if err != nil {
		return nil, errors.Errorf("parsing template: %w", err)
	}

	diagnostics, err := GetDiagnosticsFromParsed(ctx, nodes, registry)
	if err != nil {
		return nil, errors.Errorf("getting diagnostics from parsed template: %w", err)
	}

	return diagnostics, nil
}
