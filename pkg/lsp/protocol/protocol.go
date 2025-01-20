// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"
	"gitlab.com/tozd/go/errors"
)

// Core protocol constants and types
var (
	RequestCancelledError = &jrpc2.Error{Code: -32800, Message: "JSON RPC cancelled"}
)

// Dispatcher types and interfaces

type ServerDispatcher struct {
	client *jrpc2.Client
}

func (c *ServerDispatcher) Notify(ctx context.Context, method string, params interface{}) error {
	// zerolog.Ctx(ctx).Debug().
	// 	Str("method", method).
	// 	Interface("params", params).
	// 	Msg("forwarding notification to gopls")
	return c.client.Notify(ctx, method, params)
}

func (c *ServerDispatcher) Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error) {
	// zerolog.Ctx(ctx).Debug().

	res, err := c.client.Call(ctx, method, params)
	if err != nil {

		return nil, err
	}

	return res, nil
}

type ClientDispatcher struct {
	serverOpts *jrpc2.ServerOptions
	server     *jrpc2.Server
}

func (c *ClientDispatcher) Notify(ctx context.Context, method string, params any) error {
	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackRequestRaw(ctx, method, params)
	}

	if err := c.server.Notify(ctx, method, params); err != nil {
		return err
	}

	return nil
}

func (c *ClientDispatcher) Callback(ctx context.Context, method string, params any) (*jrpc2.Response, error) {
	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackRequestRaw(ctx, method, params)
	}

	res, err := c.server.Callback(ctx, method, params)
	if err != nil {
		return nil, err
	}

	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackResponse(ctx, res)
	}

	return res, nil
}

func NewCallbackClient(server *jrpc2.Server, serverOpts *jrpc2.ServerOptions) *ClientDispatcher {
	return &ClientDispatcher{server: server, serverOpts: serverOpts}
}

func NewCallbackServer(server *jrpc2.Client) *ServerDispatcher {
	return &ServerDispatcher{client: server}
}

type ServerInstance struct {
	server         *jrpc2.Server
	callbackClient *ClientDispatcher
	creationCtx    context.Context

	ServerOpts    *jrpc2.ServerOptions
	methods       jrpc2.Assigner
	backgroundCmd *exec.Cmd
	rpcTracker    *RPCTracker // Internal field for tracking RPC messages
}

func (me *ServerInstance) AddArgsToBackgroundCmd(args ...string) {
	if me.backgroundCmd == nil {
		return
	}
	me.backgroundCmd.Args = append(me.backgroundCmd.Args, args...)
}

func (s *ServerInstance) Server() *jrpc2.Server {
	if s.server == nil {
		panic("server not started")
	}
	return s.server
}

func (s *ServerInstance) CallbackClient() *ClientDispatcher {
	if s.callbackClient == nil {
		panic("callback client not started")
	}
	return s.callbackClient
}

func (s *ServerInstance) newContext() context.Context {
	ctx := ApplyClientToZerolog(s.creationCtx, s.callbackClient)
	if s.rpcTracker != nil {
		ctx = ContextWithRPCTracker(ctx, s.rpcTracker)
	}
	return ctx
}

func (s *ServerInstance) StartAndDetach(reader io.Reader, writer io.WriteCloser) (*jrpc2.Server, error) {
	multiLogger := &MultiRPCLogger{}

	if s.ServerOpts.RPCLog == nil {
		s.ServerOpts.RPCLog = multiLogger
	} else {
		multiLogger.AddLogger(s.ServerOpts.RPCLog)
		s.ServerOpts.RPCLog = multiLogger
	}

	if s.rpcTracker != nil {
		multiLogger.AddLogger(s.rpcTracker)
	}

	if s.backgroundCmd != nil {
		for _, logger := range multiLogger.loggers {
			if tl, ok := logger.(*rpcTestLogger); ok {
				if tl.enableBackgroundCmdToStderr {
					s.backgroundCmd.Stderr = os.Stderr
				}
			}
		}

		if err := s.backgroundCmd.Start(); err != nil {
			return nil, errors.Errorf("starting external server: %w", err)
		}

		go func() {
			if err := s.backgroundCmd.Wait(); err != nil {
				panic(err)
			}
		}()
	}

	s.server = jrpc2.NewServer(s.methods, s.ServerOpts)

	s.callbackClient = NewCallbackClient(s.server, s.ServerOpts)

	s.server = s.server.Start(channel.LSP(reader, writer))

	return s.server, nil
}

func (s *ServerInstance) StartAndDetachWithClient(opts *jrpc2.ClientOptions) (*jrpc2.Server, *ServerDispatcher, error) {
	pr, pw := io.Pipe()

	server, err := s.StartAndDetach(pr, pw)
	if err != nil {
		return nil, nil, errors.Errorf("starting external server: %w", err)
	}

	opts.OnCallback = func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		var params interface{}
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, err
		}
		return s.server.Callback(ctx, r.Method(), params)
	}

	return server, NewCallbackServer(jrpc2.NewClient(channel.LSP(pr, pw), opts)), nil
}

func (s *ServerInstance) StartAndWait(reader io.Reader, writer io.WriteCloser) error {

	server, err := s.StartAndDetach(reader, writer)
	if err != nil {
		return errors.Errorf("starting external server: %w", err)
	}

	return server.Wait()
}

func NewServerInstance(ctx context.Context, server Server, opts *jrpc2.ServerOptions) *ServerInstance {
	methods := buildServerDispatchMap(server)
	if opts == nil {
		opts = &jrpc2.ServerOptions{}
	}

	instance := &ServerInstance{creationCtx: ctx}

	opts.AllowPush = true
	opts.NewContext = instance.newContext
	instance.ServerOpts = opts
	instance.methods = methods
	return instance
}

// Helper functions
func NonNilSlice[T comparable](x []T) []T {
	if x == nil {
		return []T{}
	}
	return x
}

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

		err := method(ctx, &params)

		return nil, err
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

		err := method(ctx)

		return nil, err
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
		err = res.UnmarshalResult(result)
		return err
	}

	return nil
}

func createEmptyResultCallback[I any](ctx context.Context, client Callbacker, method string, params *I) error {
	_, err := client.Callback(ctx, method, params)
	return err
}

func createEmptyCallback(ctx context.Context, client Callbacker, method string) error {
	_, err := client.Callback(ctx, method, nil)
	return err
}

func createEmptyParamsCallback[O any](ctx context.Context, client Callbacker, method string, result *O) error {
	res, err := client.Callback(ctx, method, nil)
	if err != nil {
		return err
	}

	if result != nil {
		err = res.UnmarshalResult(result)
		return err
	}

	return nil
}

func createNotify[I any](ctx context.Context, client Callbacker, method string, params *I) error {
	err := client.Notify(ctx, method, params)
	return err
}

func createEmptyNotify(ctx context.Context, client Callbacker, method string) error {
	err := client.Notify(ctx, method, nil)
	return err
}

// GetRPCTracker returns the RPCTracker instance for testing purposes
func (s *ServerInstance) GetRPCTracker() *RPCTracker {
	return s.rpcTracker
}

// SetRPCTracker sets the RPCTracker instance for testing purposes
func (s *ServerInstance) SetRPCTracker(tracker *RPCTracker) {
	s.rpcTracker = tracker
}
