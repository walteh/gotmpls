package lsp

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/ast"
	"github.com/walteh/gotmpls/pkg/diagnostic"
	"github.com/walteh/gotmpls/pkg/hover"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
	"gitlab.com/tozd/go/errors"
	"gopkg.in/fsnotify.v1"
)

// Document represents a text document with its metadata
type Document struct {
	URI        string
	LanguageID protocol.LanguageKind
	Version    int32
	Content    string
	AST        *parser.ParsedTemplateFile
}

// DocumentManager handles document operations
type DocumentManager struct {
	store *sync.Map // map[string]*Document
}

func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		store: &sync.Map{},
	}
}

func (m *DocumentManager) Get(uri protocol.DocumentURI) (*Document, bool) {
	normalizedURI := normalizeURI(string(uri))
	content, ok := m.store.Load(normalizedURI)
	if !ok {
		// Try with the original URI as fallback
		content, ok = m.store.Load("file://" + uri)
	}
	if !ok {
		// try filesystem
		file, err := os.Open(normalizedURI)
		if err != nil {
			return nil, false
		}
		defer file.Close()
		contentz, err := io.ReadAll(file)
		if err != nil {
			return nil, false
		}
		doc := &Document{
			URI:     normalizedURI,
			Content: string(contentz),
		}
		m.store.Store(normalizedURI, doc)
		return doc, true
	}

	doc, ok := content.(*Document)
	return doc, ok
}

func (m *DocumentManager) Store(uri protocol.DocumentURI, doc *Document) {
	normalizedURI := normalizeURI(string(uri))
	m.store.Store(normalizedURI, doc)
}

func (m *DocumentManager) Delete(uri string) {
	normalizedURI := normalizeURI(uri)
	m.store.Delete(normalizedURI)
}

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
	instance *protocol.ServerInstance
}

func NewServer(ctx context.Context) *Server {
	return &Server{
		id:          xid.New().String(),
		documents:   NewDocumentManager(),
		cancelFuncs: &sync.Map{},
		debug:       false, // Disabled debug mode
	}
}

// func (s *Server) Run(ctx context.Context, reader io.Reader, writer io.WriteCloser, opts *jrpc2.ServerOptions) error {
// 	server := s.Detach(ctx, reader, writer, opts)
// 	return server.Wait()
// }

func (s *Server) BuildServerInstance(ctx context.Context, opts *jrpc2.ServerOptions) *protocol.ServerInstance {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("starting LSP server")

	if s.instance != nil {
		s.instance.ServerOpts = opts
		return s.instance
	}

	s.instance = protocol.NewServerInstance(ctx, s, opts)

	return s.instance
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
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
				Value: true,
			},
			TextDocumentSync: &protocol.Or_ServerCapabilities_textDocumentSync{
				Value: protocol.Incremental,
			},

			SemanticTokensProvider: &protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes: []string{
						string(protocol.NamespaceType),
						string(protocol.TypeType),
						string(protocol.ClassType),
						string(protocol.EnumType),
						string(protocol.InterfaceType),
						string(protocol.StructType),
						string(protocol.TypeParameterType),
						string(protocol.ParameterType),
						string(protocol.VariableType),
						string(protocol.PropertyType),
						string(protocol.EnumMemberType),
						string(protocol.EventType),
						string(protocol.FunctionType),
						string(protocol.MethodType),
						string(protocol.MacroType),
						string(protocol.KeywordType),
						string(protocol.ModifierType),
						string(protocol.CommentType),
						string(protocol.StringType),
						string(protocol.NumberType),
						string(protocol.RegexpType),
						string(protocol.OperatorType),
						string(protocol.DecoratorType),
						string(protocol.LabelType),
					},
					TokenModifiers: []string{
						string(protocol.ModDeclaration),
						string(protocol.ModDefinition),
						string(protocol.ModReadonly),
						string(protocol.ModStatic),
						string(protocol.ModAbstract),
						string(protocol.ModAsync),
						string(protocol.ModDefaultLibrary),
						string(protocol.ModDeprecated),
						string(protocol.ModDocumentation),
						string(protocol.ModModification),
					},
				},
				Full:  &protocol.Or_SemanticTokensOptions_full{Value: true},
				Range: &protocol.Or_SemanticTokensOptions_range{Value: true},
			},
		},
	}, nil
}

func (s *Server) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return nil // Not implemented yet
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
				if change.Text != "" {
					// zerolog.Ctx(ctx).Trace().Str("uri", string(params.TextDocument.URI)).Str("content", doc.Content).Any("change", change).Msg("document changed")
					doc.Content = replaceContentFromRange(doc.Content, change.Range, change.Text)
				}
			}
		}

		s.documents.Store(params.TextDocument.URI, doc)

		return s.publishDiagnostics(ctx, params.TextDocument.URI, doc.Content)
	}

	return nil
}

func replaceContentFromRange(content string, rangez *protocol.Range, text string) string {
	startPos := position.NewRawPositionFromLineAndColumn(int(rangez.Start.Line), int(rangez.Start.Character), "", content)
	endPos := position.NewRawPositionFromLineAndColumn(int(rangez.End.Line), int(rangez.End.Character), "", content)
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

	s.documents.Delete(string(params.TextDocument.URI))
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
	zerolog.Ctx(ctx).Debug().Msgf("hover request received: %+v", params)

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
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("getting semantic tokens")

	_, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	return nil, nil
}

func (s *Server) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (any, error) {
	return nil, nil // Not implemented yet
}

func (s *Server) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("uri", string(params.TextDocument.URI)).Msg("getting semantic tokens for range")

	_, ok := s.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	return nil, nil
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

	return s.instance.CallbackClient().PublishDiagnostics(ctx, params)
}
