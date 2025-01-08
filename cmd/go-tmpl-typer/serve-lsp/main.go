package serve_lsp

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

type Handler struct {
	debug bool
}

func NewServeLSPCommand() *cobra.Command {
	me := &Handler{}

	cmd := &cobra.Command{
		Use:   "serve-lsp",
		Short: "start the language server",
	}

	cmd.Flags().BoolVar(&me.debug, "debug", false, "enable debug logging")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return me.Run(cmd.Context())
	}

	return cmd
}

func (me *Handler) Run(ctx context.Context) error {
	// Create a new LSP server with all the components it needs
	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		me.debug,
	)

	if me.debug {
		fmt.Fprintf(os.Stderr, "starting language server with debug logging enabled\n")
	}

	// Start the server using stdin/stdout
	if err := server.Start(ctx, os.Stdin, os.Stdout); err != nil {
		return errors.Errorf("failed to start language server: %w", err)
	}

	return nil
}
