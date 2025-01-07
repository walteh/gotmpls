package parser

import (
	"context"
	"go/types"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

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
	line, col := 1, 12 // Type hint is always on the first line, column is fixed at 12 (after "/*gotype: ")

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
		Functions: make([]VariableLocation, 0),
	}

	// Keep track of seen functions to avoid duplicates
	seenFunctions := make(map[string]bool)

	// Walk the AST and collect variables and functions
	var walk func(node parse.Node) error
	walk = func(node parse.Node) error {
		if node == nil {
			return nil
		}

		switch n := node.(type) {
		case *parse.ActionNode:
			if err := walk(n.Pipe); err != nil {
				return err
			}
		case *parse.IfNode:
			// Handle if condition
			if err := walk(n.Pipe); err != nil {
				return err
			}
			// Handle the body of the if statement
			if err := walk(n.List); err != nil {
				return err
			}
			// Handle the else clause if it exists
			if n.ElseList != nil {
				if err := walk(n.ElseList); err != nil {
					return err
				}
			}
		case *parse.ListNode:
			if n != nil {
				for _, node := range n.Nodes {
					if err := walk(node); err != nil {
						return err
					}
				}
			}
		case *parse.PipeNode:
			if n != nil {
				if len(n.Cmds) > 0 {
					// the result of the first command is the argument to the next command
				}
				args := make([]types.Type, 0)
				for _, cmd := range n.Cmds {
					for i, arg := range cmd.Args {
						if i == 0 {
							continue
						}
						switch v := arg.(type) {
						case *parse.FieldNode:
							// Variable reference
							line, col := GetLineAndColumn(contentStr, v.Position())
							endLine, endCol := GetLineAndColumn(contentStr, v.Position()+parse.Pos(len(v.String())-1))
							fullName := ""
							// Add each part of the field path as a separate variable
							for _, ident := range v.Ident {
								fullName += ident + "."
							}
							fullName = strings.TrimSuffix(fullName, ".")

							item := VariableLocation{
								Name:    fullName,
								Line:    line,
								Column:  col,
								EndLine: endLine,
								EndCol:  endCol,
							}
							// For nested fields, we want to include the full path up to this point
							// e.g., for .Address.Street, we want both "Address" and "Street"
							info.Variables = append(info.Variables, item)

							args = append(args, &item)

						case *parse.IdentifierNode:
							// if !seenFunctions[v.String()] {
							line, col := GetLineAndColumn(contentStr, v.Position())
							// For function calls, we want to include the entire function name
							endLine, endCol := GetLineAndColumn(contentStr, v.Position()+parse.Pos(len(v.String())))
							item := VariableLocation{
								Name:            v.Ident,
								Line:            line,
								Column:          col,
								EndLine:         endLine,
								EndCol:          endCol,
								MethodArguments: []types.Type{},
							}
							info.Functions = append(info.Functions, item)
							seenFunctions[v.String()] = true

							args = append(args, &item)
						}

						// }
					}

					// now process the
					root := n.Cmds[0]
					switch v := root.Args[0].(type) {
					case *parse.FieldNode:
						line, col := GetLineAndColumn(contentStr, v.Position())
						endLine, endCol := GetLineAndColumn(contentStr, v.Position()+parse.Pos(len(v.String())-1))
						fullName := ""
						// Add each part of the field path as a separate variable
						for _, ident := range v.Ident {
							fullName += ident + "."
						}
						fullName = strings.TrimSuffix(fullName, ".")

						item := VariableLocation{
							Name:            fullName,
							Line:            line,
							Column:          col,
							EndLine:         endLine,
							EndCol:          endCol,
							MethodArguments: args,
						}
						info.Variables = append(info.Functions, item)
					case *parse.IdentifierNode:
						line, col := GetLineAndColumn(contentStr, v.Position())
						endLine, endCol := GetLineAndColumn(contentStr, v.Position()+parse.Pos(len(v.String())-1))
						item := VariableLocation{
							Name:            v.Ident,
							Line:            line,
							Column:          col,
							EndLine:         endLine,
							EndCol:          endCol,
							MethodArguments: args,
						}
						info.Functions = append(info.Functions, item)
					}
				}
			}
		}
		return nil
	}

	// Walk through all templates in the common.tmpl map
	for _, t := range parsedTmpl.Templates() {
		if t.Tree != nil {
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
	Functions []VariableLocation
	TypeHints []TypeHint
	Filename  string
}

var _ types.Type = &VariableLocation{}

// VariableLocation represents a variable usage in a template
type VariableLocation struct {
	Name    string
	Line    int
	Column  int
	EndLine int
	EndCol  int
	// Pipe               bool
	// MethodArgumentsRef *VariableLocation // take the result of this named type as the argument
	MethodArguments []types.Type
}

// String implements types.Type.
func (v *VariableLocation) String() string {
	return v.Name
}

// Underlying implements types.Type.
func (v *VariableLocation) Underlying() types.Type {
	return nil
}

// type ArgumentRef struct {
// 	Variable *VariableLocation
// 	Function *FunctionLocation
// }

// // FunctionLocation represents the location of a function call in the template
// type FunctionLocation struct {
// 	Name         string
// 	Line         int
// 	Column       int
// 	EndLine      int
// 	EndCol       int
// 	ArgumentsRef string // take the result of this named type as the argument
// 	Arguments    []types.Type
// }

// TypeHint represents a type hint comment in the template
type TypeHint struct {
	TypePath string // e.g. "github.com/walteh/minute-api/proto/cmd/protoc-gen-cdk/generator.BuilderConfig"
	Line     int
	Column   int
}
