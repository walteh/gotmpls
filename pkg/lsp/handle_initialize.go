package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
	"gitlab.com/tozd/go/errors"
)

func (s *Server) handleInitialize(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling initialize request")
	}

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal initialize params: %w", err)
	}

	// Convert workspace URI to filesystem path
	workspacePath, err := uriToPath(params.RootURI)
	if err != nil {
		return nil, errors.Errorf("invalid workspace URI: %w", err)
	}

	s.workspace = workspacePath
	s.debugf(ctx, "workspace path: %s", s.workspace)

	// Schedule a workspace scan after initialization
	go func() {
		if err := s.scanWorkspace(ctx); err != nil {
			s.debugf(ctx, "failed to scan workspace: %v", err)
		}
	}()

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncKind{
				Change: 1, // Incremental
			},
			HoverProvider: true,
			// CompletionProvider: CompletionOptions{
			// 	TriggerCharacters: []string{"."},
			// },
		},
	}, nil
}

func (s *Server) scanWorkspace(ctx context.Context) error {
	// Walk through the workspace and validate all .tmpl files
	return filepath.Walk(s.workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".tmpl" {
			content, err := os.ReadFile(path)
			if err != nil {
				return errors.Errorf("reading template file: %w", err)
			}
			uri := pathToURI(path)
			s.storeDocument(uri, string(content))
			if _, err := s.validateDocument(ctx, uri, string(content)); err != nil {
				s.debugf(ctx, "failed to validate document %s: %v", path, err)
			}
		}
		return nil
	})
}

func pathToURI(path string) string {
	return "file://" + path
}
