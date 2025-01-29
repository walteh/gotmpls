package lsp

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/ast"
	"github.com/walteh/gotmpls/pkg/diagnostic"
	"github.com/walteh/gotmpls/pkg/hover"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/semtok"
	"gitlab.com/tozd/go/errors"
	"gopkg.in/fsnotify.v1"
)

// normalizeURI ensures consistent URI handling by removing the file:// prefix if present
// and converting to a clean path
func normalizeURI(uri string) string {
	uri = strings.TrimPrefix(uri, "file://")
	// remove the file:/private prefix
	uri = strings.TrimPrefix(uri, "file:")
	return uri
}

// Server represents an LSP server instance
type Server struct {
	// Document management
	documents *DocumentManager

	// Workspace management
	workspace          string
	workspaceFSWatcher *fsnotify.Watcher

	// Server state
	initialized bool
	shutdown    bool

	// Server identification
	id    string
	debug bool

	// LSP capabilities
	clientCapabilities protocol.ClientCapabilities
	serverCapabilities protocol.ServerCapabilities

	// Context management
	cancelFuncs *sync.Map // map[string]context.CancelFunc

	// LSP client for notifications
	callbackClient protocol.Client
}

func NewServer(ctx context.Context) *Server {
	return &Server{
		id:          xid.New().String(),
		documents:   NewDocumentManager(),
		cancelFuncs: &sync.Map{},
		debug:       false, // Disabled debug mode
	}
}

func (me *Server) SetCallbackClient(client protocol.Client) {
	me.callbackClient = client
}

func (me *Server) Documents() *DocumentManager {
	return me.documents
}

// Required interface methods
func (s *Server) Progress(ctx context.Context, params *protocol.ProgressParams) error {
	return nil // Not implemented yet
}

func (s *Server) SetTrace(ctx context.Context, params *protocol.SetTraceParams) error {
	return nil // Not implemented yet
}

func (s *Server) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ResolveCodeAction(ctx context.Context, params *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ResolveCodeLens(ctx context.Context, params *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ResolveCompletionItem(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ResolveDocumentLink(ctx context.Context, params *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Exit(ctx context.Context) error {
	return nil // Not implemented yet
}

func (s *Server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("initializing server")

	// Store client capabilities
	s.clientCapabilities = params.Capabilities
	logger.Debug().
		Interface("semantic_tokens", s.clientCapabilities.TextDocument.SemanticTokens).
		Interface("workspace_semantic_tokens", s.clientCapabilities.Workspace.SemanticTokens).
		Msg("received client capabilities")

	// Store server capabilities
	s.serverCapabilities = protocol.ServerCapabilities{
		TextDocumentSync: &protocol.Or_ServerCapabilities_textDocumentSync{
			Value: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.Incremental,
			},
		},
		HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
			Value: true,
		},
		CompletionProvider: &protocol.CompletionOptions{
			WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
				WorkDoneProgress: true,
			},
			TriggerCharacters: []string{".", ":", " "},
		},
	}

	return &protocol.InitializeResult{
		Capabilities: s.serverCapabilities,
	}, nil
}

func (s *Server) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("server initialized")

	// Only register semantic tokens if the client supports dynamic registration
	if s.clientCapabilities.TextDocument.SemanticTokens.DynamicRegistration {
		logger.Debug().Msg("client supports dynamic registration of semantic tokens")

		// Register semantic tokens provider dynamically
		err := s.callbackClient.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{
				{
					ID:     "semantic-tokens",
					Method: "textDocument/semanticTokens",
					RegisterOptions: &protocol.SemanticTokensRegistrationOptions{
						TextDocumentRegistrationOptions: protocol.TextDocumentRegistrationOptions{
							DocumentSelector: []protocol.DocumentFilter{
								{
									Value: protocol.Or_TextDocumentFilter{
										Value: protocol.TextDocumentFilterLanguage{
											Language: "gotmpl",
										},
									},
								},
							},
						},
						SemanticTokensOptions: protocol.SemanticTokensOptions{
							Legend: protocol.SemanticTokensLegend{
								TokenTypes: []string{
									"variable",      // 0
									"function",      // 1
									"keyword",       // 2
									"string",        // 3
									"number",        // 4
									"comment",       // 5
									"operator",      // 6
									"macro",         // 7
									"namespace",     // 8
									"parameter",     // 9
									"type",          // 10
									"typeParameter", // 11
									"method",        // 12
									"label",         // 13
								},
								TokenModifiers: []string{
									"declaration",    // 1 << 0
									"definition",     // 1 << 1
									"readonly",       // 1 << 2
									"static",         // 1 << 3
									"deprecated",     // 1 << 4
									"abstract",       // 1 << 5
									"async",          // 1 << 6
									"modification",   // 1 << 7
									"documentation",  // 1 << 8
									"defaultLibrary", // 1 << 9
								},
							},
							Full:  &protocol.Or_SemanticTokensOptions_full{Value: true},
							Range: &protocol.Or_SemanticTokensOptions_range{Value: true},
						},
					},
				},
			},
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to register semantic tokens provider")
			return errors.Errorf("registering semantic tokens provider: %w", err)
		}
		logger.Debug().Msg("successfully registered semantic tokens provider")
	} else {
		logger.Debug().Msg("client does not support dynamic registration of semantic tokens, using static registration")
	}

	return nil
}

