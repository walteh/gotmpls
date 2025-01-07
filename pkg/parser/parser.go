package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/k0kubun/pp/v3"
	"gitlab.com/tozd/go/errors"
)

// DefaultTemplateParser is the default implementation of TemplateParser
type DefaultTemplateParser struct{}

// NewDefaultTemplateParser creates a new DefaultTemplateParser
func NewDefaultTemplateParser() *DefaultTemplateParser {
	return &DefaultTemplateParser{}
}

var typeHintRegex = regexp.MustCompile(`{{-?\s*/\*gotype:\s*([^*]+)\s*\*/\s*-?}}`)

// GetLineAndColumn calculates the line and column number for a given position in the text
// pos is 0-based, but we return 1-based column numbers as per editor/IDE conventions
func GetLineAndColumn(text string, pos parse.Pos) (line, col int) {
	if pos == 0 {
		return 1, 1
	}

	// pos is already 0-based, so we don't need to subtract 1
	offset := int(pos)

	// Count newlines before the position to get the line number
	line = 1
	lastNewline := -1
	for i := 0; i < offset && i < len(text); i++ {
		if text[i] == '\n' {
			line++
			lastNewline = i
		}
	}

	// Column is the number of characters after the last newline
	// Add 1 to convert from 0-based offset to 1-based column number
	col = offset - lastNewline

	// If we're in a pipe expression (after a |), we need to adjust the column
	// by looking backwards from our position to find the pipe
	if offset > 0 && offset < len(text) {
		for i := offset - 1; i >= 0; i-- {
			if text[i] == '|' {
				// We found a pipe before our position, adjust the column
				col--
				break
			}
			if text[i] == '{' || text[i] == '}' || text[i] == '\n' {
				// Stop looking if we hit template boundaries or newlines
				break
			}
		}
	}

	return line, col
}

