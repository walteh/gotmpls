package get_completions

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/completion"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

type Handler struct {
	packageDir string
	line       int
	character  int
	filePath   string
}

func NewGetCompletionsCommand() *cobra.Command {
	me := &Handler{}

	cmd := &cobra.Command{
		Use:   "get-completions [package-dir] [file-path] [line] [character]",
		Short: "get completions for a position in a template file",
	}

	cmd.Args = cobra.ExactArgs(4)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		me.packageDir = args[0]
		me.filePath = args[1]
		// Parse line and character from args
		var err error
		me.line, err = strconv.Atoi(args[2])
		if err != nil {
			return errors.Errorf("invalid line number: %w", err)
		}
		me.character, err = strconv.Atoi(args[3])
		if err != nil {
			return errors.Errorf("invalid character number: %w", err)
		}
		return me.Run(cmd.Context())
	}

	return cmd
}

func (me *Handler) Run(ctx context.Context) error {
	// 1. Create necessary components
	templateParser := parser.NewDefaultTemplateParser()
	typeValidator := types.NewDefaultValidator()
	packageAnalyzer := ast.NewDefaultPackageAnalyzer()

	// 2. Analyze the package to get type information
	registry, err := packageAnalyzer.AnalyzePackage(ctx, me.packageDir)
	if err != nil {
		return errors.Errorf("failed to analyze package: %w", err)
	}

	// 3. Read the template file
	content, err := os.ReadFile(me.filePath)
	if err != nil {
		return errors.Errorf("failed to read template file: %w", err)
	}

	// 4. Parse the template
	info, err := templateParser.Parse(ctx, content, me.filePath)
	if err != nil {
		return errors.Errorf("failed to parse template: %w", err)
	}

	// 5. Create completion provider and get completions
	provider := completion.NewProvider(typeValidator, registry)
	completions, err := provider.GetCompletions(ctx, info, me.line, me.character, string(content))
	if err != nil {
		return errors.Errorf("failed to get completions: %w", err)
	}

	// Output completions as JSON
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(completions); err != nil {
		return errors.Errorf("failed to encode completions: %w", err)
	}

	return nil
}
