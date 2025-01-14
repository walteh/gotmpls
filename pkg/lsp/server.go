package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/hover"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

type handlerFunc func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error)

// bufferedReadWriteCloser wraps a reader and writer with buffering
type bufferedReadWriteCloser struct {
	reader *bufio.Reader
	writer *bufio.Writer
	closer io.Closer
}

func newBufferedReadWriteCloser(r io.Reader, w io.Writer) io.ReadWriteCloser {
	return &bufferedReadWriteCloser{
		reader: bufio.NewReader(r),
		writer: bufio.NewWriter(w),
		closer: io.NopCloser(nil),
	}
}

func (b *bufferedReadWriteCloser) Read(p []byte) (n int, err error) {
	return b.reader.Read(p)
}

func (b *bufferedReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = b.writer.Write(p)
	if err != nil {
		return n, err
	}
	return n, b.writer.Flush()
}

func (b *bufferedReadWriteCloser) Close() error {
	return b.closer.Close()
}

// Server represents an LSP server instance
type Server struct {
	documents       sync.Map // map[string]string
	workspace       string
	conn            *jsonrpc2.Conn
	id              string
	extraLogWriters []io.Writer
	debug           bool
}

func NewServer(ctx context.Context, extraLogWriters ...io.Writer) *Server {
	return &Server{
		extraLogWriters: extraLogWriters,
		id:              xid.New().String(),
		conn:            nil,
		documents:       sync.Map{},
		workspace:       "",
	}
}

func Spawn(ctx context.Context, reader io.Reader, writer io.Writer, extraLogWriters ...io.Writer) error {
	server := NewServer(ctx, extraLogWriters...)
	return server.Run(ctx, reader, writer, extraLogWriters...)
}

func (s *Server) Run(ctx context.Context, reader io.Reader, writer io.Writer, extraLogWriters ...io.Writer) error {
	zerolog.Ctx(ctx).Info().Msg("starting LSP server - all logging will be redirected to LSP")

	// Create a buffered stream with VSCode codec for proper LSP message formatting
	stream := jsonrpc2.NewBufferedStream(newBufferedReadWriteCloser(reader, writer), jsonrpc2.VSCodeObjectCodec{})
	conn := jsonrpc2.NewConn(ctx, stream, s)
	s.conn = conn

	// Wait for either the connection to be closed or the context to be done
	select {
	case <-conn.DisconnectNotify():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	ctx = s.ApplyLSPWriter(ctx, conn, s.extraLogWriters...)
	if s.debug {
		s.debugf(ctx, "received request: %s", req.Method)
		if req.Params != nil {
			s.debugf(ctx, "request params: %s", string(*req.Params))
		}
	}

	ctx = zerolog.Ctx(ctx).With().Str("method", req.Method).Logger().WithContext(ctx)

	handler := s.router(ctx, req.Method)
	if handler == nil {
		if s.debug {
			s.debugf(ctx, "unhandled method: %s", req.Method)
		}
		if !req.Notif {
			if err := conn.Reply(ctx, req.ID, nil); err != nil {
				s.debugf(ctx, "error sending default reply: %v", err)
			}
		}
		return
	}

	result, err := handler(ctx, req)
	if err != nil {
		s.debugf(ctx, "error handling %s: %v", req.Method, err)
		if !req.Notif {
			if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			}); err != nil {
				s.debugf(ctx, "error sending error reply: %v", err)
			}
		}
		return
	}

	if !req.Notif && result != nil {
		if err := conn.Reply(ctx, req.ID, result); err != nil {
			s.debugf(ctx, "error sending reply: %v", err)
		}
	}
}

