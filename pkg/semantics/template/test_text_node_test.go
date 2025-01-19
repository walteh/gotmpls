package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextNode_Simple tests basic text node parsing
func TestTextNode_Simple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_empty_text",
			input:    "",
			expected: "",
		},
		{
			name:     "test_simple_text",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "test_whitespace_only",
			input:    "   \t\n  ",
			expected: "   \t\n  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a parser just for this test
			parser, err := BuildWithOptionsFunc[TextNode]()
			require.NoError(t, err, "building parser should succeed")
			// Parse the input
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.Content, "content should match")
		})
	}
}

// TestTextNode_SpecialChars tests text nodes with special characters
func TestTextNode_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_unicode",
			input:    "Hello, 世界!",
			expected: "Hello, 世界!",
		},
		{
			name:     "test_symbols",
			input:    "!@#$%^&*()",
			expected: "!@#$%^&*()",
		},
		{
			name:     "test_escaped_chars",
			input:    "\\n\\t\\r",
			expected: "\\n\\t\\r",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[TextNode]()
			require.NoError(t, err, "building parser should succeed")
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.Content, "content should match")
		})
	}
}

// TestTextNode_Delimiters tests text nodes with template delimiters
func TestTextNode_Delimiters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_single_brace",
			input:    "text { more text",
			expected: "text { more text",
		},
		{
			name:     "test_escaped_delim",
			input:    "text \\{{ more text",
			expected: "text \\{{ more text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[TextNode]()
			require.NoError(t, err, "building parser should succeed")
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.Content, "content should match")
		})
	}
}

// TestTextNode_Multiline tests text nodes with multiple lines
func TestTextNode_Multiline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "test_two_lines",
			input: `line one
line two`,
			expected: `line one
line two`,
		},
		{
			name: "test_mixed_whitespace",
			input: `  line one
	line two
    line three`,
			expected: `  line one
	line two
    line three`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := BuildWithOptionsFunc[TextNode]()
			require.NoError(t, err, "building parser should succeed")
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.Content, "content should match")
		})
	}
}
