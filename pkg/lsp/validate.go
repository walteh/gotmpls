package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	s.debugf(ctx, "validating document: %s", uri)
	s.debugf(ctx, "content:\n%s", content)

	// Convert URI to filesystem path for the document
	docPath, err := uriToPath(uri)
	if err != nil {
		s.debugf(ctx, "document path error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  fmt.Sprintf("invalid document path: %v", err),
		}})
	}
	s.debugf(ctx, "document path: %s", docPath)

	// Parse the template
	info, err := s.parser.Parse(ctx, []byte(content), docPath)
	if err != nil {
		s.debugf(ctx, "parse error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf(ctx, "parsed template info: %+v", info)

	// Use the template file's directory for package analysis
	templateDir := filepath.Dir(docPath)
	s.debugf(ctx, "analyzing package in directory: %s", templateDir)

	// Check if go.mod exists
	modPath := filepath.Join(templateDir, "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		s.debugf(ctx, "go.mod not found at %s: %v", modPath, err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  fmt.Sprintf("no go.mod found in directory: %s", templateDir),
		}})
	}
	s.debugf(ctx, "found go.mod at %s", modPath)

	// Analyze the package to get type information
	registry, err := s.analyzer.AnalyzePackage(ctx, templateDir)
	if err != nil {
		s.debugf(ctx, "package analysis error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf(ctx, "analyzed package registry: %+v", registry)

	// Generate diagnostics
	diagnostics, err := s.generator.Generate(ctx, info, s.validator, registry)
	if err != nil {
		s.debugf(ctx, "diagnostic generation error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf(ctx, "generated diagnostics: %+v", diagnostics)

	// Convert diagnostics to LSP format
	lspDiagnostics := make([]Diagnostic, 0)
	for _, d := range diagnostics.Errors {
		lspDiagnostics = append(lspDiagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: d.Column - 1},
				End:   Position{Line: d.EndLine - 1, Character: d.EndCol - 1},
			},
			Severity: 1, // Error
			Message:  d.Message,
		})
	}

	for _, d := range diagnostics.Warnings {
		lspDiagnostics = append(lspDiagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: d.Column - 1},
				End:   Position{Line: d.EndLine - 1, Character: d.EndCol - 1},
			},
			Severity: 2, // Warning
			Message:  d.Message,
		})
	}

	s.debugf(ctx, "publishing %d diagnostics for %s", len(lspDiagnostics), uri)
	for _, d := range lspDiagnostics {
		severity := "unknown"
		switch d.Severity {
		case 1:
			severity = "error"
		case 2:
			severity = "warning"
		case 3:
			severity = "information"
		case 4:
			severity = "hint"
		}
		s.debugf(ctx, "  - %s at %v: %s", severity, d.Range, d.Message)
	}

	err = s.publishDiagnostics(ctx, uri, lspDiagnostics)
	if err != nil {
		s.debugf(ctx, "error publishing diagnostics: %v", err)
	}
	return nil, err
}
