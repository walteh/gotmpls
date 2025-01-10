package diagnostic

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/bridge"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

// Diagnostic represents a diagnostic message
type Diagnostic struct {
	Message  string
	Location parser.RawPosition
}

// DiagnosticProvider provides diagnostic information for templates
type DiagnosticProvider struct {
	registry *ast.Registry
}

// NewDiagnosticProvider creates a new DiagnosticProvider
func NewDiagnosticProvider(registry *ast.Registry) *DiagnosticProvider {
	return &DiagnosticProvider{
		registry: registry,
	}
}

// GetDiagnostics returns diagnostic information for a template
func (p *DiagnosticProvider) GetDiagnostics(ctx context.Context, template string, typePath string) ([]*Diagnostic, error) {
	// Parse the template
	templateParser := parser.NewDefaultTemplateParser()
	nodes, err := templateParser.Parse(ctx, []byte(template), "template.tmpl")
	if err != nil {
		return nil, errors.Errorf("parsing template: %w", err)
	}

	// Get type information
	typeInfo, err := p.registry.ValidateType(ctx, typePath)
	if err != nil {
		return nil, errors.Errorf("validating type: %w", err)
	}

	// Check each node
	var diagnostics []*Diagnostic
	for _, variable := range nodes.Variables {
		// Validate field access
		_, err := bridge.ValidateField(ctx, typeInfo, variable.Position)
		if err != nil {
			diagnostics = append(diagnostics, &Diagnostic{
				Message:  err.Error(),
				Location: variable.Position,
			})
		}
	}

	return diagnostics, nil
}
