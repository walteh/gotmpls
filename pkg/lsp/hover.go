package lsp

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/hover"
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

	// // Get document content
	// content, ok := s.getDocument(uri)
	// if !ok {
	// 	return nil, errors.Errorf("document not found: %s", uri)
	// }

	reg, err := ast.AnalyzePackage(ctx, uri)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	content, _, ok := reg.GetTemplateFile(uri)
	if !ok {
		return nil, errors.Errorf("template %s not found, make sure its embeded", uri)
	}

	// // Parse the template
	info, err := parser.Parse(ctx, uri, []byte(content))
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.NewRawPositionFromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

	// hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, func(arg parser.VariableLocationOrType, th parser.TypeHint) []types.Type {
	// 	if arg.Type != nil {
	// 		return []types.Type{arg.Type}
	// 	}

	// 	if arg.Variable != nil {
	// 		// typ := arg.Variable.GetTypePaths(&th)

	// 		args := append([]string{th.LocalTypeName()}, strings.Split(arg.Variable.LongName(), ".")...)
	// 		scope := pkg.Package.Types.Scope().Lookup(args[0])
	// 	HERE:
	// 		for _, typ := range args[1:] {
	// 			if sig, ok := scope.Type().(*types.Struct); ok {
	// 				for i := range sig.NumFields() {
	// 					if sig.Field(i).Name() == typ {
	// 						scope = sig.Field(i)
	// 						break HERE
	// 					}
	// 				}
	// 			}
	// 		}

	// 		if sig, ok := scope.Type().(*types.Signature); ok {
	// 			typs := []types.Type{}
	// 			for i := range sig.Results().Len() {
	// 				typs = append(typs, sig.Results().At(i).Type())
	// 			}
	// 			return typs
	// 		}

	// 		return []types.Type{}
	// 	}

	// 	return []types.Type{}
	// })
	hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, reg)
	if err != nil {
		return nil, errors.Errorf("building hover response: %w", err)
	}

	if hoverInfo == nil {
		return nil, nil
	}

	hovers := make([]Hover, len(hoverInfo.Content))
	for i, hcontent := range hoverInfo.Content {
		hovers[i] = Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: hcontent,
			},
			Range: rangeToLSP(hoverInfo.Position.GetRange(content)),
		}
	}

	// TODO: Return more than one
	if len(hovers) > 0 {
		return &hovers[0], nil
	}

	return nil, nil
}

// rangeToLSP converts a position.Range to an LSP Range
func rangeToLSP(r position.Range) *Range {
	return &Range{
		Start: Position{
			Line:      r.Start.Line,
			Character: r.Start.Character,
		},
		End: Position{
			Line:      r.End.Line,
			Character: r.End.Character,
		},
	}
}
