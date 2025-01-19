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
	"fmt"

	diagnostic "github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/std/text/template/parse"
)

// tokenVisitor implements parse.NodeVisitor to collect semantic tokens
type tokenVisitor struct {
	// tokens collects the semantic tokens
	tokens []Token

	// currentCommand tracks the current command node being processed
	currentCommand *parse.CommandNode

	// currentBlock tracks the current block being processed
	currentBlock *diagnostic.BlockInfo
}

// newTokenVisitor creates a new tokenVisitor
func newTokenVisitor(block *diagnostic.BlockInfo) *tokenVisitor {
	return &tokenVisitor{
		tokens:       make([]Token, 0),
		currentBlock: block,
	}
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

// Visit implements parse.NodeVisitor
func (v *tokenVisitor) Visit(node parse.Node) {
	switch n := node.(type) {
	case *parse.ActionNode:
		v.visitAction(n)
	case *parse.CommandNode:
		v.currentCommand = n
		v.visitCommand(n)
		v.currentCommand = nil
	case *parse.FieldNode:
		v.visitField(n)
	case *parse.IdentifierNode:
		v.visitIdentifier(n)
	case *parse.StringNode:
		v.visitString(n)
	case *parse.TextNode:
		v.visitText(n)
	case *parse.CommentNode:
		v.visitComment(n)
	case *parse.IfNode:
		v.visitIf(n)
	case *parse.RangeNode:
		v.visitRange(n)
	case *parse.WithNode:
		v.visitWith(n)
	case *parse.EndNode:
		v.visitEnd(n)
	}
}

func (v *tokenVisitor) visitText(node *parse.TextNode) {
	v.tokens = append(v.tokens, Token{
		Type:     TokenString,
		Modifier: ModifierNone,
		Range:    position.NewTextNodePosition(node),
	})
}

func (v *tokenVisitor) visitEnd(node *parse.EndNode) {
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})
}

func (v *tokenVisitor) visitComment(node *parse.CommentNode) {
	v.tokens = append(v.tokens, Token{
		Type:     TokenComment,
		Modifier: ModifierNone,
		Range:    position.NewCommentNodePosition(node),
	})
}

// visitList processes a list of nodes
func (v *tokenVisitor) visitList(node *parse.ListNode) {
	if node == nil {
		return
	}
	for _, n := range node.Nodes {
		v.Visit(n)
	}
}

// visitAction processes an action node (e.g., {{ .Name }})
func (v *tokenVisitor) visitAction(node *parse.ActionNode) {
	if node.Pipe != nil {
		v.visitPipe(node.Pipe)
	}
}

// visitPipe processes a pipeline node (e.g., .Name | upper)
func (v *tokenVisitor) visitPipe(node *parse.PipeNode) {
	for i, cmd := range node.Cmds {
		v.currentCommand = cmd
		v.visitCommand(cmd)
		v.currentCommand = nil

		// Add pipe operator token between commands (but not after the last one)
		if i < len(node.Cmds)-1 {
			nextCmd := node.Cmds[i+1]
			v.tokens = append(v.tokens, Token{
				Type:     TokenOperator,
				Modifier: ModifierNone,
				Range:    position.NewPipeOperatorPosition(nextCmd),
			})
		}
	}
}

// visitCommand processes a command node (e.g., printf "%s" .Name)
func (v *tokenVisitor) visitCommand(node *parse.CommandNode) {
	for _, arg := range node.Args {
		switch n := arg.(type) {
		case *parse.IdentifierNode:
			v.visitIdentifier(n)
		case *parse.StringNode:
			v.visitString(n)
		case *parse.FieldNode:
			v.visitField(n)
		case *parse.DotNode:
			v.visitDot(n)
		case *parse.NumberNode:
			v.visitNumber(n)
		case *parse.CommandNode:
			// Handle nested commands (in parentheses)
			v.visitCommand(n)
		case *parse.PipeNode:
			// Handle nested pipes
			v.visitPipe(n)
		}
	}
}

