package parser

import (
	"context"
	"fmt"
	"go/types"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/position"
	"gitlab.com/tozd/go/errors"
)

func ParseTree(name string, text []byte) (map[string]*parse.Tree, error) {
	treeSet := make(map[string]*parse.Tree)
	t := parse.New(name)
	t.Mode = parse.ParseComments | parse.SkipFuncCheck
	_, err := t.Parse(string(text), "{{", "}}", treeSet)
	return treeSet, err
}

func ParseStringToRawTemplate(ctx context.Context, fileName string, content []byte) (*template.Template, error) {
	tmpl := template.New(fileName)
	tmpl.Tree = parse.New(fileName)
	tmpl.Mode = parse.ParseComments | parse.SkipFuncCheck

	treeSet, err := ParseTree(fileName, content)
	if err != nil {
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	for name, tree := range treeSet {
		if _, err := tmpl.AddParseTree(name, tree); err != nil {
			return nil, err
		}
	}

	return tmpl, nil
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
func createFuncLocation(fn *parse.IdentifierNode, args []VariableLocationOrType, scope string, seenFuncs *position.PositionsSeenMap) *VariableLocation {
	pos := position.NewIdentifierNodePosition(fn)

	if seenFuncs.Has(pos) {
		return nil
	}

	item := &VariableLocation{
		Position:      pos,
		PipeArguments: args,
		Scope:         scope,
	}

	seenFuncs.Add(pos)
	return item
}

// extractTypeHint extracts a type hint from a comment node
func extractTypeHint(cmt *parse.CommentNode, scope string) *TypeHint {
	text := strings.TrimSpace(cmt.Text)
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "gotype:") {
		return nil
	}

	text = strings.TrimPrefix(text, "gotype:")
	text = strings.TrimSpace(text)

	indexOfText := strings.Index(cmt.Text, text)

	th := &TypeHint{
		TypePath: text,
		Position: position.NewBasicPosition(text, int(cmt.Pos)+indexOfText-1),
		Scope:    scope,
	}

	return th
}

// walkNode processes a single node in the AST
func (block *BlockInfo) walkNode(ctx context.Context, node parse.Node, scope string, parent parse.Node, seenVars, seenFuncs *position.PositionsSeenMap) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.CommentNode:
		if hint := extractTypeHint(n, scope); hint != nil {
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
						zerolog.Ctx(ctx).Debug().Msgf("adding variable %s in position %s to block %s", item.Name(), item.Position.ID(), block.Name)
						block.Variables = append(block.Variables, *item)
					}
				}
			}
		}
		if err := block.walkNode(ctx, n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
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
		if err := block.walkNode(ctx, n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
		if err := block.walkNode(ctx, n.List, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := block.walkNode(ctx, n.ElseList, scope, node, seenVars, seenFuncs); err != nil {
				return err
			}
		}
	case *parse.ListNode:
		if n != nil {
			for _, z := range n.Nodes {
				if err := block.walkNode(ctx, z, scope, node, seenVars, seenFuncs); err != nil {
					return err
				}
			}
		}
	case *parse.PipeNode:
		if n != nil {
			var lastResult *VariableLocationOrType

			for i, cmd := range n.Cmds {
				args := make([]VariableLocationOrType, 0)

				if i > 0 && lastResult != nil {
					args = append(args, *lastResult)
				} else {
					for j, arg := range cmd.Args {
						if j == 0 {
							continue
						}
						switch v := arg.(type) {
						case *parse.FieldNode:
							item := createVarLocation(v, scope, seenVars)
							if item != nil {
								ivlt := VariableLocationOrType{Variable: item}
								block.Variables = append(block.Variables, *item)
								args = append(args, ivlt)
								lastResult = &ivlt
							}
						case *parse.StringNode:
							args = append(args, VariableLocationOrType{Type: types.Typ[types.String]})
						}
					}
				}

				if len(cmd.Args) > 0 {
					if fn, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
						allArgs := []VariableLocationOrType{}
						allArgs = append(allArgs, args...)
						item := createFuncLocation(fn, allArgs, scope, seenFuncs)
						if item != nil {
							block.Functions = append(block.Functions, *item)
							lastResult = &VariableLocationOrType{Variable: item}
						}
					} else if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
						item := createVarLocation(field, scope, seenVars)
						if item != nil {
							block.Variables = append(block.Variables, *item)
							lastResult = &VariableLocationOrType{Variable: item}
						}
					}
				}
			}
		}
	case *parse.TemplateNode:
		if err := block.walkNode(ctx, n.Pipe, scope, node, seenVars, seenFuncs); err != nil {
			return err
		}
	}

	return nil
}

// func ParseRegistry(ctx context.Context, data *ast.Registry) ([]*ParsedTemplateFile, error) {
// 	for _, pkg := range data.Packages {

// 		for _, file := range pkg.Templates {
// 			fileInfo := &ParsedTemplateFile{
// 				Filename:      file.Name(),
// 				SourceContent: file.Content,
// 				Blocks:        make([]BlockInfo, 0),
// 			}
// 		}

