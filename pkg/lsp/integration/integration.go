package integration

import (
	"context"
	"testing"
	"time"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

// IntegrationTestRunner defines the interface for LSP integration testing
type IntegrationTestRunner interface {
	// Document State Operations
	GetDiagnostics(t *testing.T, uri protocol.DocumentURI, timeout time.Duration) (*protocol.FullDocumentDiagnosticReport, error)
	GetDocumentText(t *testing.T, uri protocol.DocumentURI) (string, error)
	GetFormattedDocument(t *testing.T, ctx context.Context, uri protocol.DocumentURI) (string, error)

	// Navigation & Symbol Operations
	Hover(t *testing.T, ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error)
	GetDefinition(t *testing.T, ctx context.Context, params *protocol.DefinitionParams) ([]*protocol.Location, error)
	GetReferences(t *testing.T, ctx context.Context, params *protocol.ReferenceParams) ([]*protocol.Location, error)
	GetDocumentSymbols(t *testing.T, ctx context.Context, params *protocol.DocumentSymbolParams) ([]*protocol.DocumentSymbol, error)

	// Code Intelligence Operations
	GetCodeActions(t *testing.T, ctx context.Context, params *protocol.CodeActionParams) ([]*protocol.CodeAction, error)
	GetCompletion(t *testing.T, ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error)
	GetSignatureHelp(t *testing.T, ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error)
	GetSemanticTokensFull(t *testing.T, ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error)
	GetSemanticTokensRange(t *testing.T, ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error)

	// Document Modification Operations
	ApplyEdit(t *testing.T, uri protocol.DocumentURI, newContent string, save bool) error
	ApplyRename(t *testing.T, ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error)

	// Lifecycle Operations
	SaveAndQuit() error
	TmpFilePathOf(path string) protocol.DocumentURI
}
