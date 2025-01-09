package lsp

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
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
