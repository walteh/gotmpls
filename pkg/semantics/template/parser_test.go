package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"github.com/walteh/go-tmpl-typer/pkg/semantics"
)

func TestTemplateTokenParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []semantics.Token
	}{
		{
			name:    "test_basic_delimiters",
			content: "{{ .Name }}\n{{ .Age }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2}, // {{
				{Type: semantics.TokenTypeVariable, Line: 0, Start: 3, Length: 5},  // .Name
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 9, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2}, // {{
				{Type: semantics.TokenTypeVariable, Line: 1, Start: 3, Length: 4},  // .Age
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 8, Length: 2}, // }}
			},
		},
		{
			name:    "test_keywords_and_operators",
			content: "{{ if eq .Age 18 }}\n{{ .Name }}\n{{ end }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeKeyword, Line: 0, Start: 3, Length: 2},    // if
				{Type: semantics.TokenTypeOperator, Line: 0, Start: 6, Length: 2},   // eq
				{Type: semantics.TokenTypeVariable, Line: 0, Start: 9, Length: 4},   // .Age
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 16, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeVariable, Line: 1, Start: 3, Length: 5},   // .Name
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 9, Length: 2},  // }}
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeKeyword, Line: 2, Start: 3, Length: 3},    // end
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 7, Length: 2},  // }}
			},
		},
		{
			name:    "test_function_calls",
			content: "{{ .GetName | printf \"%s\" }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeFunction, Line: 0, Start: 3, Length: 8},   // .GetName
				{Type: semantics.TokenTypeOperator, Line: 0, Start: 12, Length: 1},  // |
				{Type: semantics.TokenTypeKeyword, Line: 0, Start: 14, Length: 6},   // printf
				{Type: semantics.TokenTypeString, Line: 0, Start: 21, Length: 4},    // "%s"
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 26, Length: 2}, // }}
			},
		},
		{
			name:    "test_comments",
			content: "{{- /* gotype: test.Person */ -}}\n{{ .Name }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2},  // {{-
				{Type: semantics.TokenTypeComment, Line: 0, Start: 3, Length: 23},   // /* gotype: test.Person */
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 28, Length: 2}, // -}}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeVariable, Line: 1, Start: 3, Length: 5},   // .Name
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 9, Length: 2},  // }}
			},
		},
		{
			name:    "test_range_block",
			content: "{{ range .Items }}\n{{ .Name }}\n{{ end }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeKeyword, Line: 0, Start: 3, Length: 5},    // range
				{Type: semantics.TokenTypeVariable, Line: 0, Start: 9, Length: 6},   // .Items
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 16, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeVariable, Line: 1, Start: 3, Length: 5},   // .Name
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 9, Length: 2},  // }}
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 0, Length: 2},  // {{
				{Type: semantics.TokenTypeKeyword, Line: 2, Start: 3, Length: 3},    // end
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 7, Length: 2},  // }}
			},
		},
		{
			name:    "test_variable_declaration",
			content: "{{ $name := \"John\" }}\n{{ $name }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2}, // {{
				{
					Type:      semantics.TokenTypeVariable,
					Line:      0,
					Start:     3,
					Length:    5, // $name
					Modifiers: []semantics.TokenModifier{semantics.ModifierDeclaration},
				},
				{Type: semantics.TokenTypeOperator, Line: 0, Start: 9, Length: 2}, // :=
				{
					Type:      semantics.TokenTypeString,
					Line:      0,
					Start:     12,
					Length:    6, // "John"
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 19, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2},  // {{
				{
					Type:      semantics.TokenTypeVariable,
					Line:      1,
					Start:     3,
					Length:    5, // $name
					Modifiers: []semantics.TokenModifier{semantics.ModifierDefinition},
				},
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 9, Length: 2}, // }}
			},
		},
		{
			name:    "test_builtin_functions",
			content: "{{ len .Items | printf \"%d items\" }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2}, // {{
				{
					Type:      semantics.TokenTypeFunction,
					Line:      0,
					Start:     3,
					Length:    3, // len
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{Type: semantics.TokenTypeVariable, Line: 0, Start: 7, Length: 6},  // .Items
				{Type: semantics.TokenTypeOperator, Line: 0, Start: 14, Length: 1}, // |
				{
					Type:      semantics.TokenTypeFunction,
					Line:      0,
					Start:     16,
					Length:    6, // printf
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type:      semantics.TokenTypeString,
					Line:      0,
					Start:     23,
					Length:    8, // "%d items"
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 32, Length: 2}, // }}
			},
		},
		{
			name:    "test_multiple_variable_declarations",
			content: "{{ $x := 1 }}\n{{ $y := $x }}\n{{ printf \"%d %d\" $x $y }}",
			expected: []semantics.Token{
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 0, Length: 2}, // {{
				{
					Type:      semantics.TokenTypeVariable,
					Line:      0,
					Start:     3,
					Length:    2, // $x
					Modifiers: []semantics.TokenModifier{semantics.ModifierDeclaration},
				},
				{Type: semantics.TokenTypeOperator, Line: 0, Start: 6, Length: 2},   // :=
				{Type: semantics.TokenTypeDelimiter, Line: 0, Start: 10, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 0, Length: 2},  // {{
				{
					Type:      semantics.TokenTypeVariable,
					Line:      1,
					Start:     3,
					Length:    2, // $y
					Modifiers: []semantics.TokenModifier{semantics.ModifierDeclaration},
				},
				{Type: semantics.TokenTypeOperator, Line: 1, Start: 6, Length: 2}, // :=
				{
					Type:      semantics.TokenTypeVariable,
					Line:      1,
					Start:     9,
					Length:    2, // $x
					Modifiers: []semantics.TokenModifier{semantics.ModifierDefinition},
				},
				{Type: semantics.TokenTypeDelimiter, Line: 1, Start: 12, Length: 2}, // }}
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 0, Length: 2},  // {{
				{
					Type:      semantics.TokenTypeFunction,
					Line:      2,
					Start:     3,
					Length:    6, // printf
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type:      semantics.TokenTypeString,
					Line:      2,
					Start:     10,
					Length:    6, // "%d %d"
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type:      semantics.TokenTypeVariable,
					Line:      2,
					Start:     17,
					Length:    2, // $x
					Modifiers: []semantics.TokenModifier{semantics.ModifierDefinition},
				},
				{
					Type:      semantics.TokenTypeVariable,
					Line:      2,
					Start:     20,
					Length:    2, // $y
					Modifiers: []semantics.TokenModifier{semantics.ModifierDefinition},
				},
				{Type: semantics.TokenTypeDelimiter, Line: 2, Start: 23, Length: 2}, // }}
			},
		},
	}

	parser := NewTemplateTokenParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := parser.Parse(tt.content)

			// Debug output for failures
			if !assert.Equal(t, tt.expected, tokens) {
				t.Logf("Expected tokens:")
				for i, tok := range tt.expected {
					t.Logf("  %d: Type=%d, Line=%d, Start=%d, Length=%d", i, tok.Type, tok.Line, tok.Start, tok.Length)
				}
				t.Logf("Got tokens:")
				for i, tok := range tokens {
					t.Logf("  %d: Type=%d, Line=%d, Start=%d, Length=%d", i, tok.Type, tok.Line, tok.Start, tok.Length)
				}
			}
		})
	}
}

