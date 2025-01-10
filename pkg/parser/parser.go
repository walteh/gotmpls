package parser

import (
	"context"
	"go/types"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// DefaultTemplateParser is the default implementation of TemplateParser
type DefaultTemplateParser struct{}

// NewDefaultTemplateParser creates a new DefaultTemplateParser
func NewDefaultTemplateParser() *DefaultTemplateParser {
	return &DefaultTemplateParser{}
}

var typeHintRegex = regexp.MustCompile(`{{-?\s*/\*gotype:\s*([^*]+)\s*\*/\s*-?}}`)

// Parse implements TemplateParser
func (p *DefaultTemplateParser) Parse(ctx context.Context, content []byte, filename string) (*TemplateInfo, error) {
	contentStr := string(content)

	doc := position.NewDocument(contentStr)

	info := &TemplateInfo{
		Filename:  filename,
		Variables: make([]VariableLocation, 0),
		Functions: make([]VariableLocation, 0),
		TypeHints: nil,
	}

	// Helper function to extract type hints from a template text
	extractTypeHints := func(text string, scope string) []TypeHint {
		var hints []TypeHint
		matches := typeHintRegex.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				typePath := strings.TrimSpace(text[match[2]:match[3]])
				hints = append(hints, TypeHint{
					TypePath: typePath,
					Position: doc.NewBasicPosition(typePath, match[2]),
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

	// Track seen variables and functions to avoid duplicates but maintain references
	seenVars := position.NewPositionsSeenMap()
	seenFuncs := position.NewPositionsSeenMap()

	// Helper function to create a variable location
	createVarLocation := func(field *parse.FieldNode, scope string) *VariableLocation {
		pos := doc.NewFieldNodePosition(field)

		if seenVars.Has(pos) {
			return nil
		}

		item := &VariableLocation{
			Position: pos,
			Scope:    scope,
		}

		seenVars.Add(pos)
		info.Variables = append(info.Variables, *item)
		return item
	}

	// Helper function to create a function location
	createFuncLocation := func(fn *parse.IdentifierNode, args []types.Type, scope string) *VariableLocation {
		pos := doc.NewIdentifierNodePosition(fn)

		if seenFuncs.Has(pos) {
			return nil
		}

		item := &VariableLocation{
			Position:        pos,
			MethodArguments: args,
			Scope:           scope,
		}

		seenFuncs.Add(pos)
		info.Functions = append(info.Functions, *item)
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
							item := createFuncLocation(fn, args, scope)
							lastResult = item
						} else if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
							// Handle field nodes that are part of a pipe operation
							item := createVarLocation(field, scope)
							lastResult = item
						}
					}
				}
			}
		case *parse.TemplateNode:
			// Handle template nodes (e.g., {{template "header"}})
			if err := walk(n.Pipe, scope); err != nil {
				return err
			}
		case *parse.TextNode:
			// Handle text nodes (e.g., "Address:")
			// if n != nil {
			// 	info.Text = n.Text
			// }
		}

		return nil
	}

	// Walk through all templates in the common.tmpl map
	for _, t := range parsedTmpl.Templates() {
		if t.Tree != nil {
			// Only use template name as scope for defined templates
			currentScope := ""
			if t.Name() != "" && t.Name() != t.ParseName {
				currentScope = t.Name()
			}
			if err := walk(t.Tree.Root, currentScope); err != nil {
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
	Position        position.RawPosition
	MethodArguments []types.Type
	Scope           string // The scope of the variable (e.g., template name or block ID)
}

// Name returns the short name of the variable (last part after dot)
func (v *VariableLocation) Name() string {
	parts := strings.Split(v.Position.Text(), ".")
	return parts[len(parts)-1]
}

// LongName returns the full name of the variable including dots
func (v *VariableLocation) LongName() string {
	return v.Position.Text()
}

// String implements types.Type - returns the short name
func (v *VariableLocation) String() string {
	return v.Name()
}

// Underlying implements types.Type
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
	Position position.RawPosition
	Scope    string // The scope of the type hint (e.g., template name or block ID)
}
