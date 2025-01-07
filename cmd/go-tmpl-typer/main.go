package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
	getdiagnosticscmd "github.com/walteh/go-tmpl-typer/cmd/go-tmpl-typer/get-diagnostics"
)

func main() {
	ctx := context.Background()

	cmd := &cobra.Command{
		Use: "go-tmpl-typer",
	}

	cmd.AddCommand(getdiagnosticscmd.NewGetDiagnosticsCommand())
	info, ok := debug.ReadBuildInfo()
	if !ok {
		cmd.Version = "unknown"
	} else {
		cmd.Version = info.Main.Version
	}

	cmd.InitDefaultVersionFlag()

	// cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
