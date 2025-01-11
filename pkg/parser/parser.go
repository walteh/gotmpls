package parser

import (
	"context"
	"fmt"
	"go/types"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

func ParseTree(name, text string) (map[string]*parse.Tree, error) {
	treeSet := make(map[string]*parse.Tree)
	t := parse.New(name)
	t.Mode = t.Mode | parse.Mode(parse.ParseComments)
	_, err := t.Parse(text, "{{", "}}", treeSet, ast.Builtins(), ast.Extras())
	return treeSet, err
}

// createVarLocation creates a new variable location and adds it to the seen map
func createVarLocation(field *parse.FieldNode, scope string, seenVars *position.PositionsSeenMap) *VariableLocation {
	pos := position.NewFieldNodePosition(field)

	if seenVars.Has(pos) {
		return nil
	}

	item := &VariableLocation{
		Position: pos,
		Scope:    scope,
	}

	seenVars.Add(pos)
	return item
}

// createFuncLocation creates a new function location and adds it to the seen map
func createFuncLocation(fn *parse.IdentifierNode, args []types.Type, scope string, seenFuncs *position.PositionsSeenMap) *VariableLocation {
	pos := position.NewIdentifierNodePosition(fn)

	if seenFuncs.Has(pos) {
		return nil
	}

	item := &VariableLocation{
		Position:        pos,
		MethodArguments: args,
		Scope:           scope,
	}

	seenFuncs.Add(pos)
	return item
}

// extractTypeHint extracts a type hint from a comment node
func extractTypeHint(cmt *parse.CommentNode, scope string, parent parse.Node) *TypeHint {
	text := strings.TrimSpace(cmt.Text)
	if !strings.HasPrefix(text, "/*gotype:") || !strings.HasSuffix(text, "*/") {
		return nil
	}

	// Extract the type path from between "/*gotype:" and "*/"
	typePath := strings.TrimSpace(text[9 : len(text)-2])
	if typePath == "" {
		return nil
	}

	th := &TypeHint{
		TypePath: typePath,
		Position: position.NewBasicPosition(typePath, int(cmt.Pos)),
		Scope:    scope,
	}

	if ln, ok := parent.(*parse.ListNode); ok && len(ln.Nodes) > 0 {
		th.StartPosition = position.NewBasicPosition(ln.Nodes[0].String(), int(ln.Nodes[0].Position()))
		th.EndPosition = position.NewBasicPosition(ln.String(), int(ln.Nodes[len(ln.Nodes)-1].Position()))
	}

	return th
}

// walkNode processes a single node in the AST
func (block *BlockInfo) walkNode(node parse.Node, scope string, parent parse.Node, seenVars, seenFuncs *position.PositionsSeenMap) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.CommentNode:
		if hint := extractTypeHint(n, scope, parent); hint != nil {
			if block.TypeHint != nil {
				return errors.New("multiple type hints found")
			}
			block.TypeHint = hint
		}
	case *parse.ActionNode:
		if n.Pipe != nil {
			// Only handle variables that are direct references (not part of a pipe operation)
			if len(n.Pipe.Cmds) == 1 && len(n.Pipe.Cmds[0].Args) == 1 {
				if field, ok := n.Pipe.Cmds[0].Args[0].(*parse.FieldNode); ok {
					if item := createVarLocation(field, scope, seenVars); item != nil {
						block.Variables = append(block.Variables, *item)
					}
				}
			}
		}
		if err := block.walkNode(n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
	case *parse.IfNode:
		// Handle if condition
		if n.Pipe != nil {
			for _, cmd := range n.Pipe.Cmds {
				for _, arg := range cmd.Args {
					if field, ok := arg.(*parse.FieldNode); ok {
						if item := createVarLocation(field, scope, seenVars); item != nil {
							block.Variables = append(block.Variables, *item)
						}
					}
				}
			}
		}
		if err := block.walkNode(n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
		if err := block.walkNode(n.List, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := block.walkNode(n.ElseList, scope, node, seenVars, seenFuncs); err != nil {
				return err
			}
		}
	case *parse.ListNode:
		if n != nil {
			for _, z := range n.Nodes {
				if err := block.walkNode(z, scope, node, seenVars, seenFuncs); err != nil {
					return err
				}
			}
		}
	case *parse.PipeNode:
		if n != nil {
			var lastResult types.Type

			for i, cmd := range n.Cmds {
				args := make([]types.Type, 0)

				if i > 0 && lastResult != nil {
					args = append(args, lastResult)
				} else {
					for j, arg := range cmd.Args {
						if j == 0 {
							continue
						}
						switch v := arg.(type) {
						case *parse.FieldNode:
							item := createVarLocation(v, scope, seenVars)
							if item != nil {
								block.Variables = append(block.Variables, *item)
								args = append(args, item)
								lastResult = item
							}
						case *parse.StringNode:
							args = append(args, types.Typ[types.String])
						}
					}
				}

				if len(cmd.Args) > 0 {
					if fn, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
						item := createFuncLocation(fn, args, scope, seenFuncs)
						if item != nil {
							block.Functions = append(block.Functions, *item)
							lastResult = item
						}
					} else if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
						item := createVarLocation(field, scope, seenVars)
						if item != nil {
							block.Variables = append(block.Variables, *item)
							lastResult = item
						}
					}
				}
			}
		}
	case *parse.TemplateNode:
		if err := block.walkNode(n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
	}

	return nil
}

