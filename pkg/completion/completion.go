package completion

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/completion/providers"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// Provider handles template completions
type Provider struct {
	fieldProvider    *providers.FieldProvider
	variableProvider *providers.VariableProvider
}

// NewProvider creates a new completion provider
func NewProvider(validator types.Validator, registry ast.PackageAnalyzer) *Provider {
	return &Provider{
		fieldProvider:    providers.NewFieldProvider(validator, registry),
		variableProvider: providers.NewVariableProvider(),
	}
}

// GetCompletions returns completion items for a given position in a template
func (p *Provider) GetCompletions(ctx context.Context, info *parser.TemplateInfo, line, character int, content string) ([]providers.CompletionItem, error) {
	// If we have a type hint, get field completions
	if len(info.TypeHints) > 0 {
		// For now, just use the first type hint
		hint := info.TypeHints[0]
		completions, err := p.fieldProvider.GetCompletions(ctx, hint.TypePath)
		if err != nil {
			return nil, errors.Errorf("failed to get field completions: %w", err)
		}
		return completions, nil
	}

	// Otherwise, get variable completions
	return p.variableProvider.GetCompletions(info), nil
}
