package lsp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/hover"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// type handlerFunc func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error)

// // bufferedReadWriteCloser wraps a reader and writer with buffering
// type bufferedReadWriteCloser struct {
// 	reader *bufio.Reader
// 	writer *bufio.Writer
// 	closer io.Closer
// }

// func newBufferedReadWriteCloser(r io.Reader, w io.Writer) io.ReadWriteCloser {
// 	return &bufferedReadWriteCloser{
// 		reader: bufio.NewReader(r),
// 		writer: bufio.NewWriter(w),
// 		closer: io.NopCloser(nil),
// 	}
// }

// func (b *bufferedReadWriteCloser) Read(p []byte) (n int, err error) {
// 	return b.reader.Read(p)
// }

// func (b *bufferedReadWriteCloser) Write(p []byte) (n int, err error) {
// 	n, err = b.writer.Write(p)
// 	if err != nil {
// 		return n, err
// 	}
// 	return n, b.writer.Flush()
// }

// func (b *bufferedReadWriteCloser) Close() error {
// 	return b.closer.Close()
// }

// Server represents an LSP server instance
type Server struct {
	documents sync.Map // map[string]string
	workspace string
	conn      *jsonrpc2.Conn
	id        string
	// extraLogWriters []io.Writer
	debug bool
}

var _ protocol.Server = (*Server)(nil)

func NewServer(ctx context.Context) *Server {
	return &Server{
		// extraLogWriters: extraLogWriters,
		id:        xid.New().String(),
		conn:      nil,
		documents: sync.Map{},
		workspace: "",
	}
}

// func Spawn(ctx context.Context, reader io.Reader, writer io.Writer, extraLogWriters ...io.Writer) error {
// 	server := NewServer(ctx, extraLogWriters...)
// 	return server.Run(ctx, reader, writer, extraLogWriters...)
// }

func (s *Server) Run(ctx context.Context, reader io.Reader, writer io.WriteCloser, opts *jrpc2.ServerOptions) error {
	zerolog.Ctx(ctx).Info().Msg("starting LSP server - all logging will be redirected to LSP")

	server := protocol.NewServerServer(ctx, s, opts)

	in := channel.LSP(reader, writer)

	server = server.Start(in)

	return server.Wait()

	// // Wait for either the connection to be closed or the context to be done
	// select {
	// case <-conn.DisconnectNotify():
	// 	return nil
	// case <-ctx.Done():
	// 	return ctx.Err()
	// }
}

// func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
// 	ctx = s.ApplyLSPWriter(ctx, conn, s.extraLogWriters...)
// 	if s.debug {
// 		s.debugf(ctx, "received request: %s", req.Method)
// 		if req.Params != nil {
// 			s.debugf(ctx, "request params: %s", string(*req.Params))
// 		}
// 	}

// 	ctx = zerolog.Ctx(ctx).With().Str("method", req.Method).Logger().WithContext(ctx)

// 	handler := s.router(ctx, req.Method)
// 	if handler == nil {
// 		if s.debug {
// 			s.debugf(ctx, "unhandled method: %s", req.Method)
// 		}
// 		if !req.Notif {
// 			if err := conn.Reply(ctx, req.ID, nil); err != nil {
// 				s.debugf(ctx, "error sending default reply: %v", err)
// 			}
// 		}
// 		return
// 	}

// 	result, err := handler(ctx, req)
// 	if err != nil {
// 		s.debugf(ctx, "error handling %s: %v", req.Method, err)
// 		if !req.Notif {
// 			if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
// 				Code:    jsonrpc2.CodeInternalError,
// 				Message: err.Error(),
// 			}); err != nil {
// 				s.debugf(ctx, "error sending error reply: %v", err)
// 			}
// 		}
// 		return
// 	}

// 	if !req.Notif && result != nil {
// 		if err := conn.Reply(ctx, req.ID, result); err != nil {
// 			s.debugf(ctx, "error sending reply: %v", err)
// 		}
// 	}
// }

// func (s *Server) router(ctx context.Context, method string) handlerFunc {
// 	zerolog.Ctx(ctx).Info().Str("method", method).Msg("routing request")
// 	switch method {
// 	case "initialize", "initialized":
// 		return s.handleInitialize
// 	// case "initialized":
// 	// 	return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 	// 		s.debugf(ctx, "initialized")

