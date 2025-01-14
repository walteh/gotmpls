// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	"io"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

// Core protocol constants and types
var (
	RequestCancelledError = &jrpc2.Error{Code: -32800, Message: "JSON RPC cancelled"}
)

// Dispatcher types and interfaces
type ClientCloser interface {
	Client
	io.Closer
}

type ServerCloser interface {
	Server
	io.Closer
}

type CallbackServer struct {
	server *jrpc2.Client
}

func (c *CallbackServer) Notify(ctx context.Context, method string, params interface{}) error {
	return c.server.Notify(ctx, method, params)
}

func (c *CallbackServer) Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error) {
	return c.server.Call(ctx, method, params)
}

type CallbackClient struct {
	serverOpts *jrpc2.ServerOptions
	client     *jrpc2.Server
}

func (c *CallbackClient) Notify(ctx context.Context, method string, params any) error {
	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackRequestRaw(ctx, method, params)
	}

	if err := c.client.Notify(ctx, method, params); err != nil {
		return err
	}

	return nil
}

func (c *CallbackClient) Callback(ctx context.Context, method string, params any) (*jrpc2.Response, error) {
	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackRequestRaw(ctx, method, params)
	}

	res, err := c.client.Callback(ctx, method, params)
	if err != nil {
		return nil, err
	}

	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackResponse(ctx, res)
	}

	return res, nil
}

func NewCallbackClient(server *jrpc2.Server, serverOpts *jrpc2.ServerOptions) *CallbackClient {
	return &CallbackClient{client: server, serverOpts: serverOpts}
}

func NewCallbackServer(server *jrpc2.Client) *CallbackServer {
	return &CallbackServer{server: server}
}

func Handlers(h handler.Func) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		if req.Method() == "$/cancelRequest" {
			var params CancelParams
			if err := req.UnmarshalParams(&params); err != nil {
				return nil, err
			}
			return nil, nil
		}
		if ctx.Err() != nil {
			ctx = Detach(ctx)
			return nil, RequestCancelledError
		}
		return h(ctx, req)
	})
}

// func NewClientServer(client Client, opts *jrpc2.ServerOptions) *jrpc2.Server {

// 	methods := buildClientDispatchMap(client)

// 	// methods["$/cancelRequest"] = handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
// 	// 	var params CancelParams
// 	// 	if err := req.UnmarshalParams(&params); err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// 	return nil, nil
// 	// })

// 	return jrpc2.NewServer(methods, opts)
// }

func NewServerServer(ctx context.Context, server Server, opts *jrpc2.ServerOptions) (*jrpc2.Server, *CallbackClient) {
	methods := buildServerDispatchMap(server)
	if opts == nil {
		opts = &jrpc2.ServerOptions{}
	}

	opts.AllowPush = true

	var callbackServer *CallbackClient = nil

	opts.NewContext = func() context.Context {
		if callbackServer == nil {
			return ctx
		}

		return ApplyClientToZerolog(ctx, callbackServer)
	}

	// Create server with method handlers
	result := jrpc2.NewServer(methods, opts)

	callbackServer = NewCallbackClient(result, opts)

	return result, callbackServer
}

// Utility functions used by generated code
// func reply_fwd(ctx context.Context, _ *jrpc2.Server, req *jrpc2.Request, result interface{}, err error) (any, error) {
// 	if err != nil {
// 		return nil, &jrpc2.Error{
// 			Code:    -32603,
// 			Message: err.Error(),
// 		}
// 	}
// 	if req.IsNotification() {
// 		return nil, nil
// 	}
// 	return result, nil
// }

func Call(ctx context.Context, client *jrpc2.Client, method string, params interface{}, result interface{}) error {
	rsp, err := client.Call(ctx, method, params)
	if err != nil {
		return err
	}
	if result != nil {
		return rsp.UnmarshalResult(result)
	}
	return nil
}

// func cancelRequest(ctx context.Context, client *jrpc2.Client) {
// 	ctx = Detach(ctx)
// 	ctx = event.Start(ctx, "protocol.canceller")
// 	client.Notify(ctx, "$/cancelRequest", &CancelParams{})
// }

// Helper functions
func NonNilSlice[T comparable](x []T) []T {
	if x == nil {
		return []T{}
	}
	return x
}

// func recoverHandlerPanic(method string) {
// 	if !crashmonitor.Supported() {
// 		defer func() {
// 			if x := recover(); x != nil {
// 				event.Error(context.Background(), "panic in LSP handler", fmt.Errorf("method %s: %v", method, x))
// 				panic(x)
// 			}
// 		}()
// 	}
// }

func newParseError(err error) *jrpc2.Error {
	return &jrpc2.Error{
		Code:    -32700, // Parse error
		Message: err.Error(),
	}
}

func createHandler[T any, O any](method func(ctx context.Context, params *T) (O, error)) handler.Func {
	return handler.New(func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		ctx = ApplyRequestToZerolog(ctx, r)
		var params T
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, newParseError(err)
		}
		result, err := method(ctx, &params)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func createEmptyResultHandler[T any](method func(ctx context.Context, params *T) error) handler.Func {
	return handler.New(func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		ctx = ApplyRequestToZerolog(ctx, r)
		var params T
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, newParseError(err)
		}
		return nil, method(ctx, &params)
	})
}

func createEmptyParamsHandler[T any](method func(ctx context.Context) (T, error)) handler.Func {
	return handler.New(func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		ctx = ApplyRequestToZerolog(ctx, r)
		result, err := method(ctx)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func createEmptyHandler(method func(ctx context.Context) error) handler.Func {
	return handler.New(func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		ctx = ApplyRequestToZerolog(ctx, r)
		return nil, method(ctx)
	})
}

type Callbacker interface {
	Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error)
	Notify(ctx context.Context, method string, params interface{}) error
}

func createCallback[I any, O any](ctx context.Context, client Callbacker, method string, params *I, result *O) error {
	res, err := client.Callback(ctx, method, params)
	if err != nil {
		return err
	}
	if result != nil {
		return res.UnmarshalResult(result)
	}
	return nil
}

func createEmptyResultCallback[I any](ctx context.Context, client Callbacker, method string, params *I) error {
	_, err := client.Callback(ctx, method, params)
	if err != nil {
		return err
	}
	return nil
}

func createEmptyCallback(ctx context.Context, client Callbacker, method string) error {
	_, err := client.Callback(ctx, method, nil)
	if err != nil {
		return err
	}
	return nil
}

func createEmptyParamsCallback[O any](ctx context.Context, client Callbacker, method string, result *O) error {
	res, err := client.Callback(ctx, method, nil)
	if err != nil {
		return err
	}
	if result != nil {
		return res.UnmarshalResult(result)
	}
	return nil
}

func createNotify[I any](ctx context.Context, client Callbacker, method string, params *I) error {
	return client.Notify(ctx, method, params)
}

func createEmptyNotify(ctx context.Context, client Callbacker, method string) error {
	return client.Notify(ctx, method, nil)
}
