package lsp

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "validating document: %s", uri)
	}

	// Parse the template
	info, err := parser.Parse(ctx, []byte(content), uri)
	if err != nil {
		return nil, errors.Errorf("parsing template for validation: %w", err)
	}

	registry, err := ast.AnalyzePackage(ctx, info.Filename)
	if err != nil {
		return nil, errors.Errorf("analyzing package: %w", err)
	}

	// Get type hints
	if len(info.TypeHints) == 0 {
		return nil, nil
	}

	// Validate each type hint
	var diagnostics []Diagnostic
	for _, hint := range info.TypeHints {
		// Validate type
		typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, hint.TypePath, registry)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: 0, Character: len(hint.TypePath)},
				},
				Severity: int(SeverityError),
				Message:  "Invalid type: " + err.Error(),
			})
			continue
		}

		// Validate variables
		for _, v := range info.Variables {
			if _, err := ast.GenerateFieldInfoFromPosition(ctx, typeInfo, v.Position); err != nil {
				loc := position.GetLocation(v.Position, content)
				diagnostics = append(diagnostics, Diagnostic{
					Range: Range{
						Start: Position{Line: loc.Start.Line - 1, Character: loc.Start.Character - 1},
						End:   Position{Line: loc.End.Line - 1, Character: loc.End.Character - 1},
					},
					Severity: int(SeverityError),
					Message:  "Invalid field: " + err.Error(),
				})
			}
		}

		// Add type info to diagnostics
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: len(hint.TypePath)},
			},
			Severity: int(SeverityInformation),
			Message:  "Type: " + typeInfo.Name,
		})
	}

	// Publish diagnostics
	if err := s.publishDiagnostics(ctx, uri, diagnostics); err != nil {
		return nil, errors.Errorf("publishing diagnostics: %w", err)
	}

	return nil, nil
}
