package semtok_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/semtok"
)

/*
Test Organization:
----------------
Each test group focuses on a specific token type or feature:

    +----------------+
    |  Test Groups   |
    +----------------+
           |
    +------+-------+
    |              |
 Simple         Complex
 Tokens         Tokens
    |              |
  Single      Multiple
  Token       Tokens
    |              |
Variables    Templates
Functions    Pipelines
Keywords     Actions
Comments

We test each token type in isolation first, then in combination.
*/

func TestVariableTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []semtok.Token
		wantErr  bool
	}{
		{
			name:  "test_simple_variable",
			input: "{{ .Name }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Name", 3),
				},
			},
		},
		// TODO(@semtok): Add more variable test cases
		// - Multiple variables
		// - Nested variables
		// - Variables with modifiers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := semtok.GetTokensForText(context.Background(), []byte(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected error for test case")
				return
			}
			require.NoError(t, err, "unexpected error getting tokens")
			assert.Equal(t, tt.expected, tokens, "tokens should match expected")
		})
	}
}

func TestFunctionTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []semtok.Token
		wantErr  bool
	}{
		{
			name:  "test_simple_function",
			input: "{{ printf .Name }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("printf", 3),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Name", 10),
				},
			},
		},
		// TODO(@semtok): Add more function test cases
		// - Built-in functions
		// - Custom functions
		// - Function chains
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := semtok.GetTokensForText(context.Background(), []byte(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected error for test case")
				return
			}
			require.NoError(t, err, "unexpected error getting tokens")
			assert.Equal(t, tt.expected, tokens, "tokens should match expected")
		})
	}
}

func TestKeywordTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []semtok.Token
		wantErr  bool
	}{
		{
			name:  "test_if_keyword",
			input: "{{ if .Ready }}ready{{end}}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("if", 3),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 6),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 20),
				},
			},
		},
		// TODO(@semtok): Add more keyword test cases
		// - range keyword
		// - with keyword
		// - template keyword
		// - define keyword
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := semtok.GetTokensForText(context.Background(), []byte(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected error for test case")
				return
			}
			require.NoError(t, err, "unexpected error getting tokens")
			assert.Equal(t, tt.expected, tokens, "tokens should match expected")
		})
	}
}

/*
Complex Test Cases:
-----------------
The following tests combine multiple token types to ensure
they work together correctly:

    Template         Expected Tokens
    --------         ---------------
    {{ if .X }}   -> [keyword, variable]
       ^^^  ^
       |    |
       |    +-- variable token
       +-- keyword token
*/

func TestComplexTemplates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []semtok.Token
		wantErr  bool
	}{
		{
			name: "test_nested_if_with_function",
			input: `{{ if .Ready }}
	{{ printf "Ready: %v" .Status }}
{{ end }}`,
			expected: []semtok.Token{
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("if", 3),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 6),
				},
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("printf", 17),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(`"Ready: %v"`, 24),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Status", 35),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 45),
				},
			},
		},
		// TODO(@semtok): Add more complex test cases
		// - Nested templates
		// - Multiple pipelines
		// - Mixed token types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := semtok.GetTokensForText(context.Background(), []byte(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected error for test case")
				return
			}
			require.NoError(t, err, "unexpected error getting tokens")
			assert.Equal(t, tt.expected, tokens, "tokens should match expected")
		})
	}
}