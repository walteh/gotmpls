package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
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
	s.conn = jsonrpc2.NewConn(ctx, stream, s)
	<-ctx.Done()
	return ctx.Err()
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if s.debug {
		s.debugf("received request: %s", req.Method)
	}

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result, err = s.handleInitialize(ctx, req)
	case "initialized":
		result, err = nil, nil
	case "shutdown":
		result, err = nil, nil
	case "exit":
		result, err = nil, nil
	case "textDocument/didOpen":
		result, err = s.handleTextDocumentDidOpen(ctx, req)
	case "textDocument/didChange":
		result, err = s.handleTextDocumentDidChange(ctx, req)
	case "textDocument/didClose":
		result, err = s.handleTextDocumentDidClose(ctx, req)
	case "textDocument/hover":
		result, err = s.handleTextDocumentHover(ctx, req)
	case "textDocument/completion":
		result, err = s.handleTextDocumentCompletion(ctx, req)
	default:
		if s.debug {
			s.debugf("unhandled method: %s", req.Method)
		}
		result, err = nil, nil
	}

	if err != nil {
		if !req.Notif {
			errResp := &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			}

			if err := conn.ReplyWithError(ctx, req.ID, errResp); err != nil {
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

func (s *Server) handleInitialize(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling initialize request")
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
	s.debugf("workspace path: %s", s.workspace)

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

// uriToPath converts a URI to a filesystem path
func uriToPath(uri string) (string, error) {
	if !strings.HasPrefix(uri, "file://") {
		return "", errors.Errorf("unsupported URI scheme: %s", uri)
	}

	// Parse the URI
	u, err := url.Parse(uri)
	if err != nil {
		return "", errors.Errorf("failed to parse URI: %w", err)
	}

	// Convert the path to a filesystem path
	path := u.Path
	if path == "" {
		return "", errors.Errorf("empty path in URI: %s", uri)
	}

	// On Windows, remove the leading slash
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Clean the path
	path = filepath.Clean(path)

	return path, nil
}

func (s *Server) handleTextDocumentDidOpen(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling textDocument/didOpen")
	}

	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal didOpen params: %w", err)
	}

	s.documents.Store(params.TextDocument.URI, params.TextDocument.Text)
	return s.validateDocument(ctx, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) handleTextDocumentDidChange(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling textDocument/didChange")
	}

	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal didChange params: %w", err)
	}

	// For now, we'll just use the full content sync
	if len(params.ContentChanges) > 0 {
		s.documents.Store(params.TextDocument.URI, params.ContentChanges[0].Text)
		return s.validateDocument(ctx, params.TextDocument.URI, params.ContentChanges[0].Text)
	}

	return nil, nil
}

func (s *Server) handleTextDocumentDidClose(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling textDocument/didClose")
	}

	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal didClose params: %w", err)
	}

	s.documents.Delete(params.TextDocument.URI)
	return nil, nil
}

func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling textDocument/hover")
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
	}

	// TODO: Implement hover
	return nil, nil
}

func (s *Server) handleTextDocumentCompletion(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf("handling textDocument/completion")
	}

	var params CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, errors.Errorf("failed to unmarshal completion params: %w", err)
	}

	// TODO: Implement completion
	return nil, nil
}

func (s *Server) debugf(format string, args ...interface{}) {
	if s.debug {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func (s *Server) validateDocument(ctx context.Context, uri string, content string) (interface{}, error) {
	s.debugf("validating document: %s", uri)
	s.debugf("content:\n%s", content)

	// Convert URI to filesystem path for the document
	docPath, err := uriToPath(uri)
	if err != nil {
		s.debugf("document path error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  fmt.Sprintf("invalid document path: %v", err),
		}})
	}

	// Parse the template
	info, err := s.parser.Parse(ctx, []byte(content), docPath)
	if err != nil {
		s.debugf("parse error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf("parsed template info: %+v", info)

	// Analyze the package to get type information
	registry, err := s.analyzer.AnalyzePackage(ctx, s.workspace)
	if err != nil {
		s.debugf("package analysis error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf("analyzed package registry: %+v", registry)

	// Generate diagnostics
	diagnostics, err := s.generator.Generate(ctx, info, s.validator, registry)
	if err != nil {
		s.debugf("diagnostic generation error: %v", err)
		return nil, s.publishDiagnostics(ctx, uri, []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: 1, // Error
			Message:  err.Error(),
		}})
	}

	s.debugf("generated diagnostics: %+v", diagnostics)

	// Convert diagnostics to LSP format
	lspDiagnostics := make([]Diagnostic, 0)
	for _, d := range diagnostics.Errors {
		lspDiagnostics = append(lspDiagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: d.Column - 1},
				End:   Position{Line: d.EndLine - 1, Character: d.EndCol - 1},
			},
			Severity: 1, // Error
			Message:  d.Message,
		})
	}

	for _, d := range diagnostics.Warnings {
		lspDiagnostics = append(lspDiagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: d.Column - 1},
				End:   Position{Line: d.EndLine - 1, Character: d.EndCol - 1},
			},
			Severity: 2, // Warning
			Message:  d.Message,
		})
	}

	s.debugf("publishing %d diagnostics for %s", len(lspDiagnostics), uri)
	for _, d := range lspDiagnostics {
		severity := "error"
		if d.Severity == 2 {
			severity = "warning"
		}
		s.debugf("  - %s at %v: %s", severity, d.Range, d.Message)
	}

	err = s.publishDiagnostics(ctx, uri, lspDiagnostics)
	if err != nil {
		s.debugf("error publishing diagnostics: %v", err)
	}
	return nil, err
}

func (s *Server) publishDiagnostics(ctx context.Context, uri string, diagnostics []Diagnostic) error {
	s.debugf("publishing diagnostics: %v", diagnostics)

	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	// Marshal the notification to ensure proper Content-Length header
	notif := &jsonrpc2.Request{
		Method: "textDocument/publishDiagnostics",
		Notif:  true,
	}

	// Marshal params to RawMessage
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return errors.Errorf("failed to marshal diagnostic params: %w", err)
	}
	notif.Params = (*json.RawMessage)(&paramsBytes)

	return s.conn.Notify(ctx, notif.Method, notif.Params)
}
