package serve_lsp

import (
	"context"
	"fmt"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/walteh/gotmpls/pkg/lsp"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

type Handler struct {
	debug bool
	trace bool
}

func NewServeLSPCommand() *cobra.Command {
	me := &Handler{}

	cmd := &cobra.Command{
		Use:   "serve-lsp",
		Short: "start the language server",
	}

	cmd.Flags().BoolVar(&me.debug, "debug", false, "enable debug logging")
	cmd.Flags().BoolVar(&me.trace, "trace", false, "enable trace logging")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return me.Run(cmd.Context())
	}

	return cmd
}

type RPCLogger struct {
}

func (me *RPCLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	zerolog.Ctx(ctx).Info().Str("rpc_params", req.ParamString()).Str("rpc_id", req.ID()).Str("rpc_method", req.Method()).Msg("client request")
}

func (me *RPCLogger) LogResponse(ctx context.Context, res *jrpc2.Response) {
	zerolog.Ctx(ctx).Info().Str("rpc_params", res.ResultString()).Str("rpc_id", res.ID()).Msg("server response")
}

func (me *Handler) Run(ctx context.Context) error {
	// Create a new LSP server with all the components it needs
	server := lsp.NewServer(ctx)

	if me.trace {
		ctx = zerolog.New(os.Stderr).With().Str("name", "gotmpls").Logger().Level(zerolog.TraceLevel).WithContext(ctx)
	} else if me.debug {
		ctx = zerolog.New(os.Stderr).With().Str("name", "gotmpls").Logger().Level(zerolog.DebugLevel).WithContext(ctx)
	} else {
		ctx = zerolog.New(os.Stderr).With().Str("name", "gotmpls").Logger().Level(zerolog.InfoLevel).WithContext(ctx)
	}

	opts := &jrpc2.ServerOptions{
		RPCLog: &RPCLogger{},
		Logger: func(msg string) {
			zerolog.Ctx(ctx).Info().Str("name", "gotmpls").Msg(msg)
		},
	}

	instance := protocol.NewServerInstance(ctx, server, opts)

	srv, err := instance.Instance().StartAndDetach(channel.LSP(os.Stdin, os.Stdout))
	if err != nil {
		return errors.Errorf("error running language server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[lsp] server started\n")

	server.SetCallbackClient(instance.Instance().ForwardingClient())

	if err := srv.Wait(); err != nil {
		return errors.Errorf("error running language server: %w", err)
	}

	return nil
}
