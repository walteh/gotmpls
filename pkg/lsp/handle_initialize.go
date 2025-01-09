package lsp

import (
	"context"
	"encoding/json"

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

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncKind{
				Change: 1, // Incremental
			},
			HoverProvider: true,
			CompletionProvider: CompletionOptions{
				TriggerCharacters: []string{"."},
			},
		},
	}, nil
}