// 	}
// }

func Parse(ctx context.Context, fileName string, content []byte) (*ParsedTemplateFile, error) {
	tmpl, err := ParseStringToRawTemplate(ctx, fileName, content)
	if err != nil {
		return nil, errors.Errorf("parsing template %s: %w", fileName, err)
	}
	return ParseRawTemplate(ctx, content, tmpl)
}

// Parse parses a template file and returns FileInfo containing all blocks and their information
func ParseRawTemplate(ctx context.Context, content []byte, tmpl *template.Template) (*ParsedTemplateFile, error) {
	contentStr := string(content)

	var err error
	fileInfo := &ParsedTemplateFile{
		Filename:      tmpl.Name(),
		SourceContent: contentStr,
		Blocks:        make([]BlockInfo, 0),
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
		} else {
			scope = t.ParseName
		}

		zerolog.Ctx(ctx).Debug().Msgf("block %s scope: %s", t.Name(), scope)

		// Process the template's AST
		if err := block.walkNode(ctx, t.Tree.Root, scope, nil, seenVars, seenFuncs); err != nil {
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
	if strings.Contains(name, `"`) {
		return position.RawPosition{}, errors.Errorf("block name %q contains quotes", name)
	}

	quotedName := regexp.QuoteMeta(name)
	// More precise regex that matches the entire block definition including braces
	pattern := `(?:{{-?\s*(?:define|block)\s+"(?:` + quotedName + `)"(?:\s+\.[^}]*)?(?:\s*-?|\s*)}})`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return position.RawPosition{}, errors.Errorf("invalid block name %q: %w", name, err)
	}

	// Find all matches to check for multiple definitions
	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return position.RawPosition{}, errors.Errorf("block %q not found in template", quotedName)
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

// // TemplateInfo contains information about a parsed template
// type TemplateInfo struct {
// 	Variables []VariableLocation
// 	Functions []VariableLocation
// 	TypeHints []TypeHint
// 	Filename  string
// }

type VariableLocationOrType struct {
	Variable *VariableLocation
	Type     types.Type
}

// VariableLocation represents a variable usage in a template
type VariableLocation struct {
	Position      position.RawPosition
	PipeArguments []VariableLocationOrType // either a VariableLocation or some other types.Type
	Scope         string                   // The scope of the variable (e.g., template name or block ID)
}

func (me *VariableLocation) GetTypePaths(th *TypeHint) []string {
	if th == nil {
		return []string{me.LongName()}
	}
	returnd := []string{th.TypePath}
	parts := strings.Split(me.LongName(), ".")
	lastPart := parts[len(parts)-1]
	parts = parts[:len(parts)-1]
	for i, part := range parts {

		returnd = append(returnd, fmt.Sprintf("%s.%s", th.TypePath, part))
		if i == len(parts)-1 {
			returnd = append(returnd, fmt.Sprintf("%s.%s[%s]", th.TypePath, part, lastPart))
		}
	}
	return returnd
}

func (me *VariableLocation) GetTypePathNames(th *TypeHint) []string {
	if th == nil {
		return []string{me.LongName()}
	}
	returnd := []string{th.TypePath}
	parts := strings.Split(me.LongName(), ".")
	lastPart := parts[len(parts)-1]
	parts = parts[:len(parts)-1]
	for _, part := range parts {
		returnd = append(returnd, fmt.Sprintf("%s.%s", th.TypePath, part))
	}
	returnd = append(returnd, lastPart)
	return returnd
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

func (me *TypeHint) LocalTypeName() string {
	parts := strings.Split(filepath.Base(me.TypePath), ".")
	return parts[len(parts)-1]
}

type ParsedTemplateFile struct {
	Filename      string
	SourceContent string
	Blocks        []BlockInfo
}

func (me *BlockInfo) GetVariableFromPosition(pos position.RawPosition) *VariableLocation {
	for _, variable := range me.Variables {
		if variable.Position.HasRangeOverlapWith(pos) {
			return &variable
		}
	}
	return nil
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

type PipedArgument struct {
	Variable  *VariableLocation
	Results   []types.Type
	Arguments []PipedArgumentOrType
}

type PipedArgumentOrType struct {
	PipedArgument *PipedArgument
	Type          types.Type
}

func (me *VariableLocation) GetPipedArguments(block *BlockInfo, getReturnTypes func(VariableLocationOrType, *TypeHint) []types.Type) *PipedArgument {
	results := getReturnTypes(VariableLocationOrType{Variable: me}, block.TypeHint)
	args := []PipedArgumentOrType{}
	for _, arg := range me.PipeArguments {
		if arg.Variable != nil {
			args = append(args, PipedArgumentOrType{
				PipedArgument: arg.Variable.GetPipedArguments(block, getReturnTypes),
			})
		} else if arg.Type != nil {
			args = append(args, PipedArgumentOrType{
				Type: arg.Type,
			})
		}
	}
	return &PipedArgument{
		Results:   results,
		Arguments: args,
		Variable:  me,
	}
}
