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
	instance *Instance
}

type ClientDispatcher struct {
	instance *Instance
}

type ClientCaller struct {
	client *jrpc2.Client
}

type ServerCaller struct {
	client *jrpc2.Client
}

type Instance struct {
	server              *jrpc2.Server
	methods             handler.Map
	serverOpts          *jrpc2.ServerOptions
	creationCtx         context.Context
	rpcTracker          *RPCTracker // Internal field for tracking RPC messages
	backgroundCmd       *exec.Cmd
	logForwardingClient Client
}

func (me *Instance) ServerOpts() *jrpc2.ServerOptions {
	return me.serverOpts
}

func (me *ServerDispatcher) Instance() *Instance {
	return me.instance
}

func (me *ClientDispatcher) Instance() *Instance {
	return me.instance
}

func (me *Instance) AddArgsToBackgroundCmd(args ...string) {
	if me.backgroundCmd == nil {
		return
	}
	me.backgroundCmd.Args = append(me.backgroundCmd.Args, args...)
}

func (s *Instance) newContext() context.Context {
	ctx := s.creationCtx
	if s.rpcTracker != nil {
		ctx = ContextWithRPCTracker(ctx, s.rpcTracker)
	}
	if s.logForwardingClient != nil {
		ctx = ApplyServerInstanceToZerolog(ctx, s.logForwardingClient)
	} else {
		ctx = ApplyClientsCurrentContextToZerolog(ctx)
	}
	return ctx
}

func (s *Instance) StartAndDetach(chans channel.Channel) (*jrpc2.Server, error) {
	multiLogger := &MultiRPCLogger{}

	if s.serverOpts.RPCLog == nil {
		s.serverOpts.RPCLog = multiLogger
	} else {
		multiLogger.AddLogger(s.serverOpts.RPCLog)
		s.serverOpts.RPCLog = multiLogger
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

	s.server = jrpc2.NewServer(s.methods, s.serverOpts)
	s.server = s.server.Start(chans)

	cli := jrpc2.NewClient(chans, nil)

	s.logForwardingClient = &ClientCaller{client: cli}

	return s.server, nil
}

func (s *Instance) StartAndWait(reader io.Reader, writer io.WriteCloser) error {

	server, err := s.StartAndDetach(channel.LSP(reader, writer))
	if err != nil {
		return errors.Errorf("starting external server: %w", err)
	}

	return server.Wait()
}

func (s *Instance) StartAndWaitChannel(chans channel.Channel) error {

	server, err := s.StartAndDetach(chans)
	if err != nil {
		return errors.Errorf("starting external server: %w", err)
	}

	return server.Wait()
}
func NewServerInstance(ctx context.Context, server Server, opts *jrpc2.ServerOptions) *ServerDispatcher {
	methods := buildServerDispatchMap(server)

	instance := NewInstance(ctx, methods, opts)

	return &ServerDispatcher{instance: instance}
}

func NewClientInstance(ctx context.Context, server Client, opts *jrpc2.ServerOptions) *ClientDispatcher {
	methods := buildClientDispatchMap(server)

	instance := NewInstance(ctx, methods, opts)

	return &ClientDispatcher{instance: instance}
}

func NewInstance(ctx context.Context, server handler.Map, opts *jrpc2.ServerOptions) *Instance {
	if opts == nil {
		opts = &jrpc2.ServerOptions{}
	}
	opts.AllowPush = true

	instance := &Instance{methods: server, creationCtx: ctx, serverOpts: opts}

	return instance

}

// Helper functions

// GetRPCTracker returns the RPCTracker instance for testing purposes
func (s *Instance) GetRPCTracker() *RPCTracker {
	return s.rpcTracker
}

// SetRPCTracker sets the RPCTracker instance for testing purposes
func (s *Instance) SetRPCTracker(tracker *RPCTracker) {
	s.rpcTracker = tracker
}
