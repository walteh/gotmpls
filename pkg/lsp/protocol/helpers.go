package protocol

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

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

// type Callbacker interface {
// 	Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error)
// 	Notify(ctx context.Context, method string, params interface{}) error
// }

func createServerCallBack[I any, O any](ctx context.Context, client *jrpc2.Server, method string, params *I, result *O) error {
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

func createServerEmptyResultCallBack[I any](ctx context.Context, client *jrpc2.Server, method string, params *I) error {
	_, err := client.Callback(ctx, method, params)
	return err
}

func createServerEmptyCallBack(ctx context.Context, client *jrpc2.Server, method string) error {
	_, err := client.Callback(ctx, method, nil)
	return err
}

func createServerEmptyParamsCallBack[O any](ctx context.Context, client *jrpc2.Server, method string, result *O) error {
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

func createServerNotifyBack[I any](ctx context.Context, client *jrpc2.Server, method string, params *I) error {
	err := client.Notify(ctx, method, params)
	return err
}

func createServerEmptyNotifyBack(ctx context.Context, client *jrpc2.Server, method string) error {
	err := client.Notify(ctx, method, nil)
	return err
}

func createClientCall[I any, O any](ctx context.Context, client *jrpc2.Client, method string, params *I, result *O) error {
	res, err := client.Call(ctx, method, params)
	if err != nil {
		return err
	}

	if result != nil {
		err = res.UnmarshalResult(result)
		return err
	}

	return nil
}

func createClientEmptyResultCall[I any](ctx context.Context, client *jrpc2.Client, method string, params *I) error {
	_, err := client.Call(ctx, method, params)
	return err
}

func createClientEmptyCall(ctx context.Context, client *jrpc2.Client, method string) error {
	_, err := client.Call(ctx, method, nil)
	return err
}

func createClientEmptyParamsCall[O any](ctx context.Context, client *jrpc2.Client, method string, result *O) error {
	res, err := client.Call(ctx, method, nil)
	if err != nil {
		return err
	}

	if result != nil {
		err = res.UnmarshalResult(result)
		return err
	}

	return nil
}

func createClientNotify[I any](ctx context.Context, client *jrpc2.Client, method string, params *I) error {
	err := client.Notify(ctx, method, params)
	return err
}

func createClientEmptyNotify(ctx context.Context, client *jrpc2.Client, method string) error {
	err := client.Notify(ctx, method, nil)
	return err
}