func (s *Server) Resolve(ctx context.Context, params *protocol.InlayHint) (*protocol.InlayHint, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) DidChangeNotebookDocument(ctx context.Context, params *protocol.DidChangeNotebookDocumentParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidCloseNotebookDocument(ctx context.Context, params *protocol.DidCloseNotebookDocumentParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidOpenNotebookDocument(ctx context.Context, params *protocol.DidOpenNotebookDocumentParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidSaveNotebookDocument(ctx context.Context, params *protocol.DidSaveNotebookDocumentParams) error {
	return nil // Not implemented yet
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil // Not implemented yet
}

func (s *Server) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Declaration(ctx context.Context, params *protocol.DeclarationParams) (*protocol.Or_textDocument_declaration, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Diagnostic(ctx context.Context, params *protocol.DocumentDiagnosticParams) (*protocol.DocumentDiagnosticReport, error) {

	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	diagnostics, err := s.identifyDiagnosticsForFile(ctx, params.TextDocument.URI, doc.Content)
	if err != nil {
		return nil, errors.Errorf("identifying diagnostics: %w", err)
	}

	return &protocol.DocumentDiagnosticReport{
		Value: protocol.RelatedFullDocumentDiagnosticReport{

			FullDocumentDiagnosticReport: protocol.FullDocumentDiagnosticReport{
				Items: diagnostics,
			},
		},
	}, nil
}

func (s *Server) DiagnosticWorkspace(ctx context.Context, params *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("document changed")

	// For now, we'll just use the full content sync
	if len(params.ContentChanges) > 0 {
		doc, ok := s.documents.Get(params.TextDocument.URI)
		if !ok {
			return errors.Errorf("document not found: %s", params.TextDocument.URI)
		}

		// Update document
		doc.Version = params.TextDocument.Version
		for _, change := range params.ContentChanges {
			if change.Range == nil {
				doc.Content = change.Text
			} else {
				doc.Content = replaceContentFromRange(ctx, doc.Content, change.Range, change.Text)
			}
		}

		s.documents.Store(params.TextDocument.URI, doc)

		return s.publishDiagnostics(ctx, params.TextDocument.URI, doc.Content)
	}

	return nil
}

func replaceContentFromRange(ctx context.Context, content string, rangez *protocol.Range, text string) string {
	startPos := position.NewRawPositionFromLineAndColumn(int(rangez.Start.Line), int(rangez.Start.Character), "", content)
	endPos := position.NewRawPositionFromLineAndColumn(int(rangez.End.Line), int(rangez.End.Character), "", content)
	zerolog.Ctx(ctx).Debug().Msgf(`replacing content from %s to %s with %s`, startPos.ID(), endPos.ID(), text)
	return content[:startPos.Offset] + text + content[endPos.Offset:]
}

func (s *Server) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("document closed")

	s.documents.Delete(normalizeURI(string(params.TextDocument.URI)))
	return nil
}

func (s *Server) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("document opened")

	doc := &Document{
		URI:        string(params.TextDocument.URI),
		LanguageID: params.TextDocument.LanguageID,
		Version:    params.TextDocument.Version,
		Content:    params.TextDocument.Text,
	}

	s.documents.Store(params.TextDocument.URI, doc)

	// Request semantic token refresh
	if s.callbackClient != nil {
		logger.Debug().Msg("requesting semantic token refresh")
		err := s.callbackClient.SemanticTokensRefresh(ctx)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to refresh semantic tokens")
		}
		logger.Debug().Msg("semantic token refresh requested")
	} else {
		logger.Warn().Msg("no callback client available for semantic token refresh")
	}

	return s.publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) error {
	return nil // Not implemented yet
}

func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("document saved")

	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	if params.Text != nil {
		doc.Content = *params.Text
		s.documents.Store(params.TextDocument.URI, doc)
	}

	zerolog.Ctx(ctx).Trace().Str("uri", string(params.TextDocument.URI)).Str("content", doc.Content).Msg("document saved")

	return s.publishDiagnostics(ctx, params.TextDocument.URI, doc.Content)

}

