package protocol

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

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

func NewInMemoryClientInstance(ctx context.Context, copts *jrpc2.ClientOptions, sopts *jrpc2.ServerOptions) (*ServerInstance, io.Reader, io.WriteCloser, error) {
	pr, pw := io.Pipe()

	type handlerft struct {
		f func(ctx context.Context, r *jrpc2.Request) (interface{}, error)
	}

	sopts.AllowPush = true

	handlerf := &handlerft{
		f: func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
			return nil, nil
		},
	}
	copts.OnCallback = handlerf.f

	client := jrpc2.NewClient(channel.LSP(pr, pw), copts)
	cbs := NewCallbackServer(client)
	instance := NewServerInstance(ctx, cbs, sopts)

	handlerf.f = func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		var params interface{}
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, err
		}
		return instance.server.Callback(ctx, r.Method(), params)
	}

	return instance, pr, pw, nil
}
