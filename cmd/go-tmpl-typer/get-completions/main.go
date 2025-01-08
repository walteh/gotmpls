package get_completions

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
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

	// 5. Get completions based on context
	completions := me.getCompletionsAtPosition(info, registry, typeValidator, string(content))

	// Output completions as JSON
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(completions); err != nil {
		return errors.Errorf("failed to encode completions: %w", err)
	}

	return nil
}

func (me *Handler) getCompletionsAtPosition(info *parser.TemplateInfo, registry *ast.TypeRegistry, validator *types.DefaultValidator, content string) []CompletionItem {
	var completions []CompletionItem

	// First, add basic template keywords
	completions = append(completions, me.getTemplateKeywords()...)

	// Get the current line's content
	lines := strings.Split(content, "\n")
	if me.line <= 0 || me.line > len(lines) {
		return completions
	}
	currentLine := lines[me.line-1]
	if me.character <= 0 || me.character > len(currentLine) {
		return completions
	}

	// Check if we're inside a template action
	if isInTemplateAction(currentLine, me.character) {
		// Add template functions
		completions = append(completions, me.getTemplateFunctions(validator)...)

		// If we're after a dot, add field/method completions
		if isDotCompletion(currentLine, me.character) {
			// TODO: Determine the type of the expression before the dot
			// and add its fields/methods as completions
			completions = append(completions, me.getFieldCompletions(registry, info)...)
		} else {
			// Add available variables in scope
			completions = append(completions, me.getVariableCompletions(info)...)
		}
	}

	return completions
}

func (me *Handler) getTemplateKeywords() []CompletionItem {
	return []CompletionItem{
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
		{
			Label:         "template",
			Kind:          "keyword",
			Detail:        "template statement",
			InsertText:    "template \"${1:name}\" ${2:data}}",
			Documentation: "Include a template",
		},
		{
			Label:         "block",
			Kind:          "keyword",
			Detail:        "block statement",
			InsertText:    "block \"${1:name}\"}${2:content}{{end}}",
			Documentation: "Define a template block",
		},
		{
			Label:         "define",
			Kind:          "keyword",
			Detail:        "define statement",
			InsertText:    "define \"${1:name}\"}${2:content}{{end}}",
			Documentation: "Define a template",
		},
	}
}

func (me *Handler) getTemplateFunctions(validator *types.DefaultValidator) []CompletionItem {
	var completions []CompletionItem
	for name, method := range validator.RootMethods {
		completions = append(completions, CompletionItem{
			Label:         name,
			Kind:          "function",
			Detail:        me.formatMethodSignature(method),
			Documentation: me.getMethodDocumentation(method),
		})
	}
	return completions
}

func (me *Handler) formatMethodSignature(method *types.MethodInfo) string {
	params := make([]string, len(method.Parameters))
	for i, param := range method.Parameters {
		params[i] = param.String()
	}

	results := make([]string, len(method.Results))
	for i, result := range method.Results {
		results[i] = result.String()
	}

	return method.Name + "(" + strings.Join(params, ", ") + ") " + strings.Join(results, ", ")
}

func (me *Handler) getMethodDocumentation(method *types.MethodInfo) string {
	// TODO: Add better documentation for each method
	return "Template function " + method.Name
}

func (me *Handler) getFieldCompletions(registry *ast.TypeRegistry, info *parser.TemplateInfo) []CompletionItem {
	var completions []CompletionItem
	// TODO: Implement field completion based on the type registry
	// This will require:
	// 1. Finding the type hint that applies to the current scope
	// 2. Getting the type info for that type
	// 3. Adding all fields and methods as completions
	return completions
}

func (me *Handler) getVariableCompletions(info *parser.TemplateInfo) []CompletionItem {
	var completions []CompletionItem
	// Add all variables that are in scope
	for _, v := range info.Variables {
		completions = append(completions, CompletionItem{
			Label:         v.Name,
			Kind:          "variable",
			Detail:        "Template variable",
			Documentation: "Variable from template scope: " + v.Scope,
		})
	}
	return completions
}

func isInTemplateAction(line string, pos int) bool {
	// Simple check - see if we're between {{ and }}
	// TODO: Make this more robust by considering nested actions
	lastOpen := strings.LastIndex(line[:pos], "{{")
	if lastOpen == -1 {
		return false
	}
	nextClose := strings.Index(line[lastOpen:], "}}")
	return nextClose == -1 || pos <= lastOpen+nextClose
}

func isDotCompletion(line string, pos int) bool {
	if pos <= 0 {
		return false
	}
	return strings.TrimSpace(line[pos-1:pos]) == "."
}
