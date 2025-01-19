package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Plan for Command:
//
// ┌──────────────────────────────────┐
// │        Command Tests            │
// │                                  │
// │ 1. Term Commands               │
// │    - String literals           │
// │    - Number literals           │
// │    - Boolean literals          │
// │    - Nil literal              │
// │    - Variables                 │
// │    - Fields                    │
// │                                  │
// │ 2. Function Calls              │
// │    - No args                   │
// │    - Single arg               │
// │    - Multiple args            │
// │                                  │
// │ 3. Method Calls               │
// │    - Simple methods           │
// │    - Chained methods          │
// │    - Methods with args        │
// └──────────────────────────────────┘

// TestCommand_Terms tests term-based commands
func TestCommand_Terms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// String literals
		{
			name:     "test_string_literal",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},
		{
			name:     "test_string_with_escapes",
			input:    `"hello \"world\""`,
			expected: `"hello \"world\""`,
		},
		// Number literals
		{
			name:     "test_integer",
			input:    "42",
			expected: "42",
		},
		{
			name:     "test_float",
			input:    "3.14",
			expected: "3.14",
		},
		// Boolean literals
		{
			name:     "test_true",
			input:    "true",
			expected: "true",
		},
		{
			name:     "test_false",
			input:    "false",
			expected: "false",
		},
		// Nil literal
		{
			name:     "test_nil",
			input:    "nil",
			expected: "nil",
		},
		// Variables
		{
			name:     "test_variable",
			input:    "$x",
			expected: "$x",
		},
		// Fields
		{
			name:     "test_field",
			input:    ".Name",
			expected: ".Name",
		},
		{
			name:     "test_nested_field",
			input:    ".User.Name",
			expected: ".User.Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Term]()
			require.NoError(t, err, "building parser should succeed")

			term, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, term, "term should not be nil")
			assert.Equal(t, tt.expected, term.String(), "parsed result should match expected")
		})
	}
}

// TestCommand_FunctionCalls tests function call commands
func TestCommand_FunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_no_args",
			input:    "now",
			expected: "now",
		},
		{
			name:     "test_single_arg",
			input:    `print "hello"`,
			expected: `print "hello"`,
		},
		{
			name:     "test_multiple_args",
			input:    "add 1 2",
			expected: "add 1 2",
		},
		{
			name:     "test_complex_args",
			input:    `printf "%s=%d" "count" 42`,
			expected: `printf "%s=%d" "count" 42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Call]()
			require.NoError(t, err, "building parser should succeed")

			call, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, call, "call should not be nil")
			assert.Equal(t, tt.expected, call.String(), "parsed result should match expected")
		})
	}
}

// TestCommand_FieldAccess tests field access commands
func TestCommand_FieldAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_method",
			input:    ".User.GetName",
			expected: ".User.GetName",
		},
		{
			name:     "test_method_with_args",
			input:    `.User.GetField "name"`,
			expected: `.User.GetField "name"`,
		},
		{
			name:     "test_chained_methods",
			input:    ".User.GetItems.First",
			expected: ".User.GetItems.First",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Field]()
			require.NoError(t, err, "building parser should succeed")

			field, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, field, "field should not be nil")
			assert.Equal(t, tt.expected, field.String(), "parsed result should match expected")
		})
	}
}

// TestCommand_EdgeCases tests edge cases and error conditions
func TestCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		expected    string
	}{
		{
			name:        "test_empty_command",
			input:       "",
			shouldError: true,
		},
		{
			name:        "test_invalid_number",
			input:       "3.",
			shouldError: true,
		},
		{
			name:        "test_unclosed_string",
			input:       `"unclosed`,
			shouldError: true,
		},
		{
			name:        "test_invalid_field",
			input:       ".",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[CommandNode]()
			require.NoError(t, err, "building parser should succeed")

			cmd, err := parser.ParseString("", tt.input)
			if tt.shouldError {
				require.Error(t, err, "parsing should fail")
				return
			}
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, cmd, "command should not be nil")
			assert.Equal(t, tt.expected, cmd.String(), "parsed result should match expected")
		})
	}
}

// TODO: Add tests for whitespace handling in commands
// TODO: Add tests for precedence between terms and calls
// TODO: Add tests for complex argument expressions