// 	// 		zerolog.Ctx(ctx).Info().Str("params", string(*req.Params)).Msg("initialized")
// 	// 		return nil, nil
// 	// 	}
// 	case "shutdown":
// 		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return nil, nil
// 		}
// 	case "exit":
// 		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return nil, nil
// 		}
// 	case "textDocument/didOpen":
// 		return s.handleTextDocumentDidOpen
// 	case "textDocument/didChange":
// 		return s.handleTextDocumentDidChange
// 	case "textDocument/didClose":
// 		return s.handleTextDocumentDidClose
// 	case "textDocument/hover":
// 		return s.handleTextDocumentHover
// 	// case "textDocument/completion":
// 	// 	return s.handleTextDocumentCompletion
// 	case "$/setTrace":
// 		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return nil, nil
// 		}
// 	case "$/cancelRequest":
// 		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return nil, nil
// 		}
// 	case "workspace/didChangeConfiguration":
// 		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return nil, nil
// 		}
// 	default:
// 		return nil
// 	}
// }

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

// func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 	if s.debug {
// 		s.debugf(ctx, "handling textDocument/hover")
// 	}

// 	var params HoverParams
// 	if err := json.Unmarshal(*req.Params, &params); err != nil {
// 		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
// 	}

// 	zerolog.Ctx(ctx).Debug().Msgf("hover request received: %+v", params)

// 	uri := s.normalizeURI(params.TextDocument.URI)

// 	// // Get document content
// 	// content, ok := s.getDocument(uri)
// 	// if !ok {
// 	// 	return nil, errors.Errorf("document not found: %s", uri)
// 	// }

// 	reg, err := ast.AnalyzePackage(ctx, uri)
// 	if err != nil {
// 		return nil, errors.Errorf("analyzing package for hover: %w", err)
// 	}

// 	content, _, ok := reg.GetTemplateFile(uri)
// 	if !ok {
// 		return nil, errors.Errorf("template %s not found, make sure its embeded", uri)
// 	}

// 	// // Parse the template
// 	info, err := parser.Parse(ctx, uri, []byte(content))
// 	if err != nil {
// 		return nil, errors.Errorf("parsing template for hover: %w", err)
// 	}

// 	pos := position.NewRawPositionFromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

// 	hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, reg)
// 	if err != nil {
// 		return nil, errors.Errorf("building hover response: %w", err)
// 	}

// 	if hoverInfo == nil {
// 		return nil, nil
// 	}

// 	hovers := make([]Hover, len(hoverInfo.Content))
// 	for i, hcontent := range hoverInfo.Content {
// 		hovers[i] = Hover{
// 			Contents: MarkupContent{
// 				Kind:  "markdown",
// 				Value: hcontent,
// 			},
// 			Range: rangeFromLSP(hoverInfo.Position.GetRange(content)),
// 		}
// 	}

// 	// TODO: Return more than one
// 	if len(hovers) > 0 {
// 		return &hovers[0], nil
// 	}

// 	return nil, nil
// }

// rangeToLSP converts a position.Range to an LSP Range
func rangeFromLSP(r position.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(r.Start.Line),
			Character: uint32(r.Start.Character),
		},
		End: protocol.Position{
			Line:      uint32(r.End.Line),
			Character: uint32(r.End.Character),
		},
	}
}

func rangeToLSP(r protocol.Range) position.Range {
	return position.Range{
		Start: position.Place{
			Line:      int(r.Start.Line),
			Character: int(r.Start.Character),
		},
		End: position.Place{
			Line:      int(r.End.Line),
			Character: int(r.End.Character),
		},
	}
}

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "validating document: %s", uri)
	}

	uri = s.normalizeURI(uri)

	registry, err := ast.AnalyzePackage(ctx, uri)
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

	// Parse the template to get type hints
	// info, err := parser.Parse(ctx, []byte(content), uri)
	// if err != nil {
	// 	return nil, errors.Errorf("parsing template for validation: %w", err)
	// }

	// Add success diagnostic for type hint if present and valid
	// for _, block := range info.Blocks {
	// 	if block.TypeHint != nil {
	// 		if typeInfo, err := ast.GenerateTypeInfoFromRegistry(ctx, block.TypeHint.TypePath, registry); err == nil {
	// 			// Add success diagnostic
	// 			diagnostics = append(diagnostics, &diagnostic.Diagnostic{
	// 				Message:  "Type hint successfully loaded: " + typeInfo.Name,
	// 				Location: block.TypeHint.Position,
	// 				Severity: diagnostic.SeverityInformation,
	// 			})
	// 		}
	// 	}
	// }

	// Publish diagnostics
	if err := s.publishDiagnostics(ctx, uri, content, diagnostics); err != nil {
		return nil, errors.Errorf("publishing diagnostics: %w", err)
	}

	return nil, nil
}

