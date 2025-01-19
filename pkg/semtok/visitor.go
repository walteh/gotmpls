/*
AST Visitor for Token Generation:
------------------------------

The visitor walks the template AST and converts nodes to semantic tokens:

	AST Tree                 Token Sources
	--------                 -------------
	Template                 Standard Parser
	   |                          |
	   +-> Action Node     <--> Diagnostic Parser
	   |      |                   |
	   |      +-> Field     Position Info
	   |      |             Type Info
	   |      +-> Ident     Scope Info
	   |
	   +-> Text Node        Standard Parser
	          |                  |
	          +-> String    Position Info

Each visitor method combines information from both parsers:
1. Standard parser for positions and basic syntax
2. Diagnostic parser for type information and scope
*/
package semtok

import (
	"context"
	"strings"

	"github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/std/text/template/parse"
)

// tokenVisitor collects semantic tokens while walking the AST
type tokenVisitor struct {
	// tokens collects the semantic tokens found during traversal
	tokens []Token

	// content is the original template text
	content []byte

	// ctx is used for diagnostic parser operations
	ctx context.Context

	// parsedFile contains diagnostic information
	parsedFile *parser.ParsedTemplateFile

	// currentBlock tracks which block we're currently processing
	currentBlock *parser.BlockInfo
}

// newVisitor creates a new token visitor for the given content
func newVisitor(ctx context.Context, content []byte) (*tokenVisitor, error) {
	// Parse with our diagnostic parser
	parsedFile, err := parser.Parse(ctx, "", content)
	if err != nil {
		return nil, err
	}

	return &tokenVisitor{
		tokens:     make([]Token, 0),
		content:    content,
		ctx:        ctx,
		parsedFile: parsedFile,
	}, nil
}

func (v *tokenVisitor) visitTree(node *parse.Tree) {
	if node.CanRelyOnKeyword() {
		v.tokens = append(v.tokens, Token{
			Type:     TokenKeyword,
			Modifier: ModifierNone,
			Range:    position.NewKeywordPosition(node),
		})
	}
	return
}

// visitNode dispatches to the appropriate visit method based on node type
func (v *tokenVisitor) visitNode(node parse.Node) error {
	switch n := node.(type) {
	case *parse.ActionNode:
		return v.visitAction(n)
	case *parse.FieldNode:
		return v.visitField(n)
	case *parse.ListNode:
		return v.visitList(n)
	case *parse.IdentifierNode:
		return v.visitIdentifier(n)
	case *parse.StringNode:
		return v.visitString(n)
	case *parse.PipeNode:
		return v.visitPipe(n)
	case *parse.CommandNode:
		return v.visitCommand(n)
	case *parse.IfNode:
		return v.visitIf(n)
	case *parse.RangeNode:
		return v.visitRange(n)
	case *parse.WithNode:
		return v.visitWith(n)
	case *parse.EndNode:
		return v.visitEnd(n)
	case *parse.TextNode:
		return v.visitText(n)
	case *parse.CommentNode:
		return v.visitComment(n)
	case *parse.DotNode:
		return v.visitDot(n)
	case *parse.NumberNode:
		return v.visitNumber(n)
	default:
		// TODO(@semtok): Add support for other node types
		return nil
	}
}

func (v *tokenVisitor) visitText(node *parse.TextNode) error {
	v.tokens = append(v.tokens, Token{
		Type:     TokenString,
		Modifier: ModifierNone,
		Range:    position.NewTextNodePosition(node),
	})
	return nil
}

func (v *tokenVisitor) visitEnd(node *parse.EndNode) error {
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})
	return nil
}

func (v *tokenVisitor) visitComment(node *parse.CommentNode) error {
	v.tokens = append(v.tokens, Token{
		Type:     TokenComment,
		Modifier: ModifierNone,
		Range:    position.NewCommentNodePosition(node),
	})
	return nil
}

// visitList processes a list of nodes
func (v *tokenVisitor) visitList(node *parse.ListNode) error {
	if node == nil {
		return nil
	}
	for _, n := range node.Nodes {
		if err := v.visitNode(n); err != nil {
			return err
		}
	}
	return nil
}

// visitAction processes an action node (e.g., {{ .Name }})
func (v *tokenVisitor) visitAction(node *parse.ActionNode) error {
	if node.Pipe != nil {
		return v.visitPipe(node.Pipe)
	}
	return nil
}

// visitPipe processes a pipe node (e.g., .Name | printf)
func (v *tokenVisitor) visitPipe(node *parse.PipeNode) error {
	// Visit all commands in the pipe
	for _, cmd := range node.Cmds {
		// First argument might be a function name
		if len(cmd.Args) > 0 {
			if err := v.visitNode(cmd.Args[0]); err != nil {
				return err
			}
		}

		// Visit remaining arguments
		for _, arg := range cmd.Args[1:] {
			if err := v.visitNode(arg); err != nil {
				return err
			}
		}
	}
	return nil
}

