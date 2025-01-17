package template

import (
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Template represents a complete template file
type Template struct {
	Pos lexer.Position

	Nodes []*Node `@@*`
}

// Node represents a node in the template
type Node struct {
	Pos lexer.Position

	Text    *string  `(  @Text`
	Action  *Action  ` | @@`
	Comment *Comment ` | @@`
	Control *Control ` | @@ )`
}

// Action represents a template action (e.g., {{.Name}})
type Action struct {
	Pos lexer.Position

	OpenDelim  string    `@OpenDelim`
	Pipeline   *Pipeline `@@?`
	CloseDelim string    `@CloseDelim`
}

// Comment represents a template comment
type Comment struct {
	Pos lexer.Position

	OpenDelim  string `@OpenDelim`
	Content    string `@CommentText`
	CloseDelim string `@CloseDelim`
}

// Control represents a control structure (if, range, with, etc.)
type Control struct {
	Pos lexer.Position

	OpenDelim  string    `@OpenDelim`
	Keyword    string    `@("if" | "range" | "with" | "template" | "block" | "define" | "end" | "else")`
	Pipeline   *Pipeline `@@?`
	CloseDelim string    `@CloseDelim`
}

// Pipeline represents a chain of commands
type Pipeline struct {
	Pos lexer.Position

	Cmd  *Command  `@@`
	Next *Pipeline `( "|" @@ )?`
}

// ToString returns the string representation of the pipeline
func (p *Pipeline) ToString() string {
	if p == nil {
		return ""
	}
	var parts []string
	current := p
	for current != nil {
		parts = append(parts, current.Cmd.ToString())
		current = current.Next
	}
	return strings.Join(parts, " | ")
}

// Command represents a command with its identifier and arguments
type Command struct {
	Pos lexer.Position

	Identifier string `@(Ident | DotIdent)`
	Args       []Arg  `@@*`
}

// ToString returns the string representation of the command
func (c *Command) ToString() string {
	if c == nil {
		return ""
	}
	var parts []string
	parts = append(parts, c.Identifier)
	for _, arg := range c.Args {
		parts = append(parts, arg.ToString())
	}
	return strings.Join(parts, " ")
}

// Arg represents an argument to a command
type Arg struct {
	Pos lexer.Position

	Number   *string `(  @Number`
	String   *string ` | @String`
	Variable *string ` | @(Ident | DotIdent) )`
}

// ToString returns the string representation of the argument
func (a *Arg) ToString() string {
	switch {
	case a.Number != nil:
		return *a.Number
	case a.String != nil:
		return *a.String
	case a.Variable != nil:
		return *a.Variable
	default:
		return ""
	}
}