func (s *Server) publishDiagnostics(ctx context.Context, uri string, content string, diagnostics []*diagnostic.Diagnostic) error {
	params := &PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: make([]Diagnostic, len(diagnostics)),
	}
	for i, d := range diagnostics {
		params.Diagnostics[i] = DiagnosticFromGoTmplTyperDiagnostic(d, content)
	}

	// if s.debug {
	// 	s.debugf(ctx, "publishing %d diagnostics for %s", len(diagnostics), uri)
	// 	for _, d := range diagnostics {
	// 		severity := "unknown"
	// 		switch d.Severity {
	// 		case 1:
	// 			severity = "error"
	// 		case 2:
	// 			severity = "warning"
	// 		case 3:
	// 			severity = "information"
	// 		case 4:
	// 			severity = "hint"
	// 		}
	// 		s.debugf(ctx, "  - %s at %v: %s", severity, d.Range, d.Message)
	// 	}
	// }

	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

// normalizeURI ensures consistent URI handling by removing the file:// prefix if present
// and converting to a clean path
func (s *Server) normalizeURI(uri string) string {

	uri = strings.TrimPrefix(uri, "file://")
	// remove the file:/private prefix
	uri = strings.TrimPrefix(uri, "file:")

	// // Remove any leading slashes for consistency
	// uri = strings.TrimLeft(uri, "/")
	return uri
}

// getDocument retrieves a document from the server's store using a normalized URI
func (s *Server) getDocument(uri string) (string, bool) {
	normalizedURI := s.normalizeURI(uri)
	content, ok := s.documents.Load(normalizedURI)
	if !ok {
		// Try with the original URI as fallback
		content, ok = s.documents.Load("file://" + uri)
	}
	if !ok {

		// try filesystem

		file, err := os.Open(normalizedURI)
		if err != nil {
			return "", false
		}
		defer file.Close()
		contentz, err := io.ReadAll(file)
		if err != nil {
			return "", false
		}
		content = string(contentz)
		s.documents.Store(normalizedURI, content)
	}

	text, ok := content.(string)
	return text, ok
}

// storeDocument stores a document in the server's store using a normalized URI
func (s *Server) storeDocument(uri string, content string) {
	normalizedURI := s.normalizeURI(uri)
	s.documents.Store(normalizedURI, content)
}

// CodeAction implements protocol.Server.
func (s *Server) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	panic("unimplemented")
}

// CodeLens implements protocol.Server.
func (s *Server) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	panic("unimplemented")
}

// ColorPresentation implements protocol.Server.
func (s *Server) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	panic("unimplemented")
}

// Completion implements protocol.Server.
func (s *Server) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	panic("unimplemented")
}

// Declaration implements protocol.Server.
func (s *Server) Declaration(context.Context, *protocol.DeclarationParams) (*protocol.Or_textDocument_declaration, error) {
	panic("unimplemented")
}

// Definition implements protocol.Server.
func (s *Server) Definition(context.Context, *protocol.DefinitionParams) ([]protocol.Location, error) {
	panic("unimplemented")
}

// Diagnostic implements protocol.Server.
func (s *Server) Diagnostic(context.Context, *protocol.DocumentDiagnosticParams) (*protocol.DocumentDiagnosticReport, error) {
	panic("unimplemented")
}

// DiagnosticWorkspace implements protocol.Server.
func (s *Server) DiagnosticWorkspace(context.Context, *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	panic("unimplemented")
}

// DidChange implements protocol.Server.
func (s *Server) DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error {
	panic("unimplemented")
}

// DidChangeConfiguration implements protocol.Server.
func (s *Server) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	panic("unimplemented")
}

// DidChangeNotebookDocument implements protocol.Server.
func (s *Server) DidChangeNotebookDocument(context.Context, *protocol.DidChangeNotebookDocumentParams) error {
	panic("unimplemented")
}

// DidChangeWatchedFiles implements protocol.Server.
func (s *Server) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	panic("unimplemented")
}

// DidChangeWorkspaceFolders implements protocol.Server.
func (s *Server) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	panic("unimplemented")
}

