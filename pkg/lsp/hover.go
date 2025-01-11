package lsp

import (
	"context"
	"encoding/json"
	"strings"
	"text/template/parse"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/hover")
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
	}

	// Get document content
	content, ok := s.getDocument(s.normalizeURI(params.TextDocument.URI))
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Parse the template
	info, err := parser.Parse(ctx, []byte(content), params.TextDocument.URI)
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.FromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

	// Find type hint for the position
	hint := FindTypeHintForPosition(info, content, pos)
	if hint == nil {
		return nil, nil
	}

	registry, err := ast.AnalyzePackage(ctx, info.Filename)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	// Get type info
	typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, hint.TypePath, registry)
	if err != nil {
		return nil, errors.Errorf("validating type for hover: %w", err)
	}

	// Find the field at the current position
	field, err := FindFieldAtPosition(info, content, pos)
	if err != nil {
		return nil, errors.Errorf("finding field at position: %w", err)
	}
	if field == nil {
		return nil, nil
	}

	// Get field info using ast.GenerateFieldInfoFromPosition
	fieldInfo, err := ast.GenerateFieldInfoFromPosition(ctx, typeInfo, field)
	if err != nil {
		return nil, errors.Errorf("getting field info: %w", err)
	}

	// Get the field's position in the line
	_, startChar := position.GetLineAndColumn(content, parse.Pos(field.Offset()))

	// Create hover response
	hover := &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: FormatHoverResponse(typeInfo.Name, field.Text(), fieldInfo.Type.String()),
		},
		Range: &Range{
			Start: Position{
				Line:      params.Position.Line,
				Character: startChar - 1, // Convert to 0-based
			},
			End: Position{
				Line:      params.Position.Line,
				Character: startChar - 1 + len(field.Text()),
			},
		},
	}

	return hover, nil
}

func FindTypeHintForPosition(info *parser.TemplateInfo, content string, pos position.RawPosition) *parser.TypeHint {
	if len(info.TypeHints) == 0 {
		return nil
	}

	// Find the type hint that contains the current position
	var lastHint *parser.TypeHint
	for i := range info.TypeHints {
		if position.HasRangeOverlap(pos, info.TypeHints[i].Position) {
			lastHint = &info.TypeHints[i]
		}
	}

	return lastHint
}

func FindFieldAtPosition(info *parser.TemplateInfo, content string, pos position.RawPosition) (position.RawPosition, error) {
	// Find the variable that contains the current position
	for _, variable := range info.Variables {
		if position.HasRangeOverlap(pos, variable.Position) {
			return variable.Position, nil
		}
	}

	return nil, nil
}

func FormatHoverResponse(typeName, fieldPath, fieldType string) string {
	// Remove the leading dot
	if strings.HasPrefix(fieldPath, ".") {
		fieldPath = fieldPath[1:]
	}

	return "**Variable**: " + typeName + "." + fieldPath + "\n**Type**: " + fieldType
}
