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

	info := &TemplateInfo{
		Filename:  filename,
		Variables: make([]VariableLocation, 0),
		Functions: make([]VariableLocation, 0),
		TypeHints: make([]TypeHint, 0),
	}

	// Helper function to extract type hints from a template text
	extractTypeHints := func(text string, scope string) []TypeHint {
		var hints []TypeHint
		matches := typeHintRegex.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				typePath := strings.TrimSpace(text[match[2]:match[3]])
				line, col := GetLineAndColumn(text, parse.Pos(match[0]))
				// Find the actual position of "gotype:" in the match
				gotypeLoc := strings.Index(text[match[0]:match[1]], "gotype:")
				if gotypeLoc >= 0 {
					col += gotypeLoc + len("gotype:") + 1 // +1 for the space after "gotype:"
				}
				hints = append(hints, TypeHint{
					TypePath: typePath,
					Line:     line,
					Column:   col,
					Scope:    scope,
				})
			}
		}
		return hints
	}

	// Extract type hints from the entire template content first
	hints := extractTypeHints(contentStr, "")
	if len(hints) > 0 {
		info.TypeHints = hints
	}

	// Create a template with all necessary functions to avoid parsing errors
	tmpl := template.New(filename).Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	})

	// Parse the template
	parsedTmpl, err := tmpl.Parse(contentStr)
	if err != nil {
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	// Track seen variables to avoid duplicates but maintain references
	seenVars := make(map[string]*VariableLocation)

	// Helper function to create a variable location
	createVarLocation := func(field *parse.FieldNode, scope string) *VariableLocation {
		line, col := GetLineAndColumn(contentStr, field.Position())
		endLine, endCol := GetLineAndColumn(contentStr, field.Position()+parse.Pos(len(field.String())-1))
		fullName := ""
		for _, ident := range field.Ident {
			fullName += ident + "."
		}
		fullName = strings.TrimSuffix(fullName, ".")

		if existing, ok := seenVars[fullName]; ok {
			return existing
		}

		item := &VariableLocation{
			Name:    fullName,
			Line:    line,
			Column:  col,
			EndLine: endLine,
			EndCol:  endCol,
			Scope:   scope,
		}
		seenVars[fullName] = item
		info.Variables = append(info.Variables, *item)
		return item
	}

	// Walk the AST and collect variables, functions, and type hints
	var walk func(node parse.Node, scope string) error
	walk = func(node parse.Node, scope string) error {
		if node == nil {
			return nil
		}

		switch n := node.(type) {
		case *parse.ActionNode:
			if n.Pipe != nil {
				// Only handle variables that are direct references (not part of a pipe operation)
				if len(n.Pipe.Cmds) == 1 && len(n.Pipe.Cmds[0].Args) == 1 {
					if field, ok := n.Pipe.Cmds[0].Args[0].(*parse.FieldNode); ok {
						createVarLocation(field, scope)
					}
				}
			}
			if err := walk(n.Pipe, scope); err != nil {
				return err
			}
		case *parse.IfNode:
			// Handle if condition
			if n.Pipe != nil {
				for _, cmd := range n.Pipe.Cmds {
					for _, arg := range cmd.Args {
						if field, ok := arg.(*parse.FieldNode); ok {
							createVarLocation(field, scope)
						}
					}
				}
			}
			if err := walk(n.Pipe, scope); err != nil {
				return err
			}
			// Handle the body of the if statement
			if err := walk(n.List, scope); err != nil {
				return err
			}
			// Handle the else clause if it exists
			if n.ElseList != nil {
				if err := walk(n.ElseList, scope); err != nil {
					return err
				}
			}
		case *parse.ListNode:
			if n != nil {
				for _, node := range n.Nodes {
					if err := walk(node, scope); err != nil {
						return err
					}
				}
			}
		// case *parse.WithNode:
		// 	// Extract type hints from the with block
		// 	if n.List != nil && n.List.Nodes != nil && len(n.List.Nodes) > 0 {
		// 		firstNode := n.List.Nodes[0]
		// 		if text, ok := firstNode.(*parse.TextNode); ok {
		// 			hints := extractTypeHints(string(text.Text), scope+":with")
		// 			info.TypeHints = append(info.TypeHints, hints...)
		// 		}
		// 	}
		// 	if err := walk(n.List, scope+":with"); err != nil {
		// 		return err
		// 	}
		// case *parse.RangeNode:
		// 	// Extract type hints from the range block
		// 	if n.List != nil && n.List.Nodes != nil && len(n.List.Nodes) > 0 {
		// 		firstNode := n.List.Nodes[0]
		// 		if text, ok := firstNode.(*parse.TextNode); ok {
		// 			hints := extractTypeHints(string(text.Text), scope+":range")
		// 			info.TypeHints = append(info.TypeHints, hints...)
		// 		}
		// 	}
		// 	if err := walk(n.List, scope+":range"); err != nil {
		// 		return err
		// 	}
		case *parse.PipeNode:
			if n != nil {
				var lastResult types.Type

				for i, cmd := range n.Cmds {
					args := make([]types.Type, 0)

					// If this isn't the first command in the pipe, add the result of the previous command as first arg
					if i > 0 && lastResult != nil {
						args = append(args, lastResult)
					} else {
						// Process all arguments except the function name
						for j, arg := range cmd.Args {
							if j == 0 {
								// Skip the function name itself
								continue
							}
							switch v := arg.(type) {
							case *parse.FieldNode:
								item := createVarLocation(v, scope)
								args = append(args, item)
								lastResult = item
							case *parse.StringNode:
								args = append(args, types.Typ[types.String])
							}
						}
					}

					// Process the function itself
					if len(cmd.Args) > 0 {
						if fn, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
							line, col := GetLineAndColumn(contentStr, fn.Position())
							endLine, endCol := GetLineAndColumn(contentStr, fn.Position()+parse.Pos(len(fn.String())))

							item := VariableLocation{
								Name:            fn.Ident,
								Line:            line,
								Column:          col,
								EndLine:         endLine,
								EndCol:          endCol,
								MethodArguments: args,
								Scope:           scope,
							}
							info.Functions = append(info.Functions, item)
							lastResult = &item
						} else if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
							// Handle field nodes that are part of a pipe operation
							item := createVarLocation(field, scope)
							lastResult = item
						}
					}
				}
			}
		}
		return nil
	}

	// Walk through all templates in the common.tmpl map
	for _, t := range parsedTmpl.Templates() {
		if t.Tree != nil {
			if err := walk(t.Tree.Root, ""); err != nil {
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
	Scope           string // The scope of the variable (e.g., template name or block ID)
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
	Scope    string // The scope of the type hint (e.g., template name or block ID)
}