func (s *Server) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]any, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (any, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) FoldingRange(ctx context.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	zerolog.Ctx(ctx).Trace().Msgf("hover request received: %+v", params)

	uripath := params.TextDocument.URI.Path()

	// Create overlay map with current document content
	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}
	overlay := map[string][]byte{
		uripath: []byte(doc.Content),
	}

	reg, err := ast.AnalyzePackage(ctx, uripath, overlay)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	content, _, ok := reg.GetTemplateFile(uripath)
	if !ok {
		return nil, errors.Errorf("template %s not found, make sure its embeded", uripath)
	}

	// Parse the template
	info, err := parser.Parse(ctx, uripath, []byte(content))
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.NewRawPositionFromLineAndColumn(int(params.Position.Line), int(params.Position.Character), string(content[params.Position.Character]), content)

	hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, reg)
	if err != nil {
		return nil, errors.Errorf("building hover response: %w", err)
	}

	if hoverInfo == nil {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  "markdown",
			Value: strings.Join(hoverInfo.Content, "\n"),
		},
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(hoverInfo.Position.GetRange(content).Start.Line),
				Character: uint32(hoverInfo.Position.GetRange(content).Start.Character),
			},
			End: protocol.Position{
				Line:      uint32(hoverInfo.Position.GetRange(content).End.Line),
				Character: uint32(hoverInfo.Position.GetRange(content).End.Character),
			},
		},
	}, nil
}

func (s *Server) Implementation(ctx context.Context, params *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) InlayHint(ctx context.Context, params *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) InlineCompletion(ctx context.Context, params *protocol.InlineCompletionParams) (*protocol.Or_Result_textDocument_inlineCompletion, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) InlineValue(ctx context.Context, params *protocol.InlineValueParams) ([]protocol.InlineValue, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Moniker(ctx context.Context, params *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.PrepareRenameResult, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) PrepareTypeHierarchy(ctx context.Context, params *protocol.TypeHierarchyPrepareParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) RangesFormatting(ctx context.Context, params *protocol.DocumentRangesFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) SelectionRange(ctx context.Context, params *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("uri", string(params.TextDocument.URI)).
		Str("method", "textDocument/semanticTokens/full").
		Msg("semantic tokens request received")

	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		logger.Error().Str("uri", string(params.TextDocument.URI)).Msg("document not found")
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Generate semantic tokens
	tokens, err := semtok.GetTokensForText(ctx, []byte(doc.Content))
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate semantic tokens")
		return nil, errors.Errorf("generating semantic tokens: %w", err)
	}

	logger.Debug().Int("token_count", len(tokens)).Msg("generated semantic tokens")
	for i, tok := range tokens {
		rng := tok.Range.GetRange(doc.Content)
		logger.Debug().
			Int("index", i).
			Str("type", string(tok.Type)).
			Str("modifier", string(tok.Modifier)).
			Int("line", rng.Start.Line).
			Int("char", rng.Start.Character).
			Int("end_char", rng.End.Character).
			Msg("token details")
	}

	// Convert to LSP format
	result := s.convertToLSPTokens(tokens, doc.Content)
	logger.Debug().Int("data_length", len(result.Data)).Msg("converted to LSP format")

	return result, nil
}

