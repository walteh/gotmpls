//go:build wasip1

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	lsp "github.com/walteh/gotmpls/cmd/gotmpls-wasi/lsp"
)

func main() {
	// Create a logger that writes to stderr
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("service", "gotmpls-wasi").Logger()
	ctx := logger.WithContext(context.Background())

	// Log our initialization message to stderr
	fmt.Fprintf(os.Stderr, "DONT MOVE FORWARD UNTIL THIS LOG SHOWS UP SOMEWHERE\n")

	// Initialize the LSP server
	if err := lsp.ServeLSP(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error serving LSP: %v\n", err)
		os.Exit(1)
	}
}

// // / just a simple thing that reads from stdin and writes to stderr what it saw
// func main() {
// 	fmt.Fprintf(os.Stderr, "we out here 😎\n")
// 	// Create a buffer to store input
// 	b := make([]byte, 1024)

// 	// Read from stdin in a loop
// 	for {
// 		n, err := os.Stdin.Read(b)
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "❌ Error reading from stdin: %v\n", err)
// 			os.Exit(1)
// 		}

// 		// Write what we read to stderr
// 		fmt.Fprintf(os.Stderr, "📥 Read from stdin: %s", b[:n])
// 	}

// }
