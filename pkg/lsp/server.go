package lsp

import (
	"bufio"
	"context"
	"io"
	"strings"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
)

type ServerSpawner struct {
	id        string
	parser    parser.TemplateParser
	validator types.Validator
	analyzer  ast.PackageAnalyzer
	generator diagnostic.Generator
	debug     bool
}

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

func NewServer(parser parser.TemplateParser, validator types.Validator, analyzer ast.PackageAnalyzer, generator diagnostic.Generator, debug bool) *ServerSpawner {

	return &ServerSpawner{
		parser:    parser,
		validator: validator,
		analyzer:  analyzer,
		generator: generator,
		debug:     debug,
		id:        xid.New().String(), // logger:    &logger,
	}
}

type Server struct {
	server          *ServerSpawner
	documents       sync.Map // map[string]string
	workspace       string
	conn            *jsonrpc2.Conn
	id              string
	extraLogWriters []io.Writer
	debug           bool
}

func (s *ServerSpawner) Spawn(ctx context.Context, reader io.Reader, writer io.Writer, extraLogWriters ...io.Writer) error {
	// ctx = s.logger.WithContext(ctx)
	zerolog.Ctx(ctx).Info().Msg("starting LSP server - all logging will be redirected to LSP")

	spawn := &Server{
		server:          s,
		extraLogWriters: extraLogWriters,
		id:              xid.New().String(),
		conn:            nil,
		documents:       sync.Map{},
		workspace:       "",
		debug:           s.debug,
	}

	// Create a buffered stream with VSCode codec for proper LSP message formatting
	stream := jsonrpc2.NewBufferedStream(newBufferedReadWriteCloser(reader, writer), jsonrpc2.VSCodeObjectCodec{})
	conn := jsonrpc2.NewConn(ctx, stream, spawn)
	spawn.conn = conn

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

	handler := s.router(req.Method)
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

func (s *Server) router(method string) handlerFunc {
	switch method {
	case "initialize":
		return s.handleInitialize
	case "initialized":
		return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
			return nil, nil
		}
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
	case "textDocument/completion":
		return s.handleTextDocumentCompletion
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

func (s *Server) publishDiagnostics(ctx context.Context, uri string, diagnostics []Diagnostic) error {
	params := &PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	if s.debug {
		s.debugf(ctx, "publishing %d diagnostics for %s", len(diagnostics), uri)
		for _, d := range diagnostics {
			severity := "unknown"
			switch d.Severity {
			case 1:
				severity = "error"
			case 2:
				severity = "warning"
			case 3:
				severity = "information"
			case 4:
				severity = "hint"
			}
			s.debugf(ctx, "  - %s at %v: %s", severity, d.Range, d.Message)
		}
	}

	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

// normalizeURI ensures consistent URI handling by removing the file:// prefix if present
// and converting to a clean path
func (s *Server) normalizeURI(uri string) string {
	// Remove file:// prefix if present
	if strings.HasPrefix(uri, "file://") {
		uri = uri[7:]
	}
	// Remove any leading slashes for consistency
	uri = strings.TrimLeft(uri, "/")
	return uri
}

// getDocument retrieves a document from the server's store using a normalized URI
func (s *Server) getDocument(uri string) (string, bool) {
	normalizedURI := s.normalizeURI(uri)
	content, ok := s.documents.Load(normalizedURI)
	if !ok {
		// Try with the original URI as fallback
		content, ok = s.documents.Load(uri)
	}
	if !ok {
		return "", false
	}
	text, ok := content.(string)
	return text, ok
}

// storeDocument stores a document in the server's store using a normalized URI
func (s *Server) storeDocument(uri string, content string) {
	normalizedURI := s.normalizeURI(uri)
	s.documents.Store(normalizedURI, content)
}
