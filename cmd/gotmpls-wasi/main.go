//go:build wasip1

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	lspcmd "github.com/walteh/gotmpls/cmd/gotmpls-wasi/lsp"
)

func main() {
	// Create a logger that writes to stderr
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("service", "gotmpls-wasi").Logger()
	ctx := logger.WithContext(context.Background())

	// Log our initialization message to stderr
	fmt.Fprintf(os.Stderr, "DONT MOVE FORWARD UNTIL THIS LOG SHOWS UP SOMEWHERE\n")

	// Initialize the LSP server
	if err := lspcmd.ServeLSP(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error serving LSP: %v\n", err)
		os.Exit(1)
	}
}