func (s *Server) router(ctx context.Context, method string) handlerFunc {
	zerolog.Ctx(ctx).Info().Str("method", method).Msg("routing request")
	switch method {
	case "initialize", "initialized":
		return s.handleInitialize
	// case "initialized":
	// 	return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	// 		s.debugf(ctx, "initialized")

	// 		zerolog.Ctx(ctx).Info().Str("params", string(*req.Params)).Msg("initialized")
	// 		return nil, nil
	// 	}
	case "shutdown":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
	case "exit":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
	case "textDocument/didOpen":
		return s.handleTextDocumentDidOpen
	case "textDocument/didChange":
		return s.handleTextDocumentDidChange
	case "textDocument/didClose":
		return s.handleTextDocumentDidClose
	case "textDocument/hover":
		return s.handleTextDocumentHover
	// case "textDocument/completion":
	// 	return s.handleTextDocumentCompletion
	case "$/setTrace":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
	case "$/cancelRequest":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
	case "workspace/didChangeConfiguration":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
	default:
		return nil
	}
}

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

func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/hover")
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
	}

	zerolog.Ctx(ctx).Debug().Msgf("hover request received: %+v", params)

	uri := s.normalizeURI(params.TextDocument.URI)

	// // Get document content
	// content, ok := s.getDocument(uri)
	// if !ok {
	// 	return nil, errors.Errorf("document not found: %s", uri)
	// }

	reg, err := ast.AnalyzePackage(ctx, uri)
	if err != nil {
		return nil, errors.Errorf("analyzing package for hover: %w", err)
	}

	content, _, ok := reg.GetTemplateFile(uri)
	if !ok {
		return nil, errors.Errorf("template %s not found, make sure its embeded", uri)
	}

	// // Parse the template
	info, err := parser.Parse(ctx, uri, []byte(content))
	if err != nil {
		return nil, errors.Errorf("parsing template for hover: %w", err)
	}

	pos := position.NewRawPositionFromLineAndColumn(params.Position.Line, params.Position.Character, string(content[params.Position.Character]), content)

	// hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, func(arg parser.VariableLocationOrType, th parser.TypeHint) []types.Type {
	// 	if arg.Type != nil {
	// 		return []types.Type{arg.Type}
	// 	}

	// 	if arg.Variable != nil {
	// 		// typ := arg.Variable.GetTypePaths(&th)

	// 		args := append([]string{th.LocalTypeName()}, strings.Split(arg.Variable.LongName(), ".")...)
	// 		scope := pkg.Package.Types.Scope().Lookup(args[0])
	// 	HERE:
	// 		for _, typ := range args[1:] {
	// 			if sig, ok := scope.Type().(*types.Struct); ok {
	// 				for i := range sig.NumFields() {
	// 					if sig.Field(i).Name() == typ {
	// 						scope = sig.Field(i)
	// 						break HERE
	// 					}
	// 				}
	// 			}
	// 		}

	// 		if sig, ok := scope.Type().(*types.Signature); ok {
	// 			typs := []types.Type{}
	// 			for i := range sig.Results().Len() {
	// 				typs = append(typs, sig.Results().At(i).Type())
	// 			}
	// 			return typs
	// 		}

	// 		return []types.Type{}
	// 	}

	// 	return []types.Type{}
	// })
	hoverInfo, err := hover.BuildHoverResponseFromParse(ctx, info, pos, reg)
	if err != nil {
		return nil, errors.Errorf("building hover response: %w", err)
	}

	if hoverInfo == nil {
		return nil, nil
	}

	hovers := make([]Hover, len(hoverInfo.Content))
	for i, hcontent := range hoverInfo.Content {
		hovers[i] = Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: hcontent,
			},
			Range: rangeToLSP(hoverInfo.Position.GetRange(content)),
		}
	}

	// TODO: Return more than one
	if len(hovers) > 0 {
		return &hovers[0], nil
	}

	return nil, nil
}

// rangeToLSP converts a position.Range to an LSP Range
func rangeToLSP(r position.Range) *Range {
	return &Range{
		Start: Position{
			Line:      r.Start.Line,
			Character: r.Start.Character,
		},
		End: Position{
			Line:      r.End.Line,
			Character: r.End.Character,
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
