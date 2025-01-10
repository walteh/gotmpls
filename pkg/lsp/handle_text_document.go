package lsp

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/completion"
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
	s.storeDocument(params.TextDocument.URI, params.TextDocument.Text)
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
		s.storeDocument(params.TextDocument.URI, params.ContentChanges[0].Text)
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

	s.documents.Delete(s.normalizeURI(params.TextDocument.URI))
	return nil, nil
}

func (s *Server) handleCompletion(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal completion params: %w", err)
	}

	// Get document content
	content, ok := s.getDocument(s.normalizeURI(params.TextDocument.URI))
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Parse the template
	info, err := s.server.parser.Parse(ctx, []byte(content), params.TextDocument.URI)
	if err != nil {
		return nil, errors.Errorf("parsing template for completion: %w", err)
	}

	// Get registry from analyzer
	registry, err := s.server.analyzer.AnalyzePackage(ctx, s.workspace)
	if err != nil {
		return nil, errors.Errorf("analyzing package: %w", err)
	}

	// Get completions
	completions, err := completion.GetCompletions(ctx, registry, info, params.Position.Line+1, params.Position.Character+1, content)
	if err != nil {
		return nil, errors.Errorf("getting completions: %w", err)
	}

	// Convert to LSP completion items
	items := make([]CompletionItem, len(completions))
	for i, c := range completions {
		items[i] = CompletionItem{
			Label:  c.Label,
			Kind:   s.completionKindToLSP(c.Kind),
			Detail: c.Detail,
			Documentation: &MarkupContent{
				Kind:  "markdown",
				Value: c.Documentation,
			},
		}
	}

	return items, nil
}

func (s *Server) completionKindToLSP(kind string) CompletionItemKind {
	switch kind {
	case string(completion.CompletionKindField):
		return CompletionItemKindField
	case string(completion.CompletionKindVariable):
		return CompletionItemKindVariable
	case string(completion.CompletionKindMethod):
		return CompletionItemKindMethod
	default:
		return CompletionItemKindText
	}
}
