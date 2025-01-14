// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sourcegraph/jsonrpc2"
	"go.lsp.dev/pkg/event"
	"golang.org/x/telemetry/crashmonitor"
)

// Core protocol constants and types
var (
	RequestCancelledError = &jsonrpc2.Error{Code: -32800, Message: "JSON RPC cancelled"}
)

// Dispatcher types and interfaces
type ClientCloser interface {
	Client
	io.Closer
}

type clientDispatcher struct {
	sender *jsonrpc2.Conn
}

type serverDispatcher struct {
	sender *jsonrpc2.Conn
}

// Dispatcher constructors and methods
func (c *clientDispatcher) Close() error {
	return c.sender.Close()
}

func NewClientDispatcher(conn *jsonrpc2.Conn) ClientCloser {
	return &clientDispatcher{sender: conn}
}

func NewServerDispatcher(conn *jsonrpc2.Conn) Server {
	return &serverDispatcher{sender: conn}
}

// Handler types and constructors
type CancelHandler struct {
	handler jsonrpc2.Handler
}

type ClientHandler struct {
	client  Client
	handler jsonrpc2.Handler
}

type ServerHandler struct {
	server  Server
	handler jsonrpc2.Handler
}

func Handlers(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &CancelHandler{handler: jsonrpc2.AsyncHandler(handler)}
}

func NewClientHandler(client Client, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &ClientHandler{client: client, handler: handler}
}

func NewServerHandler(server Server, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &ServerHandler{server: server, handler: handler}
}

// Handler implementations
func (h *CancelHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Method != "$/cancelRequest" {
		if ctx.Err() != nil {
			ctx = Detach(ctx)
			conn.ReplyWithError(ctx, req.ID, RequestCancelledError)
			return
		}
		h.handler.Handle(ctx, conn, req)
		return
	}
	var params CancelParams
	if err := UnmarshalJSON(req.Params, &params); err != nil {
		sendParseError(ctx, conn, req, err)
		return
	}
	conn.Reply(ctx, req.ID, nil)
}

func (h *ClientHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if ctx.Err() != nil {
		ctx = Detach(ctx)
		conn.ReplyWithError(ctx, req.ID, RequestCancelledError)
		return
	}
	handled, err := clientDispatch(ctx, h.client, conn, req)
	if handled || err != nil {
		if err != nil {
			if jsonErr, ok := err.(*jsonrpc2.Error); ok {
				conn.ReplyWithError(ctx, req.ID, jsonErr)
			} else {
				conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInternalError,
					Message: err.Error(),
				})
			}
		}
		return
	}
	h.handler.Handle(ctx, conn, req)
}

func (h *ServerHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if ctx.Err() != nil {
		ctx = Detach(ctx)
		conn.ReplyWithError(ctx, req.ID, RequestCancelledError)
		return
	}
	handled, err := serverDispatch(ctx, h.server, conn, req)
	if handled || err != nil {
		if err != nil {
			if jsonErr, ok := err.(*jsonrpc2.Error); ok {
				conn.ReplyWithError(ctx, req.ID, jsonErr)
			} else {
				conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInternalError,
					Message: err.Error(),
				})
			}
		}
		return
	}
	h.handler.Handle(ctx, conn, req)
}

// Utility functions used by generated code
func reply_fwd(ctx context.Context, conn *jsonrpc2.Conn, id *jsonrpc2.Request, result any, err error) error {
	if err != nil {
		return conn.ReplyWithError(ctx, id.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: err.Error(),
		})
	}
	return conn.Reply(ctx, id.ID, result)
}

func Call(ctx context.Context, conn *jsonrpc2.Conn, method string, params any, result any) error {
	err := conn.Call(ctx, method, params, result)
	if ctx.Err() != nil {
		cancelRequest(ctx, conn)
	}
	return err
}

func cancelRequest(ctx context.Context, conn *jsonrpc2.Conn) {
	ctx = Detach(ctx)
	ctx = event.Start(ctx, "protocol.canceller")
	conn.Notify(ctx, "$/cancelRequest", &CancelParams{})
}

// JSON handling utilities
func UnmarshalJSON(msg *json.RawMessage, v any) error {
	if msg == nil {
		return nil
	}
	if len(*msg) == 0 || bytes.Equal(*msg, []byte("null")) {
		return nil
	}
	return json.Unmarshal(*msg, v)
}

func sendParseError(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request, err error) error {
	return conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeParseError,
		Message: err.Error(),
	})
}

// Helper functions
func NonNilSlice[T comparable](x []T) []T {
	if x == nil {
		return []T{}
	}
	return x
}

func recoverHandlerPanic(method string) {
	if !crashmonitor.Supported() {
		defer func() {
			if x := recover(); x != nil {
				event.Error(context.Background(), "panic in LSP handler", fmt.Errorf("method %s: %v", method, x))
				panic(x)
			}
		}()
	}
}

// LSP method implementations for serverDispatcher
func (s *serverDispatcher) PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) error {
	return s.sender.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *serverDispatcher) ShowMessage(ctx context.Context, params *ShowMessageParams) error {
	return s.sender.Notify(ctx, "window/showMessage", params)
}

func (s *serverDispatcher) LogMessage(ctx context.Context, params *LogMessageParams) error {
	return s.sender.Notify(ctx, "window/logMessage", params)
}

// LSP method implementations for clientDispatcher
func (c *clientDispatcher) Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
	var result InitializeResult
	if err := c.sender.Call(ctx, "initialize", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *clientDispatcher) Initialized(ctx context.Context, params *InitializedParams) error {
	return c.sender.Notify(ctx, "initialized", params)
}

func (c *clientDispatcher) Shutdown(ctx context.Context) error {
	return c.sender.Call(ctx, "shutdown", nil, nil)
}

func (c *clientDispatcher) Exit(ctx context.Context) error {
	return c.sender.Notify(ctx, "exit", nil)
}

func (c *clientDispatcher) DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) error {
	return c.sender.Notify(ctx, "textDocument/didOpen", params)
}

func (c *clientDispatcher) DidChange(ctx context.Context, params *DidChangeTextDocumentParams) error {
	return c.sender.Notify(ctx, "textDocument/didChange", params)
}

func (c *clientDispatcher) DidClose(ctx context.Context, params *DidCloseTextDocumentParams) error {
	return c.sender.Notify(ctx, "textDocument/didClose", params)
}

func (c *clientDispatcher) DidSave(ctx context.Context, params *DidSaveTextDocumentParams) error {
	return c.sender.Notify(ctx, "textDocument/didSave", params)
}