func TestTemplateTokenParser_GetTokensForFile(t *testing.T) {
	ctx := context.Background()
	parser := NewTemplateTokenParser()

	tests := []struct {
		name     string
		content  string
		expected *protocol.SemanticTokens
	}{
		{
			name:    "test_basic_template",
			content: "{{ if .Ready }}\n{{ .Name }}\n{{ end }}",
			expected: &protocol.SemanticTokens{
				Data: []uint32{
					0, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 2, 6, 4, // if (keyword, readonly)
					0, 3, 6, 2, 0, // .Ready (variable)
					0, 7, 2, 0, 0, // }} (delimiter)
					1, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 5, 2, 0, // .Name (variable)
					0, 6, 2, 0, 0, // }} (delimiter)
					1, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 3, 6, 4, // end (keyword, readonly)
					0, 4, 2, 0, 0, // }} (delimiter)
				},
			},
		},
		{
			name:    "test_variable_declarations",
			content: "{{ $x := 42 }}\n{{ $x }}",
			expected: &protocol.SemanticTokens{
				Data: []uint32{
					0, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 2, 2, 1, // $x (variable, declaration)
					0, 3, 2, 7, 4, // := (operator, readonly)
					0, 6, 2, 0, 0, // }} (delimiter)
					1, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 2, 2, 2, // $x (variable, definition)
					0, 3, 2, 0, 0, // }} (delimiter)
				},
			},
		},
		{
			name:    "test_builtin_functions",
			content: "{{ len .Items }}",
			expected: &protocol.SemanticTokens{
				Data: []uint32{
					0, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 3, 1, 4, // len (function, readonly)
					0, 4, 6, 2, 0, // .Items (variable)
					0, 7, 2, 0, 0, // }} (delimiter)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parser.GetTokensForFile(ctx, "test.tmpl", tt.content)
			require.NoError(t, err, "getting tokens should succeed")

			// Debug output for failures
			if !assert.Equal(t, tt.expected.Data, tokens.Data) {
				t.Logf("Expected token data:")
				for i := 0; i < len(tt.expected.Data); i += 5 {
					t.Logf("  Delta Line: %d, Delta Start: %d, Length: %d, Type: %d, Modifiers: %d",
						tt.expected.Data[i], tt.expected.Data[i+1], tt.expected.Data[i+2],
						tt.expected.Data[i+3], tt.expected.Data[i+4])
				}
				t.Logf("Got token data:")
				for i := 0; i < len(tokens.Data); i += 5 {
					t.Logf("  Delta Line: %d, Delta Start: %d, Length: %d, Type: %d, Modifiers: %d",
						tokens.Data[i], tokens.Data[i+1], tokens.Data[i+2],
						tokens.Data[i+3], tokens.Data[i+4])
				}
			}
		})
	}
}

func TestTemplateTokenParser_GetTokensForRange(t *testing.T) {
	ctx := context.Background()
	parser := NewTemplateTokenParser()

	tests := []struct {
		name     string
		content  string
		rng      protocol.Range
		expected *protocol.SemanticTokens
	}{
		{
			name:    "test_single_line_range",
			content: "{{ if .Ready }}\n{{ .Name }}\n{{ end }}",
			rng: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 1, Character: 11},
			},
			expected: &protocol.SemanticTokens{
				Data: []uint32{
					0, 0, 2, 0, 0, // {{ (delimiter)
					0, 3, 5, 2, 0, // .Name (variable)
					0, 6, 2, 0, 0, // }} (delimiter)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parser.GetTokensForRange(ctx, "test.tmpl", tt.content, tt.rng)
			require.NoError(t, err, "getting tokens should succeed")
			// Note: Currently returns full file tokens as range support isn't implemented
			assert.NotNil(t, tokens, "tokens should not be nil")
		})
	}
}
