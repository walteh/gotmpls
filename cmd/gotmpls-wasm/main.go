//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	lspcmd "github.com/walteh/gotmpls/cmd/gotmpls-wasm/lsp"
)

// note about logging: the logging will go to the extension host process, so if we want to actually return logs to the extension,
// we need to do some extra work
// - like intercepting the logs and returning them to the extension in the response

func main() {
	ctx := context.Background()

	// Initialize the gotmpls object
	gotmpls := map[string]interface{}{
		"serve_lsp": wrapResult(ctx, lspcmd.ServeLSP),
	}

	// Set the gotmpls object first
	js.Global().Set("gotmpls_wasm", js.ValueOf(gotmpls))

	// Log initialization
	fmt.Println("[gotmpls-golang-wasm] initialized")

	// Set ready flag to indicate initialization is complete
	js.Global().Set("gotmpls_initialized", js.ValueOf(true))

	// Create a channel that never receives anything
	done := make(chan struct{})
	<-done // This will block forever, keeping the WASM module alive
}

func wrapResult[T any](ctx context.Context, fn func(ctx context.Context, this js.Value, args []js.Value) (T, error)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		result, err := fn(ctx, this, args)
		if err != nil {
			// Log errors to console for debugging
			js.Global().Get("console").Call("error", "[gotmpls-golang-wasm]", err.Error())
			return map[string]any{
				"result": nil,
				"error":  err.Error(),
			}
		}
		return map[string]any{
			"result": result,
			"error":  nil,
		}
	})
}
