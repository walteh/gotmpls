package get_diagnostics

// `hclFmt` command recursively looks for hcl files in the directory tree starting at workingDir, and formats them
// based on the language style guides provided by Hashicorp. This is done using the official hcl2 library.

import (
	"context"

	"github.com/spf13/cobra"
)

type Handler struct {
	packageDir             string
	templateFileExtensions []string
	format                 string // vscode, json, yaml, text
}

func NewGetDiagnosticsCommand() *cobra.Command {
	me := &Handler{}

	cmd := &cobra.Command{
		Use:   "get-diagnostics [package-dir]",
		Short: "get diagnostics from a go template file",
	}

	cmd.Flags().StringSliceVar(&me.templateFileExtensions, "template-file-extensions", []string{".tmpl", ".tmpl.go"}, "the extensions of the template files to get diagnostics from")
	cmd.Flags().StringVar(&me.format, "format", "vscode", "the format of the diagnostics")
	// the glob will will be argument one
	cmd.Args = cobra.ExactArgs(1)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		me.packageDir = args[0]
		return me.Run(cmd.Context())
	}

	return cmd
}

func (me *Handler) Run(ctx context.Context) error {

	// 1. parse the ast of the package dir

	// 2. get any template files in the package dir

	// 3. parse the template files, retaining the locations of the variables and functions used

	// 4. get the types from the template files - should be in a comment that looks like {{- /*gotype: github.com/walteh/minute-api/proto/cmd/protoc-gen-cdk/generator.BuilderConfig */ -}}

	// 5. generate the diagnostics (POC 1: don't validate the types)
	// - POC 2: validate the types

	return nil
}
