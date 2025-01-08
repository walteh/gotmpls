package lsp

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

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
	documents sync.Map // map[string]string
	workspace string
	conn      *jsonrpc2.Conn
}

type handlerFunc func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error)

func NewServer(parser parser.TemplateParser, validator types.Validator, analyzer ast.PackageAnalyzer, generator diagnostic.Generator, debug bool) *Server {
	return &Server{
		parser:    parser,
		validator: validator,
		analyzer:  analyzer,
		generator: generator,
		debug:     debug,
	}
}

func (s *Server) Start(ctx context.Context, in io.ReadCloser, out io.WriteCloser) error {
	// Create a buffered stream with VSCode codec for proper LSP message formatting
	stream := jsonrpc2.NewBufferedStream(NewReadWriteCloser(in, out), jsonrpc2.VSCodeObjectCodec{})
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
	if s.debug {
		s.debugf("received request: %s", req.Method)
		if req.Params != nil {
			s.debugf("request params: %s", string(*req.Params))
		}
	}

	handler := s.router(req.Method)
	if handler == nil {
		if s.debug {
			s.debugf("unhandled method: %s", req.Method)
		}
		if !req.Notif {
			if err := conn.Reply(ctx, req.ID, nil); err != nil {
				s.debugf("error sending default reply: %v", err)
			}
		}
		return
	}

	result, err := handler(ctx, req)
	if err != nil {
		s.debugf("error handling %s: %v", req.Method, err)
		if !req.Notif {
			if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			}); err != nil {
				s.debugf("error sending error reply: %v", err)
			}
		}
		return
	}

	if !req.Notif && result != nil {
		if err := conn.Reply(ctx, req.ID, result); err != nil {
			s.debugf("error sending reply: %v", err)
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

func (s *Server) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}

	msg := fmt.Sprintf(format, args...)

	if s.conn != nil {
		params := &LogMessageParams{
			Type:    Info,
			Message: msg,
		}
		// Use the connection's notification method directly
		_ = s.conn.Notify(context.Background(), "window/logMessage", params)
	} else {
		fmt.Fprintf(os.Stderr, "Debug: %s\n", msg)
	}
}

func (s *Server) publishDiagnostics(ctx context.Context, uri string, diagnostics []Diagnostic) error {
	params := &PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	if s.debug {
		s.debugf("publishing %d diagnostics for %s", len(diagnostics), uri)
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
			s.debugf("  - %s at %v: %s", severity, d.Range, d.Message)
		}
	}

	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}
