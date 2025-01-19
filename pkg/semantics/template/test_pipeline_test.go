package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Plan for Pipeline:
//
// ┌──────────────────────────────────┐
// │        Pipeline Tests           │
// │                                  │
// │ 1. Variable Declaration         │
// │    $x := value                  │
// │    $x, $y := val1, val2        │
// │                                  │
// │ 2. Command Chaining            │
// │    cmd1 | cmd2                  │
// │    cmd1 | cmd2 | cmd3          │
// │                                  │
// │ 3. Mixed Operations            │
// │    $x := cmd1 | cmd2           │
// │    $x, $y := cmd1 | cmd2       │
// │                                  │
// │ 4. Parentheses                 │
// │    (cmd1) | cmd2               │
// │    cmd1 | (cmd2 | cmd3)        │
// └──────────────────────────────────┘

// TestPipeline_Variables tests variable declarations in pipelines
func TestPipeline_Variables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_single_var",
			input:    "$x := .Name",
			expected: "$x := .Name",
		},
		{
			name:     "test_multi_var",
			input:    "$x, $y := .First, .Last",
			expected: "$x, $y := .First, .Last",
		},
		{
			name:     "test_var_with_expr",
			input:    "$result := add 1 2",
			expected: "$result := add 1 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Pipeline]()
			require.NoError(t, err, "building parser should succeed")

			pipeline, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestPipeline_Chaining tests command chaining in pipelines
func TestPipeline_Chaining(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_pipe",
			input:    ".Name | upper",
			expected: ".Name | upper",
		},
		{
			name:     "test_multi_pipe",
			input:    ".Name | lower | title",
			expected: ".Name | lower | title",
		},
		{
			name:     "test_pipe_with_args",
			input:    ".Items | index 0 | upper",
			expected: ".Items | index 0 | upper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Pipeline]()
			require.NoError(t, err, "building parser should succeed")

			pipeline, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestPipeline_Mixed tests mixed operations in pipelines
func TestPipeline_Mixed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_var_and_pipe",
			input:    "$x := .Name | upper",
			expected: "$x := .Name | upper",
		},
		{
			name:     "test_multi_var_and_pipe",
			input:    "$x, $y := .Items | split \",\"",
			expected: "$x, $y := .Items | split \",\"",
		},
		{
			name:     "test_complex_mixed",
			input:    "$result := .Items | filter .Active | count",
			expected: "$result := .Items | filter .Active | count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Pipeline]()
			require.NoError(t, err, "building parser should succeed")

			pipeline, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestPipeline_Parentheses tests parentheses grouping in pipelines
func TestPipeline_Parentheses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_parens",
			input:    "(.Name | upper) | quote",
			expected: "(.Name | upper) | quote",
		},
		{
			name:     "test_nested_parens",
			input:    ".Items | (index 0 | upper)",
			expected: ".Items | (index 0 | upper)",
		},
		{
			name:     "test_complex_parens",
			input:    "(.Users | filter .Active) | (count | gt 0)",
			expected: "(.Users | filter .Active) | (count | gt 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Pipeline]()
			require.NoError(t, err, "building parser should succeed")

			pipeline, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestPipeline_EdgeCases tests edge cases and error conditions
func TestPipeline_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		expected    string
	}{
		{
			name:        "test_empty_pipeline",
			input:       "",
			shouldError: true,
		},
		{
			name:        "test_lone_pipe",
			input:       "|",
			shouldError: true,
		},
		{
			name:        "test_unmatched_paren",
			input:       "(.Name | upper",
			shouldError: true,
		},
		{
			name:        "test_invalid_var_decl",
			input:       "$x, := .Name",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Pipeline]()
			require.NoError(t, err, "building parser should succeed")

			pipeline, err := parser.ParseString("", tt.input)
			if tt.shouldError {
				require.Error(t, err, "parsing should fail")
			} else {
				require.NoError(t, err, "parsing should succeed")
				require.NotNil(t, pipeline, "pipeline should not be nil")
				assert.Equal(t, tt.expected, pipeline.String(), "parsed result should match expected")
			}
		})
	}
}

// TestPipeline_SimpleFunctionCall tests the most basic function call parsing
func TestPipeline_SimpleFunctionCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_func",
			input:    "len .Items",
			expected: "len .Items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[CommandNode]()
			require.NoError(t, err, "building parser should succeed")

			cmd, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, cmd, "command should not be nil")
			require.NotNil(t, cmd.Call, "call should not be nil")

			// Debug output
			t.Logf("Command: %#v", cmd)
			t.Logf("Call: %#v", cmd.Call)

			assert.Equal(t, tt.expected, cmd.String(), "parsed result should match expected")
		})
	}
}

// TODO: Add tests for whitespace handling in pipelines
// TODO: Add tests for complex nested expressions
// TODO: Add tests for operator precedence
