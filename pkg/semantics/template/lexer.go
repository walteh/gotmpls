// Package template provides parsing and analysis for Go templates with support for multiple text formats.
package template

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Lexer State Machine:
//
// Root (initial state)
//   |
//   +-- [{{] -------> Action
//   |                   |
//   |                   +-- [/*] -------> Comment
//   |                   |                   |
//   |                   |                   +-- [*/] ----> Root
//   |                   |
//   |                   +-- ["|"] -------> Pipeline
//   |                   |                   |
//   |                   |                   +-- [}}] ----> Root
//   |                   |
//   |                   +-- [}}] --------> Root
//   |
//   +-- [text] -----> Root
//
// State Descriptions:
//   Root:
//     - Matches text outside delimiters
//     - Recognizes {{ to enter Action state
//     - Handles single-line comments without state change
//     - Captures any non-delimiter text
//
//   Action:
//     - Inside {{ ... }}
//     - Recognizes:
//       * Whitespace (ignored)
//       * Comments (/*...*/})
//       * Pipeline separator (|)
//       * Field access (.)
//       * Numbers
//       * String literals
//       * Operators
//       * Keywords
//       * Identifiers
//       * Closing delimiter (}})
//
//   Comment:
//     - Inside {{/* ... */}}
//     - Captures comment text
//     - Recognizes */ to exit
//
//   String:
//     - Inside quoted strings
//     - Handles escaped characters
//     - Recognizes closing quote

var (
	// TemplateLexer is a stateful lexer for Go templates with support for different text formats
	TemplateLexer = lexer.MustStateful(lexer.Rules{
		"Root": { // Initial state - processes text and opening delimiters
			{`InlineComment`, `{{-?\s*/\*.*?\*/\s*-?}}`, nil}, // Handle single-line comments without state change
			{`OpenDelim`, `{{-?`, lexer.Push("Action")},       // Enter action state
			{`Text`, `[^{]+|{[^{]|{$`, nil},                   // Any text that's not an action start
			{`Char`, `.|\n`, nil},                             // Catch-all for single chars
		},
		"Action": { // Inside {{ ... }}
			{`whitespace`, `\s+`, nil},
			{`CommentStart`, `/\*`, lexer.Push("Comment")},                             // Enter comment state
			{`Pipe`, `\|`, nil},                                                        // Pipeline separator
			{`Dot`, `\.`, nil},                                                         // Field access
			{`Number`, `[-+]?\d*\.?\d+`, nil},                                          // Numbers
			{`String`, `"(?:\\"|[^"])*"`, nil},                                         // String literals
			{`Operator`, `==|!=|<=|>=|&&|\|\||[!<>]=?`, nil},                           // Operators
			{`Keyword`, `(?i)\b(if|else|range|with|template|block|define|end)\b`, nil}, // Keywords
			{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`, nil},                                   // Identifiers
			{`CloseDelim`, `-?}}`, lexer.Pop()},                                        // Exit action state
			{`Char`, `.|\n`, nil},                                                      // Catch-all for single chars
		},
		"Comment": { // Inside {{/* ... */}}
			{`CommentEnd`, `\*/`, lexer.Pop()},   // Exit comment state
			{`CommentText`, `[^*]+|\*[^/]`, nil}, // Comment content
		},
		"String": { // Inside quoted strings
			{`StringEnd`, `"`, lexer.Pop()}, // Exit string state
			{`Escaped`, `\\.`, nil},         // Escaped characters
			{`StringText`, `[^"\\]+`, nil},  // String content
		},
	})

	// Parser is the compiled template parser
	Parser = participle.MustBuild[Template](
		participle.Lexer(TemplateLexer),
		participle.Elide("whitespace"),
		participle.UseLookahead(2),
	)
)
