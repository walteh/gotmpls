package lsp

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "validating document: %s", uri)
	}

	// Parse the template
	info, err := s.server.parser.Parse(ctx, []byte(content), uri)
	if err != nil {
		return nil, errors.Errorf("parsing template for validation: %w", err)
	}

	// Get type hints
	if len(info.TypeHints) == 0 {
		return nil, nil
	}

	// Validate each type hint
	var diagnostics []Diagnostic
	for _, hint := range info.TypeHints {
		// Validate type
		typeInfo, err := s.server.validator.ValidateType(ctx, hint.TypePath)
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
			if _, err := s.server.validator.ValidateField(ctx, hint.TypePath, v.Name()); err != nil {
				startLine, startCol, endLine, endCol := parser.GetLineColumnRange(content, v.Position)
				diagnostics = append(diagnostics, Diagnostic{
					Range: Range{
						Start: Position{Line: startLine - 1, Character: startCol - 1},
						End:   Position{Line: endLine - 1, Character: endCol - 1},
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