// visitField processes a field node (e.g., .Name)
func (v *tokenVisitor) visitField(node *parse.FieldNode) {
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
		return
	}

	// Create a variable token for the field
	v.tokens = append(v.tokens, Token{
		Type:     TokenVariable,
		Modifier: modifier,
		Range:    rawPos,
	})
}

// visitIdentifier processes an identifier
func (v *tokenVisitor) visitIdentifier(node *parse.IdentifierNode) {
	// Create a function token
	v.tokens = append(v.tokens, Token{
		Type:     TokenFunction,
		Modifier: ModifierNone,
		Range:    position.NewIdentifierNodePosition(node),
	})
}

// visitString processes a string node (e.g., "hello")
func (v *tokenVisitor) visitString(node *parse.StringNode) {
	v.tokens = append(v.tokens, Token{
		Type:     TokenString,
		Modifier: ModifierNone,
		Range:    position.NewStringNodePosition(node),
	})
}

// isFormatString checks if a string node is being used as a format string
func (v *tokenVisitor) isFormatString(node *parse.StringNode) bool {

	return fmt.Sprintf(node.Text) != node.Text
}

// isFormatSpecifierEnd checks if a character marks the end of a format specifier
func isFormatSpecifierEnd(c byte) bool {
	return c == 'b' || c == 'c' || c == 'd' || c == 'e' || c == 'E' ||
		c == 'f' || c == 'F' || c == 'g' || c == 'G' || c == 'o' ||
		c == 'p' || c == 'q' || c == 's' || c == 't' || c == 'T' ||
		c == 'U' || c == 'v' || c == 'x' || c == 'X'
}

// visitIf processes an if node
func (v *tokenVisitor) visitIf(node *parse.IfNode) {
	// Create a keyword token for "if"
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the condition
	if node.Pipe != nil {
		v.visitPipe(node.Pipe)
	}

	// Visit the list
	if node.List != nil {
		for _, n := range node.List.Nodes {
			v.Visit(n)
		}
	}

	// Visit the else list
	if node.ElseList != nil {
		for _, n := range node.ElseList.Nodes {
			v.Visit(n)
		}
	}
}

// visitRange processes a range node
func (v *tokenVisitor) visitRange(node *parse.RangeNode) {
	// Create a keyword token for "range"
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the pipe
	if node.Pipe != nil {
		v.visitPipe(node.Pipe)
	}

	// Visit the list
	if node.List != nil {
		for _, n := range node.List.Nodes {
			v.Visit(n)
		}
	}
}

// visitWith processes a with node
func (v *tokenVisitor) visitWith(node *parse.WithNode) {
	// Create a keyword token for "with"
	v.tokens = append(v.tokens, Token{
		Type:     TokenKeyword,
		Modifier: ModifierNone,
		Range:    position.NewKeywordPosition(node),
	})

	// Visit the pipe
	if node.Pipe != nil {
		v.visitPipe(node.Pipe)
	}

	// Visit the list
	if node.List != nil {
		for _, n := range node.List.Nodes {
			v.Visit(n)
		}
	}
}

// visitDot processes a dot node (e.g., .)
func (v *tokenVisitor) visitDot(node *parse.DotNode) {
	// Create a variable token for the dot
	v.tokens = append(v.tokens, Token{
		Type:     TokenVariable,
		Modifier: ModifierNone,
		Range:    position.NewDotNodePosition(node),
	})
}

// visitNumber processes a number node (e.g., 42)
func (v *tokenVisitor) visitNumber(node *parse.NumberNode) {
	// Create a number token
	v.tokens = append(v.tokens, Token{
		Type:     TokenNumber,
		Modifier: ModifierNone,
		Range:    position.NewNumberNodePosition(node),
	})
}

// getTokens returns the collected tokens
func (v *tokenVisitor) getTokens() []Token {
	return v.tokens
}
