package get_completions

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"gitlab.com/tozd/go/errors"
)

type CompletionItem struct {
	Label         string    `json:"label"`
	Kind          string    `json:"kind"` // "function", "variable", "field", etc.
	Detail        string    `json:"detail,omitempty"`
	Documentation string    `json:"documentation,omitempty"`
	SortText      string    `json:"sortText,omitempty"`
	FilterText    string    `json:"filterText,omitempty"`
	InsertText    string    `json:"insertText,omitempty"`
	TextEdit      *TextEdit `json:"textEdit,omitempty"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

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

	// 2. Read the template file
	content, err := os.ReadFile(me.filePath)
	if err != nil {
		return errors.Errorf("failed to read template file: %w", err)
	}

	// 4. Parse the template
	_, err = templateParser.Parse(ctx, content, me.filePath)
	if err != nil {
		return errors.Errorf("failed to parse template: %w", err)
	}

	// TODO: Implement actual completion logic here
	// For now, return some placeholder completions
	completions := []CompletionItem{
		{
			Label:         "if",
			Kind:          "keyword",
			Detail:        "if condition",
			InsertText:    "if ${1:condition}}",
			Documentation: "Basic if statement",
		},
		{
			Label:         "range",
			Kind:          "keyword",
			Detail:        "range statement",
			InsertText:    "range ${1:collection}}",
			Documentation: "Range over a collection",
		},
		{
			Label:         "with",
			Kind:          "keyword",
			Detail:        "with statement",
			InsertText:    "with ${1:value}}",
			Documentation: "With statement for scoped variables",
		},
	}

	// Output completions as JSON
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(completions); err != nil {
		return errors.Errorf("failed to encode completions: %w", err)
	}

	return nil
}
