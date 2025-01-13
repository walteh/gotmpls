package lsp

import (
	"context"
	"encoding/json"
	"go/types"

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

	// Get document content
	content, ok := s.getDocument(uri)
	if !ok {
		return nil, errors.Errorf("document not found: %s", uri)
	}

	reg, err := ast.AnalyzePackage(ctx, uri)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	content, pkg, ok := reg.GetTemplateFile(uri)
	if !ok {
		return nil, errors.Errorf("template %s not found in package %s, make sure its embeded", uri, pkg.Package.PkgPath)
	}

	// // Parse the template
	info, err := parser.Parse(ctx, uri, []byte(content))
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.NewRawPositionFromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

	hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, func(arg parser.VariableLocationOrType) []types.Type {
		if arg.Type != nil {
			return []types.Type{arg.Type}
		}

		if arg.Variable != nil {
			// we know its a signature
			typ := pkg.Package.Types.Scope().Lookup(arg.Variable.Name()).Type()
			sig, ok := typ.(*types.Signature)
			if !ok {
				return []types.Type{}
			}
			out := []types.Type{}
			for i := range sig.Results().Len() {
				out = append(out, sig.Results().At(i).Type())
			}
			return out
		}

		return []types.Type{}
	})
	if err != nil {
		return nil, errors.Errorf("building hover response: %w", err)
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
