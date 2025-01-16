package integration

import (
	"context"
	"testing"

	"github.com/walteh/gotmpls/pkg/lsp/protocol"
)

// IntegrationTestRunner defines the interface for LSP integration testing
type IntegrationTestRunner interface {
	// Document State Operations
	GetDiagnostics(t *testing.T, uri protocol.DocumentURI, severity protocol.DiagnosticSeverity) ([]protocol.Diagnostic, []protocol.RPCMessage)
	GetDocumentText(t *testing.T, uri protocol.DocumentURI) (string, error)
	GetFormattedDocument(t *testing.T, ctx context.Context, uri protocol.DocumentURI) (string, []protocol.RPCMessage)

	// Navigation & Symbol Operations
	Hover(t *testing.T, ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, []protocol.RPCMessage)
	GetDefinition(t *testing.T, ctx context.Context, params *protocol.DefinitionParams) ([]*protocol.Location, []protocol.RPCMessage)
	GetReferences(t *testing.T, ctx context.Context, params *protocol.ReferenceParams) ([]*protocol.Location, []protocol.RPCMessage)
	GetDocumentSymbols(t *testing.T, ctx context.Context, params *protocol.DocumentSymbolParams) ([]*protocol.DocumentSymbol, []protocol.RPCMessage)

	// Code Intelligence Operations
	GetCodeActions(t *testing.T, ctx context.Context, params *protocol.CodeActionParams) ([]*protocol.CodeAction, []protocol.RPCMessage)
	GetCompletion(t *testing.T, ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, []protocol.RPCMessage)
	GetSignatureHelp(t *testing.T, ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, []protocol.RPCMessage)
	GetSemanticTokensFull(t *testing.T, ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, []protocol.RPCMessage)
	GetSemanticTokensRange(t *testing.T, ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, []protocol.RPCMessage)

	// Document Modification Operations
	ApplyEdit(t *testing.T, uri protocol.DocumentURI, newContent string, save bool) []protocol.RPCMessage
	ApplyRename(t *testing.T, ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, []protocol.RPCMessage)

	// Lifecycle Operations
	SaveAndQuit() error
	TmpFilePathOf(path string) protocol.DocumentURI
}
