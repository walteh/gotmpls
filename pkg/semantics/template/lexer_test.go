package template

import (
	"bytes"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/require"
)

// Helper function to compare tokens ignoring positions
func compareTokens(t *testing.T, expected, actual []lexer.Token) {
	t.Helper()
	require.Equal(t, len(expected), len(actual), "number of tokens should match")
	for i := range expected {
		require.Equal(t, expected[i].Type, actual[i].Type, "token types should match at position %d", i)
		require.Equal(t, expected[i].Value, actual[i].Value, "token values should match at position %d", i)
	}
}

func TestTemplateLexer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []lexer.Token
	}{
		{
			name:  "simple_text",
			input: "hello world",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["Text"], Value: "hello world"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "simple_action",
			input: "{{.Field}}",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["Dot"], Value: "."},
				{Type: TemplateLexer.Symbols()["Ident"], Value: "Field"},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "pipeline",
			input: "{{.Field | upper}}",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["Dot"], Value: "."},
				{Type: TemplateLexer.Symbols()["Ident"], Value: "Field"},
				{Type: TemplateLexer.Symbols()["Pipe"], Value: "|"},
				{Type: TemplateLexer.Symbols()["Ident"], Value: "upper"},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "comment",
			input: "{{/* hello */}}",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["InlineComment"], Value: "{{/* hello */}}"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "string_literal",
			input: "{{\"hello \\\"world\\\"\"}}",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["String"], Value: "\"hello \\\"world\\\"\""},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "number",
			input: "{{42}}",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["Number"], Value: "42"},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: lexer.EOF, Value: ""},
			},
		},
		{
			name:  "mixed_content",
			input: "Hello {{.Name}}, welcome to {{.Place}}!",
			expected: []lexer.Token{
				{Type: TemplateLexer.Symbols()["Text"], Value: "Hello "},
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["Dot"], Value: "."},
				{Type: TemplateLexer.Symbols()["Ident"], Value: "Name"},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: TemplateLexer.Symbols()["Text"], Value: ", welcome to "},
				{Type: TemplateLexer.Symbols()["OpenDelim"], Value: "{{"},
				{Type: TemplateLexer.Symbols()["Dot"], Value: "."},
				{Type: TemplateLexer.Symbols()["Ident"], Value: "Place"},
				{Type: TemplateLexer.Symbols()["CloseDelim"], Value: "}}"},
				{Type: TemplateLexer.Symbols()["Text"], Value: "!"},
				{Type: lexer.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex, err := TemplateLexer.Lex("", bytes.NewReader([]byte(tt.input)))
			require.NoError(t, err, "lexing should succeed")

			var tokenList []lexer.Token
			for {
				token, err := lex.Next()
				require.NoError(t, err, "getting next token should succeed")
				tokenList = append(tokenList, token)
				if token.Type == lexer.EOF {
					break
				}
			}

			compareTokens(t, tt.expected, tokenList)
		})
	}
}
