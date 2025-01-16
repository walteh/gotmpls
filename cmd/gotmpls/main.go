package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	serve_lsp "github.com/walteh/gotmpls/cmd/gotmpls/serve-lsp"
	"gitlab.com/tozd/go/errors"
)

func main() {
	if err := run(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "gotmpls",
		Short: "A tool for type checking go templates",
	}

	rootCmd.AddCommand(serve_lsp.NewServeLSPCommand())

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		return errors.Errorf("failed to execute command: %w", err)
	}

	return nil
}