func (s *Server) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (any, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("uri", string(params.TextDocument.URI)).
		Str("method", "textDocument/semanticTokens/full/delta").
		Str("previous_result_id", params.PreviousResultID).
		Msg("semantic tokens delta request received")

	// We don't support delta updates yet, fallback to full
	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Generate semantic tokens
	tokens, err := semtok.GetTokensForText(ctx, []byte(doc.Content))
	if err != nil {
		return nil, errors.Errorf("generating semantic tokens: %w", err)
	}

	// Convert to LSP format
	return s.convertToLSPTokens(tokens, doc.Content), nil
}

func (s *Server) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("uri", string(params.TextDocument.URI)).
		Str("method", "textDocument/semanticTokens/range").
		Interface("range", params.Range).
		Msg("semantic tokens range request received")

	doc, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// For now, we'll just return tokens for the full document
	// TODO: Implement range-based token generation
	tokens, err := semtok.GetTokensForText(ctx, []byte(doc.Content))
	if err != nil {
		return nil, errors.Errorf("generating semantic tokens: %w", err)
	}

	// Convert to LSP format
	return s.convertToLSPTokens(tokens, doc.Content), nil
}

func (s *Server) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Subtypes(ctx context.Context, params *protocol.TypeHierarchySubtypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Supertypes(ctx context.Context, params *protocol.TypeHierarchySupertypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) Symbol(ctx context.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) TextDocumentContent(ctx context.Context, params *protocol.TextDocumentContentParams) (*string, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) error {
	return nil // Not implemented yet
}

func (s *Server) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) error {
	return nil // Not implemented yet
}

func (s *Server) ResolveWorkspaceSymbol(ctx context.Context, params *protocol.WorkspaceSymbol) (*protocol.WorkspaceSymbol, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) identifyDiagnosticsForFile(ctx context.Context, urid protocol.DocumentURI, content string) ([]protocol.Diagnostic, error) {
	logger := zerolog.Ctx(ctx)
	uri := normalizeURI(string(urid))
	logger.Debug().Str("uri", uri).Msg("validating document")

	// Create overlay map with current document content
	overlay := map[string][]byte{
		urid.Path(): []byte(content),
	}

	registry, err := ast.AnalyzePackage(ctx, uri, overlay)
	if err != nil {
		return nil, errors.Errorf("analyzing package: %w", err)
	}

	nodes, err := parser.Parse(ctx, uri, []byte(content))
	if err != nil {
		return nil, errors.Errorf("parsing template for validation: %w", err)
	}

	diagnostics, err := diagnostic.GetDiagnosticsFromParsed(ctx, nodes, registry)
	if err != nil {
		return nil, errors.Errorf("getting diagnostics: %w", err)
	}

	var result []protocol.Diagnostic = make([]protocol.Diagnostic, len(diagnostics))

	for i, d := range diagnostics {
		result[i] = protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(d.Location.GetRange(content).Start.Line),
					Character: uint32(d.Location.GetRange(content).Start.Character),
				},
				End: protocol.Position{
					Line:      uint32(d.Location.GetRange(content).End.Line),
					Character: uint32(d.Location.GetRange(content).End.Character),
				},
			},
			Severity: protocol.DiagnosticSeverity(d.Severity),
			Message:  d.Message,
		}
	}

	return result, nil
}

func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, content string) error {

	diagnostics, err := s.identifyDiagnosticsForFile(ctx, uri, content)
	if err != nil {
		return errors.Errorf("identifying diagnostics: %w", err)
	}

	params := &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	zerolog.Ctx(ctx).Debug().Msgf("found diagnostics: %+v", params)

	if s.callbackClient != nil {
		return s.callbackClient.PublishDiagnostics(ctx, params)
	} else {
		zerolog.Ctx(ctx).Warn().Msg("no callback client, skipping publish diagnostics")
	}

	return nil
}

