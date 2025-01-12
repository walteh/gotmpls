package lsp

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "validating document: %s", uri)
	}

	uri = s.normalizeURI(uri)

	registry, err := ast.AnalyzePackage(ctx, uri)
	if err != nil {
		return nil, errors.Errorf("analyzing package: %w", err)
	}

	nodes, err := parser.Parse(ctx, []byte(content), uri)
	if err != nil {
		return nil, errors.Errorf("parsing template for validation: %w", err)
	}

	diagnostics, err := diagnostic.GetDiagnosticsFromParsed(ctx, nodes, registry)
	if err != nil {
		return nil, errors.Errorf("getting diagnostics: %w", err)
	}

	// Parse the template to get type hints
	// info, err := parser.Parse(ctx, []byte(content), uri)
	// if err != nil {
	// 	return nil, errors.Errorf("parsing template for validation: %w", err)
	// }

	// Add success diagnostic for type hint if present and valid
	// for _, block := range info.Blocks {
	// 	if block.TypeHint != nil {
	// 		if typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, block.TypeHint.TypePath, registry); err == nil {
	// 			// Add success diagnostic
	// 			diagnostics = append(diagnostics, &diagnostic.Diagnostic{
	// 				Message:  "Type hint successfully loaded: " + typeInfo.Name,
	// 				Location: block.TypeHint.Position,
	// 				Severity: diagnostic.SeverityInformation,
	// 			})
	// 		}
	// 	}
	// }

	// Publish diagnostics
	if err := s.publishDiagnostics(ctx, uri, content, diagnostics); err != nil {
		return nil, errors.Errorf("publishing diagnostics: %w", err)
	}

	return nil, nil
}
