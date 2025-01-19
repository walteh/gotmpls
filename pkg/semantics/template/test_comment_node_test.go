package template

import (
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Plan for CommentNode:
//
// ┌──────────────────────────────────┐
// │        CommentNode Tests         │
// │                                  │
// │ 1. Simple Comments              │
// │    {{/* comment */}}            │
// │                                  │
// │ 2. Multi-line Comments          │
// │    {{/* line 1                  │
// │         line 2 */}}             │
// │                                  │
// │ 3. Special Characters           │
// │    {{/* symbols !@# */}}        │
// │                                  │
// │ 4. Nested-looking Comments      │
// │    {{/* outer /* inner */ */}}  │
// │                                  │
// │ 5. Whitespace Handling          │
// │    {{-/* comment */-}}          │
// └──────────────────────────────────┘

// TestCommentNode_Simple tests basic comment parsing
func TestCommentNode_Simple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_empty_comment",
			input:    "{{/**/}}",
			expected: "/**/",
		},
		{
			name:     "test_simple_comment",
			input:    "{{/* hello */}}",
			expected: "/* hello */",
		},
		{
			name:     "test_spaced_comment",
			input:    "{{/*   spaced   */}}",
			expected: "/*   spaced   */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := participle.MustBuild[CommentNode](
				participle.Lexer(TemplateLexer),
				participle.Elide("whitespace"),
				participle.UseLookahead(2),
			)
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.String(), "parsed result should match expected")
		})
	}
}

// TestCommentNode_Multiline tests multi-line comment parsing
func TestCommentNode_Multiline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "test_two_lines",
			input: `{{/* line one
line two */}}`,
			expected: `/* line one
line two */`,
		},
		{
			name: "test_indented_lines",
			input: `{{/* first line
    indented line
        more indented */}}`,
			expected: `/* first line
    indented line
        more indented */`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := participle.MustBuild[CommentNode](
				participle.Lexer(TemplateLexer),
				participle.Elide("whitespace"),
				participle.UseLookahead(2),
			)
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.String(), "parsed result should match expected")
		})
	}
}

// TestCommentNode_SpecialChars tests comments with special characters
func TestCommentNode_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test_symbols",
			input:    "{{/* !@#$%^&*() */}}",
			expected: "/* !@#$%^&*() */",
		},
		{
			name:     "test_unicode",
			input:    "{{/* Hello, 世界! */}}",
			expected: "/* Hello, 世界! */",
		},
		{
			name:     "test_quotes",
			input:    `{{/* "quoted" 'text' */}}`,
			expected: `/* "quoted" 'text' */`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := participle.MustBuild[CommentNode](
				participle.Lexer(TemplateLexer),
				participle.Elide("whitespace"),
				participle.UseLookahead(2),
			)
			node, err := parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.NotNil(t, node, "node should not be nil")
			assert.Equal(t, tt.expected, node.String(), "parsed result should match expected")
		})
	}
}

// TestCommentNode_EdgeCases tests edge cases and error conditions
func TestCommentNode_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		expected    string
	}{
		{
			name:        "test_unclosed_comment",
			input:       "{{/* unclosed",
			shouldError: true,
		},
		{
			name:        "test_unopened_comment",
			input:       "comment */}}",
			shouldError: true,
		},
		{
			name:        "test_nested_comment_markers",
			input:       "{{/* outer /* inner */ */}}",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := participle.MustBuild[CommentNode](
				participle.Lexer(TemplateLexer),
				participle.Elide("whitespace"),
				participle.UseLookahead(2),
			)
			node, err := parser.ParseString("", tt.input)
			if tt.shouldError {
				require.Error(t, err, "parsing should fail")
			} else {
				require.NoError(t, err, "parsing should succeed")
				require.NotNil(t, node, "node should not be nil")
				assert.Equal(t, tt.expected, node.String(), "parsed result should match expected")
			}
		})
	}
}

// TODO: Add tests for whitespace trimming ({{- and -}})
// TODO: Add tests for comments containing template-like content
// TODO: Add tests for comments with escaped characters