// visitField processes a field node (e.g., .Name)
func (v *tokenVisitor) visitField(node *parse.FieldNode) error {

	// Create the position
	rawPos := position.NewFieldNodePosition(node)

	// Try to get additional info from diagnostic parser
	var modifier TokenModifier = ModifierNone
	if v.currentBlock != nil {
		if varLoc := v.currentBlock.GetVariableFromPosition(rawPos); varLoc != nil {
			// If this is a first occurrence, mark it as a declaration
			// TODO(@semtok): Implement declaration detection
		}
	}

	// If this is just a dot, create a dot token
	if len(node.Ident) == 0 {
		v.tokens = append(v.tokens, Token{
			Type:     TokenVariable,
			Modifier: modifier,
			Range:    position.NewBasicPosition(".", int(node.Pos)),
		})
		return nil
	}

	// Create a variable token for the field
	v.tokens = append(v.tokens, Token{
		Type:     TokenVariable,
		Modifier: modifier,
		Range:    rawPos,
	})
	return nil
}

// visitIdentifier processes an identifier node (e.g., printf)
func (v *tokenVisitor) visitIdentifier(node *parse.IdentifierNode) error {
	// Create position from node
	pos := position.NewIdentifierNodePosition(node)

	// Try to get additional info from diagnostic parser
	var tokenType TokenType = TokenFunction
	var modifier TokenModifier = ModifierNone

	// Check if this is a keyword
	switch node.Ident {
	case "if", "range", "with", "template", "define", "end":
		tokenType = TokenKeyword
	}

	v.tokens = append(v.tokens, Token{
		Type:     tokenType,
		Modifier: modifier,
		Range:    pos,
	})
	return nil
}

// visitString processes a string node (e.g., "hello")
func (v *tokenVisitor) visitString(node *parse.StringNode) error {
	// Handle escaped quotes in the text
	text := node.Text
	if !strings.HasPrefix(text, `"`) {
		text = `"` + text + `"`
	}

	v.tokens = append(v.tokens, Token{
		Type:     TokenString,
		Modifier: ModifierNone,
		Range:    position.NewBasicPosition(text, int(node.Pos)),
	})
	return nil
}

// visitCommand processes a command node (e.g., printf "hello")
func (v *tokenVisitor) visitCommand(node *parse.CommandNode) error {
	// Visit all arguments
	for _, arg := range node.Args {
		if err := v.visitNode(arg); err != nil {
			return err
		}
	}
	return nil
}

// visitIf processes an if node (e.g., {{ if .Ready }}...{{ end }})
func (v *tokenVisitor) visitIf(node *parse.IfNode) error {
	// Add the 'if' keyword token
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the condition
	if err := v.visitPipe(node.Pipe); err != nil {
		return err
	}

	// Visit the true branch
	if err := v.visitList(node.List); err != nil {
		return err
	}

	// Visit the else branch if it exists
	if node.ElseList != nil {
		// TODO(@semtok): Add 'else' token
		if err := v.visitList(node.ElseList); err != nil {
			return err
		}
	}

	return nil
}

// visitRange processes a range node (e.g., {{ range .Items }}...{{ end }})
func (v *tokenVisitor) visitRange(node *parse.RangeNode) error {
	// Add the 'range' keyword token
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the range expression
	if err := v.visitPipe(node.Pipe); err != nil {
		return err
	}

	// Visit the range body
	if err := v.visitList(node.List); err != nil {
		return err
	}

	// Visit the else branch if it exists
	if node.ElseList != nil {
		// TODO(@semtok): Add 'else' token
		if err := v.visitList(node.ElseList); err != nil {
			return err
		}
	}

	return nil
}

// visitWith processes a with node (e.g., {{ with .User }}...{{ end }})
func (v *tokenVisitor) visitWith(node *parse.WithNode) error {
	// Add the 'with' keyword token
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the with expression
	if err := v.visitPipe(node.Pipe); err != nil {
		return err
	}

	// Visit the with body
	if err := v.visitList(node.List); err != nil {
		return err
	}

	// Visit the else branch if it exists
	if node.ElseList != nil {
		// TODO(@semtok): Add 'else' token
		if err := v.visitList(node.ElseList); err != nil {
			return err
		}
	}

	return nil
}

// visitDot processes a dot node (e.g., .)
func (v *tokenVisitor) visitDot(node *parse.DotNode) error {
	v.tokens = append(v.tokens, Token{
		Type:     TokenVariable,
		Modifier: ModifierNone,
		Range:    position.NewDotNodePosition(node),
	})
	return nil
}

// visitNumber processes a number node (e.g., 0, 1.5)
func (v *tokenVisitor) visitNumber(node *parse.NumberNode) error {
	v.tokens = append(v.tokens, Token{
		Type:     TokenNumber,
		Modifier: ModifierNone,
		Range:    position.NewNumberNodePosition(node),
	})
	return nil
}

// getTokens returns the collected tokens
func (v *tokenVisitor) getTokens() []Token {
	return v.tokens
}