// Parse parses a template file and returns FileInfo containing all blocks and their information
func Parse(ctx context.Context, content []byte, filename string) (*FileInfo, error) {
	contentStr := string(content)

	fileInfo := &FileInfo{
		Filename:      filename,
		SourceContent: contentStr,
		Blocks:        make([]BlockInfo, 0),
	}

	// Create a template with all necessary functions to avoid parsing errors
	tmpl := template.New(filename).Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	})

	tmpl.Tree = parse.New(filename)
	tmpl.Mode = tmpl.Mode | parse.Mode(parse.ParseComments)
	tmpl.Mode = tmpl.Mode | parse.Mode(parse.SkipFuncCheck)

	// Parse the template
	trees, err := ParseTree(filename, contentStr)
	if err != nil {
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	for name, tree := range trees {
		if _, err := tmpl.AddParseTree(name, tree); err != nil {
			return nil, err
		}
	}

	// Track seen variables and functions to avoid duplicates
	seenVars := position.NewPositionsSeenMap()
	seenFuncs := position.NewPositionsSeenMap()

	// Process each template
	for _, t := range tmpl.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		var startPos position.RawPosition
		if t.Name() == tmpl.ParseName {
			startPos = position.NewBasicPosition("<<SOF>>", -1)
		} else {
			startPos, err = UseRegexToFindStartOfBlock(ctx, contentStr, t.Name())
			if err != nil {
				return nil, errors.Errorf("finding start of block %s: %w", t.Name(), err)
			}
		}

		// Create a new block for this template
		block := BlockInfo{
			Name:          t.Name(),
			Variables:     make([]VariableLocation, 0),
			Functions:     make([]VariableLocation, 0),
			StartPosition: startPos,
			EndPosition:   hackGetEndPositionForBlock(t.Tree),
			node:          t,
		}

		// Set scope based on template name
		scope := ""
		if t.Name() != "" && t.Name() != t.ParseName {
			scope = t.Name()
		}

		// Process the template's AST
		if err := block.walkNode(t.Tree.Root, scope, nil, seenVars, seenFuncs); err != nil {
			return nil, errors.Errorf("failed to walk template %s: %w", t.Name(), err)
		}

		fileInfo.Blocks = append(fileInfo.Blocks, block)
	}

	// Sort blocks by start position
	sort.Slice(fileInfo.Blocks, func(i, j int) bool {
		return fileInfo.Blocks[i].StartPosition.Offset < fileInfo.Blocks[j].StartPosition.Offset
	})

	return fileInfo, nil
}

// UseRegexToFindStartOfBlock finds the start of a template block definition.
// It returns an error if:
// - The block is defined multiple times
// - The block cannot be found
// - The block definition is malformed
func UseRegexToFindStartOfBlock(ctx context.Context, content string, name string) (position.RawPosition, error) {
	// More precise regex that matches the entire block definition including braces
	pattern := fmt.Sprintf(`{{-?\s*(?:define|block)\s+"(?:%s)"(?:\s+\.[^}]*)?(?:\s*-?|\s*)}}`, regexp.QuoteMeta(name))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return position.RawPosition{}, errors.Errorf("invalid block name %q: %w", name, err)
	}

	// Find all matches to check for multiple definitions
	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return position.RawPosition{}, errors.Errorf("block %q not found in template", name)
	}
	if len(matches) > 1 {
		// Get the line numbers for each definition for better error reporting
		var locations []string
		for _, match := range matches {
			line := 1 + strings.Count(content[:match[0]], "\n")
			locations = append(locations, fmt.Sprintf("line %d", line))
		}
		return position.RawPosition{}, errors.Errorf("block %q is defined multiple times: found at %s", name, strings.Join(locations, ", "))
	}

	// Get the matched text and its position
	match := content[matches[0][0]:matches[0][1]]
	return position.RawPosition{
		Text:   match,
		Offset: matches[0][0],
	}, nil
}

func hackGetEndPositionForBlock(t *parse.Tree) position.RawPosition {
	// Access the unexported skipCaller field
	v := reflect.ValueOf(t).Elem() // Get the value of the pointer
	field := v.FieldByName("token")

	if field.IsValid() && field.CanAddr() && field.Type().Kind() == reflect.Array {
		// get the first element
		firstElement := field.Index(0)
		if firstElement.IsValid() && firstElement.CanAddr() {
			typ := firstElement.FieldByName("typ").Int()
			pos := firstElement.FieldByName("pos").Int()
			val := firstElement.FieldByName("val").String()
			if typ == 8 && val == "" {
				return position.NewBasicPosition("<<EOF>>", int(pos))
			}
			return position.NewBasicPosition(val, int(pos))
		}

	}

	panic("failed to find end position for block")
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
	parts := strings.Split(v.Position.Text, ".")
	return parts[len(parts)-1]
}

// LongName returns the full name of the variable including dots
func (v *VariableLocation) LongName() string {
	return v.Position.Text
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
	TypePath      string // e.g. "github.com/walteh/minute-api/proto/cmd/protoc-gen-cdk/generator.BuilderConfig"
	Position      position.RawPosition
	StartPosition position.RawPosition
	EndPosition   position.RawPosition
	Scope         string // The scope of the type hint (e.g., template name or block ID)
}

type FileInfo struct {
	Filename      string
	SourceContent string
	Blocks        []BlockInfo
}

type BlockInfo struct {
	Name          string
	StartPosition position.RawPosition
	TypeHint      *TypeHint
	Variables     []VariableLocation
	Functions     []VariableLocation
	EndPosition   position.RawPosition
	node          *template.Template
}