// DidClose implements protocol.Server.
func (s *Server) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	panic("unimplemented")
}

// DidCloseNotebookDocument implements protocol.Server.
func (s *Server) DidCloseNotebookDocument(context.Context, *protocol.DidCloseNotebookDocumentParams) error {
	panic("unimplemented")
}

// DidCreateFiles implements protocol.Server.
func (s *Server) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	panic("unimplemented")
}

// DidDeleteFiles implements protocol.Server.
func (s *Server) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	panic("unimplemented")
}

// DidOpen implements protocol.Server.
func (s *Server) DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error {
	return nil
}

// DidOpenNotebookDocument implements protocol.Server.
func (s *Server) DidOpenNotebookDocument(context.Context, *protocol.DidOpenNotebookDocumentParams) error {
	panic("unimplemented")
}

// DidRenameFiles implements protocol.Server.
func (s *Server) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	panic("unimplemented")
}

// DidSave implements protocol.Server.
func (s *Server) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

// DidSaveNotebookDocument implements protocol.Server.
func (s *Server) DidSaveNotebookDocument(context.Context, *protocol.DidSaveNotebookDocumentParams) error {
	panic("unimplemented")
}

// DocumentColor implements protocol.Server.
func (s *Server) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	panic("unimplemented")
}

// DocumentHighlight implements protocol.Server.
func (s *Server) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	panic("unimplemented")
}

// DocumentLink implements protocol.Server.
func (s *Server) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	panic("unimplemented")
}

// DocumentSymbol implements protocol.Server.
func (s *Server) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]any, error) {
	panic("unimplemented")
}

// ExecuteCommand implements protocol.Server.
func (s *Server) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (any, error) {
	panic("unimplemented")
}

// Exit implements protocol.Server.
func (s *Server) Exit(context.Context) error {
	return nil
}

// FoldingRange implements protocol.Server.
func (s *Server) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	panic("unimplemented")
}

// Formatting implements protocol.Server.
func (s *Server) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	panic("unimplemented")
}

// Hover implements protocol.Server.
func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {

	zerolog.Ctx(ctx).Debug().Msgf("hover request received: %+v", params)

	// // Get document content
	// content, ok := s.getDocument(uri)
	// if !ok {
	// 	return nil, errors.Errorf("document not found: %s", uri)
	// }

	uripath := params.TextDocument.URI.Path()

	reg, err := ast.AnalyzePackage(ctx, uripath)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	content, _, ok := reg.GetTemplateFile(uripath)
	if !ok {
		return nil, errors.Errorf("template %s not found, make sure its embeded", uripath)
	}

	// // Parse the template
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

	hovers := make([]protocol.Hover, len(hoverInfo.Content))
	for i, hcontent := range hoverInfo.Content {
		hovers[i] = protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  "markdown",
				Value: hcontent,
			},
			Range: rangeFromLSP(hoverInfo.Position.GetRange(content)),
		}
	}

	// TODO: Return more than one
	if len(hovers) > 0 {
		return &hovers[0], nil
	}

	return nil, nil
}

// Implementation implements protocol.Server.
func (s *Server) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	panic("unimplemented")
}

// IncomingCalls implements protocol.Server.
func (s *Server) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	panic("unimplemented")
}

// Initialize implements protocol.Server.
func (s *Server) Initialize(context.Context, *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			HoverProvider: &protocol.Or_ServerCapabilities_hoverProvider{
				Value: true,
			},
		},
	}, nil
}

// Initialized implements protocol.Server.
func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
}

// InlayHint implements protocol.Server.
func (s *Server) InlayHint(context.Context, *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	panic("unimplemented")
}

// InlineCompletion implements protocol.Server.
func (s *Server) InlineCompletion(context.Context, *protocol.InlineCompletionParams) (*protocol.Or_Result_textDocument_inlineCompletion, error) {
	panic("unimplemented")
}

// InlineValue implements protocol.Server.
func (s *Server) InlineValue(context.Context, *protocol.InlineValueParams) ([]protocol.InlineValue, error) {
	panic("unimplemented")
}

// LinkedEditingRange implements protocol.Server.
func (s *Server) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	panic("unimplemented")
}

// Moniker implements protocol.Server.
func (s *Server) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	panic("unimplemented")
}

// OnTypeFormatting implements protocol.Server.
func (s *Server) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	panic("unimplemented")
}

