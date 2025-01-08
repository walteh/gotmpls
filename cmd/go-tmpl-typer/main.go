package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	get_completions "github.com/walteh/go-tmpl-typer/cmd/go-tmpl-typer/get-completions"
	get_diagnostics "github.com/walteh/go-tmpl-typer/cmd/go-tmpl-typer/get-diagnostics"
	serve_lsp "github.com/walteh/go-tmpl-typer/cmd/go-tmpl-typer/serve-lsp"
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
		Use:   "go-tmpl-typer",
		Short: "A tool for type checking go templates",
	}

	rootCmd.AddCommand(get_completions.NewGetCompletionsCommand())
	rootCmd.AddCommand(get_diagnostics.NewGetDiagnosticsCommand())
	rootCmd.AddCommand(serve_lsp.NewServeLSPCommand())

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		return errors.Errorf("failed to execute command: %w", err)
	}

	return nil
}