// Token type indices in the legend array
const (
	tokenTypeNamespace = iota
	tokenTypeType
	tokenTypeClass
	tokenTypeEnum
	tokenTypeInterface
	tokenTypeStruct
	tokenTypeTypeParameter
	tokenTypeParameter
	tokenTypeVariable
	tokenTypeProperty
	tokenTypeEnumMember
	tokenTypeEvent
	tokenTypeFunction
	tokenTypeMethod
	tokenTypeMacro
	tokenTypeKeyword
	tokenTypeModifier
	tokenTypeComment
	tokenTypeString
	tokenTypeNumber
	tokenTypeRegexp
	tokenTypeOperator
	tokenTypeDecorator
	tokenTypeLabel
)

// Token modifier bit flags
const (
	tokenModDeclaration = 1 << iota
	tokenModDefinition
	tokenModReadonly
	tokenModStatic
	tokenModAbstract
	tokenModAsync
	tokenModDefaultLibrary
	tokenModDeprecated
	tokenModDocumentation
	tokenModModification
)

// convertToLSPTokens converts our semantic tokens to LSP format
func (s *Server) convertToLSPTokens(tokens []semtok.Token, content string) *protocol.SemanticTokens {
	// LSP requires tokens to be sorted by line and character
	sort.Slice(tokens, func(i, j int) bool {
		iRange := tokens[i].Range.GetRange(content)
		jRange := tokens[j].Range.GetRange(content)
		if iRange.Start.Line != jRange.Start.Line {
			return iRange.Start.Line < jRange.Start.Line
		}
		return iRange.Start.Character < jRange.Start.Character
	})

	// Convert to LSP's relative encoding
	data := make([]uint32, 0, len(tokens)*5)
	var prevLine, prevChar uint32

	// Map our token types to LSP token type indices
	tokenTypeMap := map[semtok.TokenType]uint32{
		semtok.TokenVariable:  0,
		semtok.TokenFunction:  1,
		semtok.TokenKeyword:   2,
		semtok.TokenString:    3,
		semtok.TokenNumber:    4,
		semtok.TokenComment:   5,
		semtok.TokenOperator:  6,
		semtok.TokenMacro:     7,
		semtok.TokenNamespace: 8,
		semtok.TokenParameter: 9,
		semtok.TokenTypeKind:  10,
		semtok.TokenTypeParam: 11,
		semtok.TokenMethod:    12,
		semtok.TokenLabel:     13,
	}

	// Map our token modifiers to LSP token modifier bit positions
	tokenModifierMap := map[semtok.TokenModifier]uint32{
		semtok.ModifierDeclaration:    1 << 0,
		semtok.ModifierDefinition:     1 << 1,
		semtok.ModifierReadonly:       1 << 2,
		semtok.ModifierStatic:         1 << 3,
		semtok.ModifierDeprecated:     1 << 4,
		semtok.ModifierAbstract:       1 << 5,
		semtok.ModifierAsync:          1 << 6,
		semtok.ModifierModification:   1 << 7,
		semtok.ModifierDocumentation:  1 << 8,
		semtok.ModifierDefaultLibrary: 1 << 9,
	}

	for _, tok := range tokens {
		// Get token range
		rng := tok.Range.GetRange(content)
		line := uint32(rng.Start.Line)
		char := uint32(rng.Start.Character)
		length := uint32(rng.End.Character - rng.Start.Character)

		// Calculate relative positions
		deltaLine := line - prevLine
		deltaChar := char
		if deltaLine == 0 {
			deltaChar = char - prevChar
		}

		// Get token type index
		tokenType := tokenTypeMap[tok.Type]

		// Get token modifiers
		tokenModifiers := tokenModifierMap[tok.Modifier]

		// Add token data in LSP format:
		// [deltaLine, deltaChar, length, tokenType, tokenModifiers]
		data = append(data, deltaLine, deltaChar, length, tokenType, tokenModifiers)

		// Update previous positions
		prevLine = line
		prevChar = char
	}

	return &protocol.SemanticTokens{
		Data: data,
	}
}
