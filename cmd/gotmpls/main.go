package main

import (
	"context"
	"os"
	"runtime/debug"

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

	info, ok := debug.ReadBuildInfo()
	if !ok {
		rootCmd.Version = "unknown"
	} else {
		rootCmd.Version = info.Main.Version
	}

	cmdVersion := &cobra.Command{
		Use: "raw-version",
		Run: func(cmdz *cobra.Command, args []string) {
			cmdz.Println(rootCmd.Version)
		},
		Hidden: true,
	}

	rootCmd.AddCommand(cmdVersion)

	rootCmd.AddCommand(serve_lsp.NewServeLSPCommand())

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		return errors.Errorf("failed to execute command: %w", err)
	}

	return nil
}
