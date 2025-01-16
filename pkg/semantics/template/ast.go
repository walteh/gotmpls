package template

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Template represents a complete template file
type Template struct {
	Nodes []Node `@@*`
}

// Node represents a single node in the template
type Node struct {
	Text    *TextNode    `  @@`
	Action  *ActionNode  `| @@`
	Comment *CommentNode `| @@`
}

// TextNode represents plain text between actions
type TextNode struct {
	Content string `@Text`
}

// ActionNode represents a template action (anything between {{ and }})
type ActionNode struct {
	OpenDelim  string    `@Delimiter`
	Pipeline   *Pipeline `@@?`
	CloseDelim string    `@Delimiter`
}

// CommentNode represents a template comment
type CommentNode struct {
	OpenDelim  string `@Delimiter`
	Content    string `@Comment`
	CloseDelim string `@Delimiter`
}

// Pipeline represents a sequence of commands
type Pipeline struct {
	Cmd  *Command  `@@`
	Next *Pipeline `( "|" @@ )?`
}

// Command represents a command in a pipeline
type Command struct {
	Identifier string     `@(DotIdent|Ident|Operator)`
	Args       []Argument `@@*`
}

// Argument represents an argument to a command
type Argument struct {
	Variable string `@(DotIdent|Ident|Operator)`
	Number   string `| @Number`
	String   string `| @String`
}

var (
	// TemplateLexer defines the lexer rules for Go templates
	TemplateLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"Comment", `\/\*[^*]*\*\/`},
		{"Delimiter", `{{-?|-?}}`},
		{"Operator", `\||:=|eq|ne|lt|le\b|gt|ge|and|or|not`},
		{"DotIdent", `\.[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Ident", `[$]?[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `[-+]?\d*\.?\d+`},
		{"String", `"(?:\\"|[^"])*"`},
		{"Space", `[ \t]+`},
		{"EOL", `[\n\r]+`},
		{"Text", `[^{]+`},
	})

	// Parser is the compiled template parser
	Parser = participle.MustBuild[Template](
		participle.Lexer(TemplateLexer),
		participle.Elide("Space", "EOL"),
		participle.UseLookahead(2),
	)
)
