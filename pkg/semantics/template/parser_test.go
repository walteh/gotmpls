package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/semantics"
)

// NewTemplateTokenParser creates a new template token parser
func NewTemplateTokenParser() *TemplateTokenParser {
	return &TemplateTokenParser{}
}

func TestParseSimpleText(t *testing.T) {
	t.Run("single_text_node", func(t *testing.T) {
		parser := NewTemplateTokenParser()
		tokens, err := parser.Parse("Hello World")
		require.NoError(t, err, "parsing should succeed")
		require.NotNil(t, tokens, "tokens should not be nil")
		require.Len(t, tokens, 1, "should have one token")

		assert.Equal(t, semantics.TokenTypeText, tokens[0].Type)
		assert.Equal(t, 0, tokens[0].Position.Offset)
		assert.Equal(t, "Hello World", tokens[0].Position.Text)
	})

	t.Run("text_followed_by_delim", func(t *testing.T) {
		parser := NewTemplateTokenParser()
		tokens, err := parser.Parse("Hello {{")
		require.NoError(t, err, "parsing should succeed")
		require.NotNil(t, tokens, "tokens should not be nil")
		require.Len(t, tokens, 2, "should have two tokens")

		assert.Equal(t, semantics.TokenTypeText, tokens[0].Type)
		assert.Equal(t, 0, tokens[0].Position.Offset)
		assert.Equal(t, "Hello ", tokens[0].Position.Text)

		assert.Equal(t, semantics.TokenTypeDelimiter, tokens[1].Type)
		assert.Equal(t, 6, tokens[1].Position.Offset)
		assert.Equal(t, "{{", tokens[1].Position.Text)
	})

	t.Run("complete_action", func(t *testing.T) {
		parser := NewTemplateTokenParser()
		tokens, err := parser.Parse("Hello {{.Name}}")
		require.NoError(t, err, "parsing should succeed")
		require.NotNil(t, tokens, "tokens should not be nil")
		require.Len(t, tokens, 4, "should have four tokens")

		assert.Equal(t, semantics.TokenTypeText, tokens[0].Type)
		assert.Equal(t, 0, tokens[0].Position.Offset)
		assert.Equal(t, "Hello ", tokens[0].Position.Text)

		assert.Equal(t, semantics.TokenTypeDelimiter, tokens[1].Type)
		assert.Equal(t, 6, tokens[1].Position.Offset)
		assert.Equal(t, "{{", tokens[1].Position.Text)

		assert.Equal(t, semantics.TokenTypeVariable, tokens[2].Type)
		assert.Equal(t, 8, tokens[2].Position.Offset)
		assert.Equal(t, ".Name", tokens[2].Position.Text)

		assert.Equal(t, semantics.TokenTypeDelimiter, tokens[3].Type)
		assert.Equal(t, 13, tokens[3].Position.Offset)
		assert.Equal(t, "}}", tokens[3].Position.Text)
	})
}
