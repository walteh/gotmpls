// Package template provides parsing and analysis for Go templates
package template

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	// LexerRules defines the lexer rules for Go templates
	LexerRules = lexer.Rules{
		"Root": {
			// Comment start
			{"CommentStart", `{{-?\s*/\*`, lexer.Push("Comment")},
			// Action start
			{"OpenDelim", `{{-?`, lexer.Push("Action")},
			// Text content
			{"Text", `[^{]+|{[^{]|{$`, nil},
			// Catch-all for any remaining characters
			{"Char", `.|\n`, nil},
		},
		"Comment": {
			// Comment content
			{"CommentText", `[^*]+|\*[^/]`, nil},
			// Comment end
			{"CommentEnd", `\*/\s*-?}}`, lexer.Pop()},
		},
		"Action": {
			// Whitespace
			{`whitespace`, `\s+`, nil},
			// Control keywords
			{`If`, `if\b`, nil},
			{`Range`, `range\b`, nil},
			{`With`, `with\b`, nil},
			{`End`, `end\b`, nil},
			{`Else`, `else\b`, nil},
			// Function names and built-ins
			{`Func`, `(?i)\b(len|index|and|or|not|eq|ne|lt|le|gt|ge|printf|print|println|html|js|urlquery|upper|lower)\b`, nil},
			// Operators and punctuation
			{`Pipe`, `\|`, nil},
			{`Dot`, `\.`, nil},
			{`Dollar`, `\$`, nil},
			{`Comma`, `,`, nil},
			{`Assign`, `:=`, nil},
			{`LeftParen`, `\(`, nil},
			{`RightParen`, `\)`, nil},
			// Literals
			{`String`, `"(?:\\"|[^"])*"`, nil},
			{`Number`, `[-+]?\d*\.?\d+`, nil},
			{`Bool`, `true|false`, nil},
			{`Nil`, `nil\b`, nil},
			// Identifiers
			{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
			// Delimiters
			{`CloseDelim`, `-?}}`, lexer.Pop()},
			// Catch any remaining characters
			{`Char`, `.|\n`, nil},
		},
	}

	// TemplateLexer is the stateful lexer for Go templates
	TemplateLexer = lexer.MustStateful(LexerRules)
)

func BuildWithOptionsFunc[T Node](opts ...participle.Option) (*participle.Parser[T], error) {
	args := []participle.Option{
		participle.Lexer(TemplateLexer),
		participle.Elide("whitespace"),
		participle.UseLookahead(2),
		participle.Union[Node](TextNode{}, ActionNode{}, CommentNode{}),
		participle.Union[Command](Term{}, Call{}, Field{}),
	}
	args = append(args, opts...)
	return participle.Build[T](args...)
}
