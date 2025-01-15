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
	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

// Core protocol constants and types
var (
	RequestCancelledError = &jrpc2.Error{Code: -32800, Message: "JSON RPC cancelled"}
)

// Dispatcher types and interfaces

type CallbackServer struct {
	client *jrpc2.Client
}

func (c *CallbackServer) Notify(ctx context.Context, method string, params interface{}) error {
	zerolog.Ctx(ctx).Debug().
		Str("method", method).
		Interface("params", params).
		Msg("🔄 forwarding notification to gopls")
	return c.client.Notify(ctx, method, params)
}

func (c *CallbackServer) Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error) {
	zerolog.Ctx(ctx).Debug().
		Str("method", method).
		Interface("params", params).
		Msg("🎯 forwarding callback to gopls")

	res, err := c.client.Call(ctx, method, params)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("method", method).
			Err(err).
			Msg("❌ gopls callback error")
		return nil, err
	}

	zerolog.Ctx(ctx).Debug().
		Str("method", method).
		Str("result", res.ResultString()).
		Msg("✅ received response from gopls")

	return res, nil
}

type CallbackClient struct {
	serverOpts *jrpc2.ServerOptions
	server     *jrpc2.Server
}

func (c *CallbackClient) Notify(ctx context.Context, method string, params any) error {
	if rl, ok := c.serverOpts.RPCLog.(CallbackRPCLogger); ok {
		rl.LogCallbackRequestRaw(ctx, method, params)
	}

	if err := c.server.Notify(ctx, method, params); err != nil {
		return err
	}

	return nil
}

func (c *CallbackClient) Callback(ctx context.Context, method string, params any) (*jrpc2.Response, error) {
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

func NewCallbackClient(server *jrpc2.Server, serverOpts *jrpc2.ServerOptions) *CallbackClient {
	return &CallbackClient{server: server, serverOpts: serverOpts}
}

func NewCallbackServer(server *jrpc2.Client) *CallbackServer {
	return &CallbackServer{client: server}
}

type ServerInstance struct {
	server         *jrpc2.Server
	callbackClient *CallbackClient
	creationCtx    context.Context

	ServerOpts    *jrpc2.ServerOptions
	methods       jrpc2.Assigner
	backgroundCmd *exec.Cmd
	// backgroundCmdFlags []string
}

func (s *ServerInstance) Server() *jrpc2.Server {
	if s.server == nil {
		panic("server not started")
	}
	return s.server
}

func (s *ServerInstance) CallbackClient() *CallbackClient {
	if s.callbackClient == nil {
		panic("callback client not started")
	}
	return s.callbackClient
}

func (s *ServerInstance) newContext() context.Context {
	return ApplyClientToZerolog(s.creationCtx, s.callbackClient)
}

func (s *ServerInstance) StartAndDetach(reader io.Reader, writer io.WriteCloser) (*jrpc2.Server, error) {
	if s.backgroundCmd != nil {
		if tl, ok := s.ServerOpts.RPCLog.(*rpcTestLogger); ok {
			if tl.enableBackgroundCmdToStderr {
				s.backgroundCmd.Stderr = os.Stderr
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

func (s *ServerInstance) StartAndWait(reader io.Reader, writer io.WriteCloser) error {

	server, err := s.StartAndDetach(reader, writer)
	if err != nil {
		return errors.Errorf("starting external server: %w", err)
	}

	return server.Wait()
}

func NewGoplsServerInstance(ctx context.Context) (*ServerInstance, error) {

	ctx = zerolog.New(os.Stderr).With().Str("name", "gopls").Logger().WithContext(ctx)

	cmd := exec.CommandContext(ctx, "gopls", "-v", "-vv", "-rpc.trace")
	copts := &jrpc2.ClientOptions{
		// Logger: func(msg string) {
		// 	zerolog.Ctx(ctx).Info().Msgf("gopls [client]: %s", msg)
		// },
	}
	sopts := &jrpc2.ServerOptions{
		// Logger: func(msg string) {
		// 	zerolog.Ctx(ctx).Info().Msgf("gopls [server]: %s", msg)
		// },
	}

	inst, err := NewCmdServerInstance(ctx, cmd, copts, sopts)
	if err != nil {
		return nil, errors.Errorf("creating server instance: %w", err)
	}

	return inst, nil
}

func NewCmdServerInstance(ctx context.Context, cmd *exec.Cmd, copts *jrpc2.ClientOptions, sopts *jrpc2.ServerOptions) (*ServerInstance, error) {
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Errorf("getting stdout pipe: %w", err)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.Errorf("getting stdin pipe: %w", err)
	}

	sopts.AllowPush = true

	type handlerft struct {
		f func(ctx context.Context, r *jrpc2.Request) (interface{}, error)
	}

	handlerf := &handlerft{
		f: func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
			return nil, nil
		},
	}

	copts.OnCallback = handlerf.f

	client := jrpc2.NewClient(channel.LSP(out, in), copts)
	cbs := NewCallbackServer(client)
	instance := NewServerInstance(ctx, cbs, sopts)

	handlerf.f = func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		var params interface{}
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, err
		}
		return instance.server.Callback(ctx, r.Method(), params)
	}

	instance.backgroundCmd = cmd

	return instance, nil
}

func NewServerInstance(ctx context.Context, server Server, opts *jrpc2.ServerOptions) *ServerInstance {
	methods := buildServerDispatchMap(server)
	if opts == nil {
		opts = &jrpc2.ServerOptions{}
	}

	opts.AllowPush = true

	instance := &ServerInstance{creationCtx: ctx}

	opts.NewContext = instance.newContext

	instance.ServerOpts = opts
	instance.methods = methods
	return instance
}

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
