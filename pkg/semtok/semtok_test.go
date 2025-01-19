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
					Range:    position.NewBasicPosition(".Name", 2),
				},
			},
		},
		{
			name:  "test_multiple_variables",
			input: "{{ .First }} {{ .Second }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".First", 2),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(" ", 11),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Second", 15),
				},
			},
		},
		{
			name:  "test_nested_variable",
			input: "{{ .User.Name }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".User.Name", 2),
				},
			},
		},
		{
			name:  "test_variable_with_len",
			input: "{{ len .Items }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("len", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Items", 6),
				},
			},
		},
		{
			name:  "test_variable_with_index",
			input: "{{ index .Array 0 }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("index", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Array", 8),
				},
				{
					Type:     semtok.TokenNumber,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("0", 15),
				},
			},
		},
		{
			name:  "test_variable_with_chained_modifiers",
			input: "{{ len (index .Array 0) }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("len", 2),
				},
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("index", 7),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Array", 13),
				},
				{
					Type:     semtok.TokenNumber,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("0", 20),
				},
			},
		},
		// TODO(@semtok): Add more variable test cases
		// - Variables with more complex modifiers
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
					Range:    position.NewBasicPosition("printf", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Name", 9),
				},
			},
		},
		{
			name:  "test_builtin_function_len",
			input: "{{ len .Items }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("len", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Items", 6),
				},
			},
		},
		{
			name:  "test_custom_function",
			input: "{{ myFunc .Value }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("myFunc", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Value", 9),
				},
			},
		},
		{
			name:  "test_function_chain",
			input: "{{ .Name | upper  | printf }}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Name", 2),
				},
				{
					Type:     semtok.TokenOperator,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("|", 8),
				},
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("upper", 10),
				},
				{
					Type:     semtok.TokenOperator,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("|", 17),
				},
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("printf", 19),
				},
			},
		},
		{
			name:  "test_function_multiple_args",
			input: `{{ printf "%s-%s" .First .Last }}`,
			expected: []semtok.Token{
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("printf", 2),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(`"%s-%s"`, 9),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".First", 17),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Last", 24),
				},
			},
		},
		// TODO(@semtok): Add more function test cases
		// - Functions with complex arguments
		// - Functions with nested function calls
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
					Range:    position.NewBasicPosition("if", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 5),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("ready", 14),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 21),
				},
			},
		},
		{
			name:  "test_if_else_comment",
			input: "{{ if .Ready }}ready{{else if .Readz }}{{/* readz */}}yo{{end}}",
			expected: []semtok.Token{

				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("if", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 5),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("ready", 14),
				},

				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("else if", 21),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Readz", 29),
				},

				{
					Type:     semtok.TokenComment,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("/* readz */", 40),
				},

				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("yo", 53),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 57),
				},
			},
		},
		{
			name:  "test_if_else_comment_with_space_in_else_if",
			input: "{{ if .Ready }}ready{{else  if .Readz }}{{/* readz */}}yo{{end}}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("if", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 5),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("ready", 14),
				},

				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("else  if", 21),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Readz", 30),
				},

				{
					Type:     semtok.TokenComment,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("/* readz */", 41),
				},

				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("yo", 54),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 58),
				},
			},
		},
		{
			name:  "test_range_keyword",
			input: "{{ range .Items }}{{.}}{{end}}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("range", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Items", 8),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".", 19),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 24),
				},
			},
		},
		{
			name:  "test_with_keyword",
			input: "{{ with .User }}{{.Name}}{{end}}",
			expected: []semtok.Token{
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("with", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".User", 7),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Name", 17),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 26),
				},
			},
		},
		// TODO(@semtok): Add more keyword test cases
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
					Range:    position.NewBasicPosition("if", 2),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Ready", 5),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("\n\t", 14),
				},
				{
					Type:     semtok.TokenFunction,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("printf", 19),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(`"Ready: %v"`, 26),
				},
				{
					Type:     semtok.TokenVariable,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition(".Status", 38),
				},
				{
					Type:     semtok.TokenString,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("\n", 48),
				},
				{
					Type:     semtok.TokenKeyword,
					Modifier: semtok.ModifierNone,
					Range:    position.NewBasicPosition("end", 52),
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
