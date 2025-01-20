package protocol

import (
	"context"
	"os"
	"os/exec"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

func NewGoplsServerInstance(ctx context.Context) (*ServerDispatcher, error) {

	ctx = zerolog.New(os.Stderr).With().Str("name", "gopls").Logger().WithContext(ctx)

	cmd := exec.CommandContext(ctx, "gopls", "-v", "-vv", "-rpc.trace")
	copts := &jrpc2.ClientOptions{
		Logger: func(msg string) {
			zerolog.Ctx(ctx).Info().Msgf("gopls [client]: %s", msg)
		},
	}
	sopts := &jrpc2.ServerOptions{
		Logger: func(msg string) {
			zerolog.Ctx(ctx).Info().Msgf("gopls [server]: %s", msg)
		},
	}

	inst, err := NewCmdServerInstance(ctx, cmd, copts, sopts)
	if err != nil {
		return nil, errors.Errorf("creating server instance: %w", err)
	}

	return inst, nil
}

func NewCmdServerInstance(ctx context.Context, cmd *exec.Cmd, copts *jrpc2.ClientOptions, sopts *jrpc2.ServerOptions) (*ServerDispatcher, error) {
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

		n func(r *jrpc2.Request)
	}

	handlerf := &handlerft{
		f: func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
			return nil, nil
		},
		n: func(r *jrpc2.Request) {
		},
	}

	copts.OnCallback = handlerf.f
	copts.OnNotify = handlerf.n

	client := jrpc2.NewClient(channel.LSP(out, in), copts)
	cbs := &ServerCaller{client: client}
	instance := NewServerInstance(ctx, cbs, sopts)

	handlerf.f = func(ctx context.Context, r *jrpc2.Request) (interface{}, error) {
		zerolog.Ctx(ctx).Info().Msgf("gopls [hndld client callback]: %s", r.Method())
		var params interface{}
		if err := r.UnmarshalParams(&params); err != nil {
			return nil, err
		}
		return instance.Instance().server.Callback(ctx, r.Method(), params)
	}

	handlerf.n = func(r *jrpc2.Request) {
		if r == nil {
			return
		}
		zerolog.Ctx(ctx).Info().Msgf("gopls [hndld client notify]: %s", r.Method())
		var params interface{}
		if err := r.UnmarshalParams(&params); err != nil {
			return
		}
		instance.Instance().server.Notify(ctx, r.Method(), params)
	}

	instance.Instance().backgroundCmd = cmd

	return instance, nil
}
