package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Plan for ActionNode:
//
// ┌──────────────────────────────────┐
// │        ActionNode Tests          │
// │                                  │
// │ 1. Simple Field Access          │
// │    .Name, .User.Name            │
// │                                  │
// │ 2. Variable Operations          │
// │    $x := value                  │
// │    $x                           │
// │                                  │
// │ 3. Function Calls               │
// │    len $arr                     │
// │    index $arr 0                 │
// │                                  │
// │ 4. Method Calls                 │
// │    .User.GetName                │
// │    .Slice.Index 0               │
// │                                  │
// │ 5. Pipelines                    │
// │    .Name | upper                │
// │    .List | first | upper        │
// └──────────────────────────────────┘

// TestActionNode_FieldAccess tests simple field access in actions
func TestActionNode_FieldAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_field",
			input:    "{{.Name}}",
			expected: ".Name",
		},
		{
			name:     "test_nested_field",
			input:    "{{.User.Name}}",
			expected: ".User.Name",
		},
		{
			name:     "test_very_nested_field",
			input:    "{{.User.Profile.DisplayName}}",
			expected: ".User.Profile.DisplayName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestActionNode_Variables tests variable declarations and usage
func TestActionNode_Variables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_var_declaration",
			input:    "{{$x := .Name}}",
			expected: "$x := .Name",
		},
		{
			name:     "test_multi_var_declaration",
			input:    "{{$x, $y := .First, .Last}}",
			expected: "$x, $y := .First, .Last",
		},
		{
			name:     "test_var_usage",
			input:    "{{$x}}",
			expected: "$x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestActionNode_FunctionCalls tests function call commands
func TestActionNode_FunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_func",
			input:    "{{len .Items}}",
			expected: "len .Items",
		},
		{
			name:     "test_func_multiple_args",
			input:    "{{index .Items 0}}",
			expected: "index .Items 0",
		},
		{
			name:     "test_nested_func",
			input:    "{{len .User.Items}}",
			expected: "len .User.Items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestActionNode_MethodCalls tests method calls in actions
func TestActionNode_MethodCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_method",
			input:    "{{.User.GetName}}",
			expected: ".User.GetName",
		},
		{
			name:     "test_method_with_args",
			input:    "{{.Slice.Index 0}}",
			expected: ".Slice.Index 0",
		},
		{
			name:     "test_chained_methods",
			input:    "{{.User.GetProfile.GetName}}",
			expected: ".User.GetProfile.GetName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestActionNode_Pipelines tests pipeline operations in actions
func TestActionNode_Pipelines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_simple_pipe",
			input:    "{{.Name | upper}}",
			expected: ".Name | upper",
		},
		{
			name:     "test_multi_pipe",
			input:    "{{.List | first | upper}}",
			expected: ".List | first | upper",
		},
		{
			name:     "test_pipe_with_args",
			input:    "{{.Items | index 0 | upper}}",
			expected: ".Items | index 0 | upper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TestActionNode_EdgeCases tests edge cases and error conditions
func TestActionNode_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		expected    string
	}{
		{
			name:        "test_empty_action",
			input:       "{{}}",
			shouldError: true,
		},
		{
			name:        "test_unclosed_action",
			input:       "{{.Name",
			shouldError: true,
		},
		{
			name:        "test_invalid_field",
			input:       "{{.}}",
			shouldError: true,
		},
		{
			name:        "test_invalid_pipe",
			input:       "{{.Name | }}",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[Template]()
			require.NoError(t, err, "building parser should succeed")

			node, err := parser.ParseString("", tt.input)
			if tt.shouldError {
				require.Error(t, err, "parsing should fail")
				return
			}
			require.NoError(t, err, "parser should build successfully")
			require.Len(t, node.Nodes, 1, "should have exactly one node")
			actionNode, ok := node.Nodes[0].(*ActionNode)
			require.True(t, ok, "node should be an ActionNode")
			require.NotNil(t, actionNode.Pipeline, "pipeline should not be nil")
			assert.Equal(t, tt.expected, actionNode.Pipeline.String(), "parsed result should match expected")
		})
	}
}

// TODO: Add tests for whitespace handling ({{- and -}})
// TODO: Add tests for more complex combinations of features
// TODO: Add tests for precedence and grouping with parentheses