// Parse implements TemplateParser
func (p *DefaultTemplateParser) Parse(ctx context.Context, content []byte, filename string) (*TemplateInfo, error) {
	contentStr := string(content)

	// First, extract type hints
	matches := typeHintRegex.FindSubmatch(content)
	if len(matches) < 2 {
		return nil, errors.Errorf("no type hint found in template %s", filename)
	}

	typePath := strings.TrimSpace(string(matches[1]))
	line, col := 1, 12 // We'll fix the exact positions later

	// Create a template with all necessary functions to avoid parsing errors
	tmpl := template.New(filename).Funcs(template.FuncMap{
		"printf": func(format string, args ...interface{}) string { return "" },
		"upper":  strings.ToUpper,
	})

	// Parse the template
	parsedTmpl, err := tmpl.Parse(contentStr)
	if err != nil {
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	info := &TemplateInfo{
		Filename: filename,
		TypeHints: []TypeHint{
			{
				TypePath: typePath,
				Line:     line,
				Column:   col,
			},
		},
		Variables: make([]VariableLocation, 0),
		Functions: make([]FunctionLocation, 0),
	}

	// pp.Println("Parsed template", parsedTmpl)

	// Keep track of seen functions to avoid duplicates
	seenFunctions := make(map[string]bool)

	// Walk the AST and collect variables and functions
	var walk func(node parse.Node) error
	walk = func(node parse.Node) error {
		// pp.Println("Node", node)

		if node == nil {
			return nil
		}

		fmt.Printf("Node type: %T\n", node)
		switch n := node.(type) {
		case *parse.ActionNode:
			fmt.Printf("Action node: %s\n", n.String())
			if err := walk(n.Pipe); err != nil {
				return err
			}
		case *parse.ListNode:
			if n != nil {
				for _, node := range n.Nodes {
					if err := walk(node); err != nil {
						return err
					}
				}
			}
		case *parse.TemplateNode:
			fmt.Printf("Template node: %s\n", n.Name)
			// Handle template definitions
			if err := walk(n.Pipe); err != nil {
				return err
			}
		case *parse.WithNode:
			fmt.Printf("With node: %s\n", n.String())
			if err := walk(n.Pipe); err != nil {
				return err
			}
			if err := walk(n.List); err != nil {
				return err
			}
		case *parse.IfNode:
			fmt.Printf("If node: %s\n", n.String())
			if err := walk(n.Pipe); err != nil {
				return err
			}
			if err := walk(n.List); err != nil {
				return err
			}
			if err := walk(n.ElseList); err != nil {
				return err
			}
		case *parse.RangeNode:
			fmt.Printf("Range node: %s\n", n.String())
			if err := walk(n.Pipe); err != nil {
				return err
			}
			if err := walk(n.List); err != nil {
				return err
			}
			if err := walk(n.ElseList); err != nil {
				return err
			}
		case *parse.PipeNode:
			// Process pipe commands
			for i, cmd := range n.Cmds {
				fmt.Printf("Command: %s\n", cmd.String())
				// First argument might be a function
				if len(cmd.Args) > 0 {
					fmt.Printf("First arg type: %T\n", cmd.Args[0])
					if ident, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
						fmt.Printf("Found function: %s\n", ident.Ident)
						if !seenFunctions[ident.Ident] {
							startLine, startCol := GetLineAndColumn(contentStr, ident.Pos)
							endLine, endCol := GetLineAndColumn(contentStr, ident.Pos+parse.Pos(len(ident.Ident)))
							info.Functions = append(info.Functions, FunctionLocation{
								Name:    ident.Ident,
								Line:    startLine,
								Column:  startCol,
								EndLine: endLine,
								EndCol:  endCol,
							})
							seenFunctions[ident.Ident] = true
						}
					}
				}

				// Look for field nodes (variables)
				for _, arg := range cmd.Args {
					fmt.Printf("Arg type: %T\n", arg)
					if field, ok := arg.(*parse.FieldNode); ok {
						fmt.Printf("Found variable: %s\n", field.Ident[0])
						startLine, startCol := GetLineAndColumn(contentStr, field.Pos)
						endLine, endCol := GetLineAndColumn(contentStr, field.Pos+parse.Pos(len(field.Ident[0])))
						info.Variables = append(info.Variables, VariableLocation{
							Name:    field.Ident[0],
							Line:    startLine,
							Column:  startCol,
							EndLine: endLine,
							EndCol:  endCol,
						})
					}
				}

				// If this is not the last command in the pipe, check if the next command is a function
				if i < len(n.Cmds)-1 {
					nextCmd := n.Cmds[i+1]
					if len(nextCmd.Args) > 0 {
						if ident, ok := nextCmd.Args[0].(*parse.IdentifierNode); ok {
							fmt.Printf("Found piped function: %s\n", ident.Ident)
							if !seenFunctions[ident.Ident] {
								startLine, startCol := GetLineAndColumn(contentStr, ident.Pos)
								endLine, endCol := GetLineAndColumn(contentStr, ident.Pos+parse.Pos(len(ident.Ident)))
								info.Functions = append(info.Functions, FunctionLocation{
									Name:    ident.Ident,
									Line:    startLine,
									Column:  startCol,
									EndLine: endLine,
									EndCol:  endCol,
								})
								seenFunctions[ident.Ident] = true
							}
						}
					}
				}
			}
		}
		return nil
	}

	// Walk all templates in the tree
	if err := walk(parsedTmpl.Tree.Root); err != nil {
		return nil, errors.Errorf("failed to walk template AST: %w", err)
	}

	// Walk through all templates in the common.tmpl map
	for _, t := range parsedTmpl.Templates() {
		if t.Tree != nil && t.Name() != parsedTmpl.Name() {
			pp.Println("Walking template", t.Name())
			if err := walk(t.Tree.Root); err != nil {
				return nil, errors.Errorf("failed to walk template %s: %w", t.Name(), err)
			}
		}
	}

	return info, nil
}

// TemplateParser is responsible for parsing Go template files and extracting type information
type TemplateParser interface {
	// Parse parses a template file and returns the locations of variables and functions used
	Parse(ctx context.Context, content []byte, filename string) (*TemplateInfo, error)
}

// TemplateInfo contains information about a parsed template
type TemplateInfo struct {
	Variables []VariableLocation
	Functions []FunctionLocation
	TypeHints []TypeHint
	Filename  string
}

// VariableLocation represents a variable usage in a template
type VariableLocation struct {
	Name    string
	Line    int
	Column  int
	EndLine int
	EndCol  int
}

// FunctionLocation represents a function call in a template
type FunctionLocation struct {
	Name    string
	Line    int
	Column  int
	EndLine int
	EndCol  int
}

// TypeHint represents a type hint comment in the template
type TypeHint struct {
	TypePath string // e.g. "github.com/walteh/minute-api/proto/cmd/protoc-gen-cdk/generator.BuilderConfig"
	Line     int
	Column   int
}
