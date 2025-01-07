package get_diagnostics

import (
	"context"
	"os"
	"path/filepath"

	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cobra"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

type Handler struct {
	packageDir             string
	templateFileExtensions []string
	format                 string // vscode, json, yaml, text
	showHints              bool
}

func NewGetDiagnosticsCommand() *cobra.Command {
	me := &Handler{}

	cmd := &cobra.Command{
		Use:   "get-diagnostics [package-dir]",
		Short: "get diagnostics from a go template file",
	}

	cmd.Flags().StringSliceVar(&me.templateFileExtensions, "template-file-extensions", []string{".tmpl", ".tmpl.go"}, "the extensions of the template files to get diagnostics from")
	cmd.Flags().StringVar(&me.format, "format", "vscode", "the format of the diagnostics")
	cmd.Flags().BoolVar(&me.showHints, "show-hints", false, "show hints")
	// the glob will will be argument one
	cmd.Args = cobra.ExactArgs(1)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		me.packageDir = args[0]
		return me.Run(cmd.Context())
	}

	return cmd
}

func (me *Handler) Run(ctx context.Context) error {
	// 1. Create a new template parser, type validator, and package analyzer
	templateParser := parser.NewDefaultTemplateParser()
	typeValidator := types.NewDefaultValidator()
	packageAnalyzer := ast.NewDefaultPackageAnalyzer()

	// 2. Analyze the package to get type information
	registry, err := packageAnalyzer.AnalyzePackage(ctx, me.packageDir)
	if err != nil {
		return errors.Errorf("failed to analyze package: %w", err)
	}

	// 3. Find all template files in the package directory
	var templateFiles []string
	err = filepath.Walk(me.packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			for _, ext := range me.templateFileExtensions {
				if filepath.Ext(path) == ext {
					templateFiles = append(templateFiles, path)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return errors.Errorf("failed to walk package directory: %w", err)
	}

	// 4. Parse each template file and collect diagnostics
	var allDiagnostics []*diagnostic.Diagnostics
	for _, templateFile := range templateFiles {
		// Read the template file
		content, err := os.ReadFile(templateFile)
		if err != nil {
			return errors.Errorf("failed to read template file %s: %w", templateFile, err)
		}

		// Parse the template
		info, err := templateParser.Parse(ctx, content, templateFile)
		if err != nil {
			// Add a diagnostic for the parse error
			allDiagnostics = append(allDiagnostics, &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{
					{
						Message:  err.Error(),
						Line:     1,
						Column:   1,
						EndLine:  1,
						EndCol:   1,
						Severity: diagnostic.Error,
					},
				},
			})
			continue
		}

		// Create a diagnostic generator
		generator := diagnostic.NewDefaultGenerator()

		pp.Println(info)

		// Generate diagnostics for this template, using the registry for type validation
		diagnostics, err := generator.Generate(ctx, info, typeValidator, registry)
		if err != nil {
			return errors.Errorf("failed to generate diagnostics for %s: %w", templateFile, err)
		}

		allDiagnostics = append(allDiagnostics, diagnostics)
	}

	// 5. Format and output the diagnostics
	switch me.format {
	case "vscode":
		formatter := diagnostic.NewVSCodeFormatter()
		for _, d := range allDiagnostics {
			if !me.showHints {
				d.Hints = []diagnostic.Diagnostic{}
			}
			output, err := formatter.Format(d)
			if err != nil {
				return errors.Errorf("failed to format diagnostics: %w", err)
			}
			if output != nil {
				os.Stdout.Write(output)
			}
		}
	default:
		// For other formats, just print the diagnostics
		for _, d := range allDiagnostics {
			for _, err := range d.Errors {
				println(err.Message)
			}
			for _, warn := range d.Warnings {
				println(warn.Message)
			}
		}
	}

	return nil
}
