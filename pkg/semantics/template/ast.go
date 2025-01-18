// Package template provides parsing and analysis for Go templates with support for multiple text formats.
package template

// AST Structure for Template Parser
//
// This file outlines the proposed structure for parsing Go templates into an AST.
// The parser will support multiple output formats (starting with JSON and TXT).
//
// ╭──────────────────────────────────────────────────────────╮
// │                    AST Structure                         │
// │                                                          │
// │           Template                                       │
// │              │                                           │
// │              ├─── Node                                   │
// │              │     ├─── TextNode                        │
// │              │     │     └─── ContentParser             │
// │              │     ├─── ActionNode                      │
// │              │     └─── CommentNode                     │
// │              │                                           │
// │              └─── ContentParser                         │
// │                    ├─── TextParser (default)            │
// │                    ├─── JSONParser                      │
// │                    └─── (future: YAML, JS, etc)         │
// ╰──────────────────────────────────────────────────────────╯
//
// Lexer State Machine:
// ┌──────────────────┐
// │       Root       │
// │                  │
// │  [Text outside   │
// │   delimiters]    │
// └──────────┬───────┘
//            │
//            │ "{{" encountered
//            ▼
// ┌──────────────────┐
// │      Action      │
// │                  │
// │  [Inside action  │
// │   delimiters]    │
// └──────────┬───────┘
//            │
//            │ "}}" encountered
//            ▼
// Back to Root State
//
// The lexer uses different rules based on the current state:
// - Root State: Matches text and opening delimiters
// - Action State: Matches identifiers, operators, and closing delimiters
//
// Pipeline Structure:
// ┌──────────────────┐
// │     Pipeline     │
// │                  │
// │  First Command   │
// │  Rest Commands   │
// └──────────┬───────┘
//            │
//            ├── FuncCall
//            │   └── Args
//            │
//            ├── MethodCall
//            │   └── Args
//            │
//            └── Argument
//                ├── String
//                ├── Number
//                ├── Bool
//                ├── Nil
//                ├── Dot
//                ├── Field
//                └── SubExpr

// Template represents a complete template with a sequence of nodes
type Template struct {
	Nodes []*Node `@@*`
}

// Node represents a single node in the template
// It can be either a TextNode, ActionNode, or CommentNode
type Node struct {
	Text    *TextNode    `@@ |`
	Action  *ActionNode  `@@ |`
	Comment *CommentNode `@@`
}

func (n *Node) ToString() string {
	if n.Text != nil {
		return n.Text.ToString()
	}
	if n.Action != nil {
		return n.Action.ToString()
	}
	if n.Comment != nil {
		return n.Comment.ToString()
	}
	return ""
}

// TextNode represents plain text outside of delimiters
type TextNode struct {
	Text string `@Text`
}

func (t *TextNode) ToString() string {
	return t.Text
}

// ActionNode represents code inside delimiters {{...}}
type ActionNode struct {
	Pipeline *Pipeline `"{{" @@ "}}"`
}

func (a *ActionNode) ToString() string {
	if a.Pipeline == nil {
		return "{{}}"
	}
	return "{{" + a.Pipeline.ToString() + "}}"
}

// CommentNode represents comments {{/* ... */}}
type CommentNode struct {
	Text string `@InlineComment`
}

func (c *CommentNode) ToString() string {
	return c.Text
}

// Pipeline represents a sequence of commands
type Pipeline struct {
	First *Command   `@@`
	Rest  []*Command `( "|" @@ )*`
}

func (p *Pipeline) ToString() string {
	if p.First == nil {
		return ""
	}
	result := p.First.ToString()
	for _, cmd := range p.Rest {
		result += " | " + cmd.ToString()
	}
	return result
}

// Command represents a single command in a pipeline
// It can be either a function call, method call, or variable reference
type Command struct {
	FuncCall   *FuncCall   `@@ |`
	MethodCall *MethodCall `@@ |`
	Argument   *Argument   `@@`
}

func (c *Command) ToString() string {
	if c.FuncCall != nil {
		return c.FuncCall.ToString()
	}
	if c.MethodCall != nil {
		return c.MethodCall.ToString()
	}
	if c.Argument != nil {
		return c.Argument.ToString()
	}
	return ""
}

// FuncCall represents a function call with arguments
type FuncCall struct {
	Name string      `@Ident`
	Args []*Argument `@@*`
}

func (f *FuncCall) ToString() string {
	result := f.Name
	for _, arg := range f.Args {
		result += " " + arg.ToString()
	}
	return result
}

// MethodCall represents a method call on a variable or field
type MethodCall struct {
	Name string      `@Ident`
	Args []*Argument `@@*`
}

func (m *MethodCall) ToString() string {
	result := m.Name
	for _, arg := range m.Args {
		result += " " + arg.ToString()
	}
	return result
}

// Argument represents a value that can be passed to a function or method
type Argument struct {
	String  *string   `@String |`
	Number  *string   `@Number |`
	Bool    *bool     `( @"true" | @"false" ) |`
	Nil     bool      `@"nil" |`
	Field   *Field    `@@ |`
	SubExpr *Pipeline `"(" @@ ")"`
}

func (a *Argument) ToString() string {
	if a.String != nil {
		return *a.String
	}
	if a.Number != nil {
		return *a.Number
	}
	if a.Bool != nil {
		if *a.Bool {
			return "true"
		}
		return "false"
	}
	if a.Nil {
		return "nil"
	}
	if a.Field != nil {
		return a.Field.ToString()
	}
	if a.SubExpr != nil {
		return "(" + a.SubExpr.ToString() + ")"
	}
	return ""
}

// Field represents a field access on a variable
type Field struct {
	Dot   bool     `@"."`
	Names []string `@Ident ( "." @Ident )*`
}

func (f *Field) ToString() string {
	result := ""
	if f.Dot {
		result = "."
	}
	for i, name := range f.Names {
		if i > 0 {
			result += "."
		}
		result += name
	}
	return result
}

// ContentParser is an interface for parsing text into different formats
type ContentParser interface {
	Parse(text string) (interface{}, error)
}

// TextParser implements ContentParser for unstructured text
type TextParser struct{}

func (p *TextParser) Parse(text string) (interface{}, error) {
	return text, nil
}

// JSONParser implements ContentParser for JSON format
type JSONParser struct {
	Examples map[string]interface{}
}

func (p *JSONParser) Parse(text string) (interface{}, error) {
	// TODO: Implement JSON parsing based on examples
	return nil, nil
}
