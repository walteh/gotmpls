package grammar_test

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/grammar"
)

func TestNewStore(t *testing.T) {
	// Setup logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	t.Run("test_store_creation", func(t *testing.T) {
		store, err := grammar.NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")
		require.NotNil(t, store, "store should not be nil")
	})

	t.Run("test_embedded_grammar_loading", func(t *testing.T) {
		store, err := grammar.NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")

		// Try to get a known embedded grammar (we expect at least go.tmLanguage.json to exist)
		gram, err := store.GetGrammar("source.go")
		require.NoError(t, err, "getting Go grammar should succeed")
		assert.Equal(t, "source.go", gram.ScopeName, "grammar should have correct scope name")
	})

	t.Run("test_custom_grammar_loading", func(t *testing.T) {
		store, err := grammar.NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")

		customGrammar := []byte(`{
			"scopeName": "source.custom",
			"name": "Custom",
			"patterns": [
				{
					"match": "test",
					"name": "keyword.custom"
				}
			]
		}`)

		err = store.LoadCustomGrammar(ctx, "source.custom", customGrammar)
		require.NoError(t, err, "loading custom grammar should succeed")

		gram, err := store.GetGrammar("source.custom")
		require.NoError(t, err, "getting custom grammar should succeed")
		assert.Equal(t, "source.custom", gram.ScopeName, "custom grammar should have correct scope name")
	})

	t.Run("test_nonexistent_grammar", func(t *testing.T) {
		store, err := grammar.NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")

		_, err = store.GetGrammar("nonexistent.grammar")
		require.Error(t, err, "getting nonexistent grammar should fail")
	})
}
