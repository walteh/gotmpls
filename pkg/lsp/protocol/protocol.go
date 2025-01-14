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

	"go.lsp.dev/pkg/event"
	"golang.org/x/telemetry/crashmonitor"

	// "golang.org/x/tools/gopls/internal/util/bug"
	// "golang.org/x/tools/internal/event"
	// "golang.org/x/tools/internal/jsonrpc2"
	// jsonrpc2_v2 "golang.org/x/tools/internal/jsonrpc2_v2"
	// "golang.org/x/tools/internal/xcontext"
	"github.com/sourcegraph/jsonrpc2"
)

var (
	// RequestCancelledError should be used when a request is cancelled early.
	RequestCancelledError = &jsonrpc2.Error{Code: -32800, Message: "JSON RPC cancelled"}
)

type ClientCloser interface {
	Client
	io.Closer
}

type connSender interface {
	io.Closer

	Notify(ctx context.Context, method string, params any) error
	Call(ctx context.Context, method string, params, result any) error
}

type clientDispatcher struct {
	sender connSender
}

func (c *clientDispatcher) Close() error {
	return c.sender.Close()
}

// ClientDispatcher returns a Client that dispatches LSP requests across the
// given jsonrpc2 connection.
func ClientDispatcher(conn *jsonrpc2.Conn) ClientCloser {
	return &clientDispatcher{sender: clientConn{conn}}
}

type clientConn struct {
	conn *jsonrpc2.Conn
}

func (c clientConn) Close() error {
	return c.conn.Close()
}

func (c clientConn) Notify(ctx context.Context, method string, params any) error {
	return c.conn.Notify(ctx, method, params)
}

func (c clientConn) Call(ctx context.Context, method string, params any, result any) error {
	err := c.conn.Call(ctx, method, params, result)
	if ctx.Err() != nil {
		// Create a new ID for cancellation since we can't access the original
		cancelCall(ctx, c, jsonrpc2.ID{})
	}
	return err
}

// func ClientDispatcherV2(conn *jsonrpc2_v2.Connection) ClientCloser {
// 	return &clientDispatcher{clientConnV2{conn}}
// }

// type clientConnV2 struct {
// 	conn *jsonrpc2_v2.Connection
// }

// func (c clientConnV2) Close() error {
// 	return c.conn.Close()
// }

// func (c clientConnV2) Notify(ctx context.Context, method string, params any) error {
// 	return c.conn.Notify(ctx, method, params)
// }

// func (c clientConnV2) Call(ctx context.Context, method string, params any, result any) error {
// 	call := c.conn.Call(ctx, method, params)
// 	err := call.Await(ctx, result)
// 	if ctx.Err() != nil {
// 		detached := Detach(ctx)
// 		c.conn.Notify(detached, "$/cancelRequest", &CancelParams{ID: call.ID().Raw()})
// 	}
// 	return err
// }

// ServerDispatcher returns a Server that dispatches LSP requests across the
// given jsonrpc2 connection.
func ServerDispatcher(conn *jsonrpc2.Conn) Server {
	return &serverDispatcher{sender: clientConn{conn}}
}

// func ServerDispatcherV2(conn *jsonrpc2_v2.Connection) Server {
// 	return &serverDispatcher{sender: clientConnV2{conn}}
// }

type serverDispatcher struct {
	sender connSender
}

func Handlers(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &CancelHandler{handler: jsonrpc2.AsyncHandler(handler)}
}

type CancelHandler struct {
	handler jsonrpc2.Handler
}

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
	// Handle cancellation request
	conn.Reply(ctx, req.ID, nil)
}

func Call(ctx context.Context, conn *jsonrpc2.Conn, method string, params any, result any) error {
	err := conn.Call(ctx, method, params, result)
	if ctx.Err() != nil {
		// Create a new ID for cancellation since we can't access the original
		cancelCall(ctx, clientConn{conn}, jsonrpc2.ID{})
	}
	return err
}

func cancelCall(ctx context.Context, sender connSender, id jsonrpc2.ID) {
	ctx = Detach(ctx)
	ctx = event.Start(ctx, "protocol.canceller")

	// Note that only *jsonrpc2.ID implements json.Marshaler.
	sender.Notify(ctx, "$/cancelRequest", &CancelParams{ID: &id})
}

// UnmarshalJSON unmarshals msg into the variable pointed to by
// params. In JSONRPC, optional messages may be
// "null", in which case it is a no-op.
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

// NonNilSlice returns x, or an empty slice if x was nil.
//
// (Many slice fields of protocol structs must be non-nil
// to avoid being encoded as JSON "null".)
func NonNilSlice[T comparable](x []T) []T {
	if x == nil {
		return []T{}
	}
	return x
}

// recoverHandlerPanic recovers from panics in handlers and logs them using zerolog
func recoverHandlerPanic(method string) {
	if !crashmonitor.Supported() {
		defer func() {
			if x := recover(); x != nil {
				// Use zerolog for structured logging
				event.Error(context.Background(), "panic in LSP handler", fmt.Errorf("method %s: %v", method, x))
				panic(x) // Re-panic after logging
			}
		}()
	}
}

// ClientHandler implements jsonrpc2.Handler for client-side LSP message handling
type ClientHandler struct {
	client  Client
	handler jsonrpc2.Handler
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

// ServerHandler implements jsonrpc2.Handler for server-side LSP message handling
type ServerHandler struct {
	server  Server
	handler jsonrpc2.Handler
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

func NewClientHandler(client Client, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &ClientHandler{client: client, handler: handler}
}

func NewServerHandler(server Server, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return &ServerHandler{server: server, handler: handler}
}
