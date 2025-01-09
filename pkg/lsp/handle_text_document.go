package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) handleTextDocumentDidOpen(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/didOpen")
	}

	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.debugf(ctx, "failed to unmarshal didOpen params: %v", err)
		return nil, errors.Errorf("failed to unmarshal didOpen params: %w", err)
	}

	s.debugf(ctx, "storing document in memory: %s", params.TextDocument.URI)
	s.documents.Store(params.TextDocument.URI, params.TextDocument.Text)
	s.debugf(ctx, "document stored successfully, validating...")
	return s.validateDocument(ctx, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) handleTextDocumentDidChange(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/didChange")
	}

	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal didChange params: %w", err)
	}

	// For now, we'll just use the full content sync
	if len(params.ContentChanges) > 0 {
		s.documents.Store(params.TextDocument.URI, params.ContentChanges[0].Text)
		return s.validateDocument(ctx, params.TextDocument.URI, params.ContentChanges[0].Text)
	}

	return nil, nil
}

func (s *Server) handleTextDocumentDidClose(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/didClose")
	}

	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal didClose params: %w", err)
	}

	s.documents.Delete(params.TextDocument.URI)
	return nil, nil
}

func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/hover")
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.debugf(ctx, "failed to unmarshal hover params: %v", err)
		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
	}

	s.debugf(ctx, "looking for document: %s", params.TextDocument.URI)
	// Get the document content
	content, ok := s.documents.Load(params.TextDocument.URI)
	if !ok {
		s.debugf(ctx, "document not found in store: %s", params.TextDocument.URI)
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	text, ok := content.(string)
	if !ok {
		s.debugf(ctx, "invalid document content type: %T", content)
		return nil, errors.Errorf("invalid document content type")
	}

	s.debugf(ctx, "parsing template at position line:%d char:%d", params.Position.Line, params.Position.Character)
	// Parse the template
	tmpl, err := s.parser.Parse(ctx, []byte(text), params.TextDocument.URI)
	if err != nil {
		s.debugf(ctx, "failed to parse template: %v", err)
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	s.debugf(ctx, "template parsed successfully, searching for hover info at position")

	// First, try to find an exact match
	for _, v := range tmpl.Variables {
		s.debugf(ctx, "checking variable: %s at line:%d col:%d end_col:%d (hover at line:%d col:%d)",
			v.Name, v.Line, v.Column, v.EndCol, params.Position.Line+1, params.Position.Character+1)

		// For field access, we need to calculate the actual end column
		actualEndCol := v.EndCol
		if strings.Contains(v.Name, ".") {
			// For field access, the end column should be start column + length of the full name
			actualEndCol = v.Column + len(v.Name)
			s.debugf(ctx, "field access detected, adjusted end_col from %d to %d", v.EndCol, actualEndCol)
			s.debugf(ctx, "field access details - name:%s start_col:%d end_col:%d cursor_col:%d",
				v.Name, v.Column, actualEndCol, params.Position.Character+1)
		}

		// Check if we're within 1 line of the target (parser might be off by one)
		lineDiff := abs(v.Line - (params.Position.Line + 1))
		if lineDiff <= 1 {
			s.debugf(ctx, "line within tolerance (diff:%d) for %s", lineDiff, v.Name)

			// If we're on a different line, adjust column checks
			adjustedStartCol := v.Column
			adjustedEndCol := actualEndCol
			if v.Line != params.Position.Line+1 {
				// If we're on the line after, check from start of line
				if v.Line > params.Position.Line+1 {
					adjustedStartCol = 1
				}
				s.debugf(ctx, "adjusted columns for line difference - start:%d end:%d", adjustedStartCol, adjustedEndCol)
			}

			if adjustedStartCol <= params.Position.Character+1 && // +1 because LSP is 0-based
				adjustedEndCol >= params.Position.Character+1 {
				s.debugf(ctx, "found match for variable: %s (col match: %d <= %d <= %d)",
					v.Name, adjustedStartCol, params.Position.Character+1, adjustedEndCol)
				return createHoverResponse(s, ctx, v)
			}
		}
	}

	// Check functions last
	for _, f := range tmpl.Functions {
		if f.Line == params.Position.Line+1 && // +1 because LSP is 0-based, our parser is 1-based
			f.Column <= params.Position.Character+1 && // +1 for same reason
			f.EndLine == params.Position.Line+1 &&
			f.EndCol >= params.Position.Character+1 {

			s.debugf(ctx, "found function at position: %s (line:%d col:%d)", f.Name, f.Line, f.Column)
			var args []string
			for _, arg := range f.MethodArguments {
				args = append(args, arg.String())
			}
			signature := fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ", "))
			s.debugf(ctx, "function signature: %s", signature)

			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("**Function**: %s\n**Signature**: %s\n**Scope**: %s", f.Name, signature, f.Scope),
				},
				Range: &Range{
					Start: Position{Line: f.Line - 1, Character: f.Column - 1},
					End:   Position{Line: f.EndLine - 1, Character: f.EndCol - 1},
				},
			}, nil
		}
	}

	s.debugf(ctx, "no hover information found at position line:%d char:%d", params.Position.Line, params.Position.Character)
	return nil, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func createHoverResponse(s *Server, ctx context.Context, v parser.VariableLocation) (interface{}, error) {
	var typeInfo string
	if v.MethodArguments != nil && len(v.MethodArguments) > 0 {
		typeInfo = v.MethodArguments[0].String()
		s.debugf(ctx, "variable type info: %s", typeInfo)
	} else {
		typeInfo = "unknown"
		s.debugf(ctx, "no type info available for variable")
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: fmt.Sprintf("**Variable**: %s\n**Type**: %s\n**Scope**: %s", v.Name, typeInfo, v.Scope),
		},
		Range: &Range{
			Start: Position{Line: v.Line - 1, Character: v.Column - 1},
			End:   Position{Line: v.EndLine - 1, Character: v.EndCol - 1},
		},
	}, nil
}

func (s *Server) handleTextDocumentCompletion(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/completion")
	}

	var params CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal completion params: %w", err)
	}

	// TODO: Implement completion
	return nil, nil
}
