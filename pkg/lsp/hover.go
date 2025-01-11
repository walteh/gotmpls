package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"
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

	zerolog.Ctx(ctx).Debug().Msgf("hover request received: %+v", params)

	uri := s.normalizeURI(params.TextDocument.URI)

	// Get document content
	content, ok := s.getDocument(uri)
	if !ok {
		return nil, errors.Errorf("document not found: %s", uri)
	}

	// Parse the template
	info, err := parser.Parse(ctx, []byte(content), uri)
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.NewRawPositionFromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

	registry, err := ast.AnalyzePackage(ctx, info.Filename)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	for _, block := range info.Blocks {

		if block.TypeHint == nil {
			continue
		}

		zerolog.Ctx(ctx).Debug().Msgf("checking block %s against type hint %s (vars: %d)", block.Name, block.TypeHint.TypePath, len(block.Variables))

		typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, block.TypeHint.TypePath, registry)
		if err != nil {
			return nil, errors.Errorf("validating type for hover: %w", err)
		}

		for _, variable := range block.Variables {
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", pos.Text, pos.Offset, variable.Position.Text, variable.Position.Offset)
			if pos.HasRangeOverlapWith(variable.Position) {

				zerolog.Ctx(ctx).Debug().Msgf("variable %s at %v overlaps with position %v", variable.Name(), variable.Position, pos)

				// Get field info
				fieldInfo, err := ast.GenerateFieldInfoFromPosition(ctx, typeInfo, variable.Position)
				if err != nil {
					return nil, errors.Errorf("getting field info: %w", err)
				}

				ranged := RangeFromGoTmplTyperRange(variable.Position.GetRange(content))

				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: FormatHoverResponse(typeInfo.Name, variable.Position.Text, fieldInfo.Type.String()),
					},
					Range: &ranged,
				}, nil
			}
		}

		for _, function := range block.Functions {
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", pos.Text, pos.Offset, function.Position.Text, function.Position.Offset)
			if pos.HasRangeOverlapWith(function.Position) {
				zerolog.Ctx(ctx).Debug().Msgf("function %s at %v overlaps with position %v", function.Name(), function.Position, pos)

				ranged := RangeFromGoTmplTyperRange(function.Position.GetRange(content))
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: FormatHoverResponse(typeInfo.Name, function.Position.Text, function.String()),
					},
					Range: &ranged,
				}, nil
			}
		}

	}

	// // Find the field at the current position
	// field, err := FindFieldAtPosition(info, content, pos)
	// if err != nil {
	// 	return nil, errors.Errorf("finding field at position: %w", err)
	// }
	// if field == nil {
	// 	return nil, nil
	// }

	// // Get field info using ast.GenerateFieldInfoFromPosition
	// fieldInfo, err := ast.GenerateFieldInfoFromPosition(ctx, typeInfo, field)
	// if err != nil {
	// 	return nil, errors.Errorf("getting field info: %w", err)
	// }

	// // Get the field's position in the line
	// _, startChar := position.GetLineAndColumn(content, parse.Pos(field.Offset()))

	// // Create hover response
	// hover := &Hover{
	// 	Contents: MarkupContent{
	// 		Kind:  "markdown",
	// 		Value: FormatHoverResponse(typeInfo.Name, field.Text(), fieldInfo.Type.String()),
	// 	},
	// 	Range: &Range{
	// 		Start: Position{
	// 			Line:      params.Position.Line,
	// 			Character: startChar - 1, // Convert to 0-based
	// 		},
	// 		End: Position{
	// 			Line:      params.Position.Line,
	// 			Character: startChar - 1 + len(field.Text()),
	// 		},
	// 	},
	// }

	return nil, nil
}

// func FindTypeHintForPosition(info *parser.TemplateInfo, content string, pos position.RawPosition) *parser.TypeHint {
// 	if len(info.TypeHints) == 0 {
// 		return nil
// 	}

// 	// Find the type hint that contains the current position
// 	var lastHint *parser.TypeHint
// 	for i := range info.TypeHints {
// 		if position.HasRangeOverlap(pos, info.TypeHints[i].Position) {
// 			lastHint = &info.TypeHints[i]
// 		}
// 	}

// 	return lastHint
// }

// func FindFieldAtPosition(info *parser.TemplateInfo, content string, pos position.RawPosition) (position.RawPosition, error) {
// 	// Find the variable that contains the current position
// 	for _, variable := range info.Variables {
// 		if position.HasRangeOverlap(pos, variable.Position) {
// 			return variable.Position, nil
// 		}
// 	}

// 	return nil, nil
// }

func FormatHoverResponse(typeName, fieldPath, fieldType string) string {
	fieldPath = strings.TrimPrefix(fieldPath, ".")

	return "**Variable**: " + typeName + "." + fieldPath + "\n**Type**: " + fieldType
}