// OutgoingCalls implements protocol.Server.
func (s *Server) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	panic("unimplemented")
}

// PrepareCallHierarchy implements protocol.Server.
func (s *Server) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	panic("unimplemented")
}

// PrepareRename implements protocol.Server.
func (s *Server) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.PrepareRenameResult, error) {
	panic("unimplemented")
}

// PrepareTypeHierarchy implements protocol.Server.
func (s *Server) PrepareTypeHierarchy(context.Context, *protocol.TypeHierarchyPrepareParams) ([]protocol.TypeHierarchyItem, error) {
	panic("unimplemented")
}

// Progress implements protocol.Server.
func (s *Server) Progress(context.Context, *protocol.ProgressParams) error {
	panic("unimplemented")
}

// RangeFormatting implements protocol.Server.
func (s *Server) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	panic("unimplemented")
}

// RangesFormatting implements protocol.Server.
func (s *Server) RangesFormatting(context.Context, *protocol.DocumentRangesFormattingParams) ([]protocol.TextEdit, error) {
	panic("unimplemented")
}

// References implements protocol.Server.
func (s *Server) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	panic("unimplemented")
}

// Rename implements protocol.Server.
func (s *Server) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	panic("unimplemented")
}

// Resolve implements protocol.Server.
func (s *Server) Resolve(context.Context, *protocol.InlayHint) (*protocol.InlayHint, error) {
	panic("unimplemented")
}

// ResolveCodeAction implements protocol.Server.
func (s *Server) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	panic("unimplemented")
}

// ResolveCodeLens implements protocol.Server.
func (s *Server) ResolveCodeLens(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	panic("unimplemented")
}

// ResolveCompletionItem implements protocol.Server.
func (s *Server) ResolveCompletionItem(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	panic("unimplemented")
}

// ResolveDocumentLink implements protocol.Server.
func (s *Server) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	panic("unimplemented")
}

// ResolveWorkspaceSymbol implements protocol.Server.
func (s *Server) ResolveWorkspaceSymbol(context.Context, *protocol.WorkspaceSymbol) (*protocol.WorkspaceSymbol, error) {
	panic("unimplemented")
}

// SelectionRange implements protocol.Server.
func (s *Server) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	panic("unimplemented")
}

// SemanticTokensFull implements protocol.Server.
func (s *Server) SemanticTokensFull(context.Context, *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	panic("unimplemented")
}

// SemanticTokensFullDelta implements protocol.Server.
func (s *Server) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (any, error) {
	panic("unimplemented")
}

// SemanticTokensRange implements protocol.Server.
func (s *Server) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	panic("unimplemented")
}

// SetTrace implements protocol.Server.
func (s *Server) SetTrace(context.Context, *protocol.SetTraceParams) error {
	panic("unimplemented")
}

// Shutdown implements protocol.Server.
func (s *Server) Shutdown(context.Context) error {
	return nil
}

// SignatureHelp implements protocol.Server.
func (s *Server) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	panic("unimplemented")
}

// Subtypes implements protocol.Server.
func (s *Server) Subtypes(context.Context, *protocol.TypeHierarchySubtypesParams) ([]protocol.TypeHierarchyItem, error) {
	panic("unimplemented")
}

// Supertypes implements protocol.Server.
func (s *Server) Supertypes(context.Context, *protocol.TypeHierarchySupertypesParams) ([]protocol.TypeHierarchyItem, error) {
	panic("unimplemented")
}

// Symbol implements protocol.Server.
func (s *Server) Symbol(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	panic("unimplemented")
}

// TextDocumentContent implements protocol.Server.
func (s *Server) TextDocumentContent(context.Context, *protocol.TextDocumentContentParams) (*string, error) {
	panic("unimplemented")
}

// TypeDefinition implements protocol.Server.
func (s *Server) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	panic("unimplemented")
}

// WillCreateFiles implements protocol.Server.
func (s *Server) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	panic("unimplemented")
}

// WillDeleteFiles implements protocol.Server.
func (s *Server) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	panic("unimplemented")
}

// WillRenameFiles implements protocol.Server.
func (s *Server) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	panic("unimplemented")
}

// WillSave implements protocol.Server.
func (s *Server) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	panic("unimplemented")
}

// WillSaveWaitUntil implements protocol.Server.
func (s *Server) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	panic("unimplemented")
}

// WorkDoneProgressCancel implements protocol.Server.
func (s *Server) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	panic("unimplemented")
}
