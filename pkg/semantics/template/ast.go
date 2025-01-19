// Package template provides parsing and analysis for Go templates
package template

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
)

// Node represents a single node in the template AST
// Go templates have three types of nodes:
// 1. Text nodes: Raw text that is output as-is
// 2. Action nodes: Code within {{ }} that is evaluated
// 3. Comment nodes: Comments within {{/* */}} that are ignored
type Node interface {
	String() string
}

// Command represents a command in a pipeline
// Commands can be either:
// 1. Terms (literals, fields, variables)
// 2. Function/method calls
// Example: upper in {{.Name | upper}}
type Command interface {
	String() string
}

var _ participle.Capture = (*Pipeline)(nil)

// CommandNode is a concrete type that implements Command
// It can be either a Term or a Call
type CommandNode struct {
	Term *Term `  @@` // Literal values, fields, or variables
	Call *Call `| @@` // Function or method calls
}

func (c CommandNode) String() string {
	if c.Term != nil {
		return c.Term.String()
	}
	if c.Call != nil {
		return c.Call.String()
	}
	return ""
}

// Template represents a complete template document
// In Go templates, this is the root node that contains a sequence of text, actions, and comments
// Example:
//
//	Hello {{.Name}}! {{/* greeting */}}
//	├── TextNode("Hello ")
//	├── ActionNode(".Name")
//	├── TextNode("! ")
//	└── CommentNode("/* greeting */")
type Template struct {
	Nodes []Node `@@*` // Captures zero or more nodes of any type
}

func (t Template) String() string {
	var result string
	for _, node := range t.Nodes {
		result += node.String()
	}
	return result
}

// TextNode represents literal text content
// This is any text outside of {{ }} delimiters
// Example: "Hello " in "Hello {{.Name}}"
type TextNode struct {
	Content string `@Text` // Matches text using the Text token pattern
}

func (t TextNode) String() string {
	return t.Content
}

// ActionNode represents a template action within {{ }}
// Actions can be:
// - Field/method access: {{.Name}}, {{.User.GetName}}
// - Function calls: {{printf "%s" .Name}}
// - Pipelines: {{.Name | upper | quote}}
// - Variable declarations: {{$x := .Name}}
type ActionNode struct {
	Start    string    `@OpenDelim`
	Pipeline *Pipeline `@@`
	End      string    `@CloseDelim`
}

func (a ActionNode) String() string {
	return a.Start + a.Pipeline.String() + a.End
}

// CommentNode represents a template comment
// Comments are ignored during template execution
// Example: {{/* this is a comment */}}
type CommentNode struct {
	Start   string   `@CommentStart` // The opening {{/* marker
	Content []string `@CommentText*` // The comment content
	End     string   `@CommentEnd`   // The closing */}} marker
}

func (c CommentNode) String() string {
	return strings.Join(c.Content, "")
}

// Pipeline represents a sequence of commands that can be piped together
type Pipeline struct {
	Variables []*Variable    `( @@ ( "," @@ )* ":=" )?`
	Commands  []*CommandNode `@@ ( "|" @@ )*`
}

func (p Pipeline) String() string {
	var result string
	if len(p.Variables) > 0 {
		for i, v := range p.Variables {
			if i > 0 {
				result += ", "
			}
			result += v.String()
		}
		result += " := "
	}
	for i, cmd := range p.Commands {
		if i > 0 {
			result += " | "
		}
		result += cmd.String()
	}
	return result
}

// Capture implements the participle.Capture interface for Pipeline
func (p *Pipeline) Capture(values []string) error {
	if len(values) == 0 {
		return nil
	}
	p.Commands = append(p.Commands, &CommandNode{
		Term: &Term{
			StringVal: &values[0],
		},
	})
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Pipeline
func (p *Pipeline) UnmarshalText(text []byte) error {
	p.Commands = append(p.Commands, &CommandNode{
		Term: &Term{
			StringVal: ptr(string(text)),
		},
	})
	return nil
}

// ptr returns a pointer to the given string
func ptr(s string) *string {
	return &s
}

// Variable represents a variable declaration
// In Go templates, variables start with $ followed by an identifier
// Example: $x in {{$x := .Name}}
type Variable struct {
	Name string `"$" @Ident` // Matches $name pattern
}

func (v Variable) String() string {
	return "$" + v.Name
}

// Term represents a literal value or reference
// Terms can be:
// 1. String literals: "hello"
// 2. Numbers: 42, 3.14
// 3. Booleans: true, false
// 4. nil
// 5. Field access: .Name
// 6. Variable references: $x
type Term struct {
	StringVal *string  `  @String`
	Number    *float64 `| @Number`
	Bool      *bool    `| @Bool`
	Nil       bool     `| @Nil`
	Field     *Field   `| @@`
	Var       *string  `| "$" @Ident`
}

func (t Term) String() string {
	if t.StringVal != nil {
		return *t.StringVal
	}
	if t.Number != nil {
		return fmt.Sprintf("%v", *t.Number)
	}
	if t.Bool != nil {
		return fmt.Sprintf("%v", *t.Bool)
	}
	if t.Nil {
		return "nil"
	}
	return ""
}

// Call represents a function or method call
// Functions can be:
// 1. Built-in functions: len, index, etc.
// 2. Method calls: .User.GetName
// 3. Chained calls: .User.GetItems.First
// Examples:
// - len .Items
// - index .Items 0
// - .User.GetName
type Call struct {
	// The function name or method chain
	// For built-ins: len, index, etc.
	// For methods: .User.GetName
	Func string `@(Func | Ident)`

	// Arguments can be either:
	// 1. Inside parentheses: len(.Items)
	// 2. Space separated: len .Items
	Args []*Term `@@*`
}

func (c Call) String() string {
	result := c.Func
	for _, arg := range c.Args {
		result += " " + arg.String()
	}
	return result
}

// Field represents field or method access
// Fields start with . and can be chained with dots
// Example: .User.Name or .User.GetName
type Field struct {
	Names []string `"." @Ident ( "." @Ident )*` // Dot-separated identifiers
}

func (f Field) String() string {
	return "." + strings.Join(f.Names, ".")
}
