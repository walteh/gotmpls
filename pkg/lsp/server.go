package lsp

import (
	"bufio"
	"context"
	"io"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
)

type Server struct {
	parser    parser.TemplateParser
	validator types.Validator
	analyzer  ast.PackageAnalyzer
	generator diagnostic.Generator
	debug     bool
	// logger    *zerolog.Logger
	documents sync.Map // map[string]string
	workspace string
	conn      *jsonrpc2.Conn
	id        string
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

func NewServer(parser parser.TemplateParser, validator types.Validator, analyzer ast.PackageAnalyzer, generator diagnostic.Generator, debug bool) *Server {

	return &Server{
		parser:    parser,
		validator: validator,
		analyzer:  analyzer,
		generator: generator,
		debug:     debug,
		id:        xid.New().String(), // logger:    &logger,
	}
}

func (s *Server) Start(ctx context.Context, reader io.Reader, writer io.Writer) error {
	// ctx = s.logger.WithContext(ctx)
	zerolog.Ctx(ctx).Info().Msg("starting LSP server - all logging will be redirected to LSP")

	// Create a buffered stream with VSCode codec for proper LSP message formatting
	stream := jsonrpc2.NewBufferedStream(newBufferedReadWriteCloser(reader, writer), jsonrpc2.VSCodeObjectCodec{})
	conn := jsonrpc2.NewConn(ctx, stream, s)
	s.conn = conn

	// // Log takeover message
	// logger.Info().Msg("LSP server taking over logging output")

	// Wait for either the connection to be closed or the context to be done
	select {
	case <-conn.DisconnectNotify():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	ctx = s.ApplyLSPWriter(ctx, conn)
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
