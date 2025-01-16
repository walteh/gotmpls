package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/semantics"
)

func TestParseDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []semantics.Token
	}{
		{
			name:  "basic_delimiters",
			input: "{{ .Name }}",
			expected: []semantics.Token{
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 0,
						Text:   "{{",
					},
				},
				{
					Type: semantics.TokenTypeVariable,
					Position: position.RawPosition{
						Offset: 3,
						Text:   ".Name",
					},
				},
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 9,
						Text:   "}}",
					},
				},
			},
		},
		{
			name:  "trimmed_delimiters",
			input: "{{- .Name -}}",
			expected: []semantics.Token{
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 0,
						Text:   "{{-",
					},
				},
				{
					Type: semantics.TokenTypeVariable,
					Position: position.RawPosition{
						Offset: 4,
						Text:   ".Name",
					},
				},
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 10,
						Text:   "-}}",
					},
				},
			},
		},
	}

	parser := NewTemplateTokenParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First verify that Participle can parse the input
			ast, err := Parser.ParseString("", tt.input)
			require.NoError(t, err, "Participle should parse the input")
			require.NotNil(t, ast, "AST should not be nil")
			require.Len(t, ast.Nodes, 1, "Should have exactly one node")
			require.NotNil(t, ast.Nodes[0].Action, "Node should be an action")

			// Then verify token generation
			tokens := parser.Parse(tt.input)
			require.NotNil(t, tokens, "Tokens should not be nil")
			assert.Equal(t, tt.expected, tokens, "Tokens should match expected")
		})
	}
}

func TestParseKeywordsAndOperators(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []semantics.Token
	}{
		{
			name:    "if_eq_condition",
			content: "{{ if eq .Age 18 }}",
			expected: []semantics.Token{
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 0,
						Text:   "{{",
					},
				},
				{
					Type: semantics.TokenTypeKeyword,
					Position: position.RawPosition{
						Offset: 3,
						Text:   "if",
					},
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type: semantics.TokenTypeOperator,
					Position: position.RawPosition{
						Offset: 6,
						Text:   "eq",
					},
				},
				{
					Type: semantics.TokenTypeVariable,
					Position: position.RawPosition{
						Offset: 9,
						Text:   ".Age",
					},
				},
				{
					Type: semantics.TokenTypeNumber,
					Position: position.RawPosition{
						Offset: 14,
						Text:   "18",
					},
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 17,
						Text:   "}}",
					},
				},
			},
		},
		{
			name:    "range_with_pipe",
			content: "{{ range .Items | len }}",
			expected: []semantics.Token{
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 0,
						Text:   "{{",
					},
				},
				{
					Type: semantics.TokenTypeKeyword,
					Position: position.RawPosition{
						Offset: 3,
						Text:   "range",
					},
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				},
				{
					Type: semantics.TokenTypeVariable,
					Position: position.RawPosition{
						Offset: 9,
						Text:   ".Items",
					},
				},
				{
					Type: semantics.TokenTypeOperator,
					Position: position.RawPosition{
						Offset: 16,
						Text:   "|",
					},
				},
				{
					Type: semantics.TokenTypeFunction,
					Position: position.RawPosition{
						Offset: 18,
						Text:   "len",
					},
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly, semantics.ModifierDefaultLibrary},
				},
				{
					Type: semantics.TokenTypeDelimiter,
					Position: position.RawPosition{
						Offset: 22,
						Text:   "}}",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &TemplateTokenParser{}
			tokens := parser.Parse(tt.content)
			assert.Equal(t, tt.expected, tokens, "Tokens should match expected")
		})
	}
}
