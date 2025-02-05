package grammar

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	// Setup logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	t.Run("test_store_creation", func(t *testing.T) {
		store, err := NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")
		require.NotNil(t, store, "store should not be nil")
	})

	t.Run("test_embedded_grammar_loading", func(t *testing.T) {
		store, err := NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")

		// Try to get a known embedded grammar (we expect at least go.tmLanguage.json to exist)
		gram, err := store.GetGrammar("source.go")
		require.NoError(t, err, "getting Go grammar should succeed")
		assert.Equal(t, "source.go", gram.ScopeName, "grammar should have correct scope name")
	})

	t.Run("test_custom_grammar_loading", func(t *testing.T) {
		store, err := NewStore(ctx)
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
		store, err := NewStore(ctx)
		require.NoError(t, err, "store creation should succeed")

		_, err = store.GetGrammar("nonexistent.grammar")
		require.Error(t, err, "getting nonexistent grammar should fail")
	})
}

func TestGrammarUnmarshal(t *testing.T) {
	// Test case: basic grammar with required fields
	t.Run("test_basic_grammar", func(t *testing.T) {
		jsonData := []byte(`{
			"name": "Test Grammar",
			"scopeName": "source.test",
			"patterns": [
				{
					"match": "\\b(true|false)\\b",
					"name": "constant.language.boolean"
				}
			]
		}`)

		var grammar Grammar
		err := json.Unmarshal(jsonData, &grammar)
		require.NoError(t, err, "should unmarshal valid grammar")

		assert.Equal(t, "Test Grammar", grammar.Name)
		assert.Equal(t, "source.test", grammar.ScopeName)
		assert.Len(t, grammar.Patterns, 1)
		assert.Equal(t, "\\b(true|false)\\b", grammar.Patterns[0].Match)
		assert.Equal(t, "constant.language.boolean", grammar.Patterns[0].Name)
	})

	// Test case: grammar with repository
	t.Run("test_grammar_with_repository", func(t *testing.T) {
		jsonData := []byte(`{
			"scopeName": "source.test",
			"patterns": [],
			"repository": {
				"keywords": {
					"match": "\\b(if|else|while)\\b",
					"name": "keyword.control"
				}
			}
		}`)

		var grammar Grammar
		err := json.Unmarshal(jsonData, &grammar)
		require.NoError(t, err, "should unmarshal grammar with repository")

		assert.NotNil(t, grammar.Repository)
		assert.Contains(t, grammar.Repository, "keywords")
		assert.Equal(t, "\\b(if|else|while)\\b", grammar.Repository["keywords"].Match)
	})

	// Test case: grammar with complex captures
	t.Run("test_grammar_with_captures", func(t *testing.T) {
		jsonData := []byte(`{
			"scopeName": "source.test",
			"patterns": [
				{
					"begin": "\\(",
					"end": "\\)",
					"beginCaptures": {
						"0": {
							"name": "punctuation.paren.open",
							"patterns": [
								{
									"match": "\\(",
									"name": "meta.paren.open"
								}
							]
						}
					},
					"endCaptures": {
						"0": {
							"name": "punctuation.paren.close"
						}
					}
				}
			]
		}`)

		var grammar Grammar
		err := json.Unmarshal(jsonData, &grammar)
		require.NoError(t, err, "should unmarshal grammar with captures")

		assert.NotEmpty(t, grammar.Patterns[0].BeginCaptures["0"], "should have begin captures")
		assert.NotEmpty(t, grammar.Patterns[0].EndCaptures["0"], "should have end captures")
	})
}

func TestStore(t *testing.T) {
	// Test case: loading and retrieving grammars
	t.Run("test_store_load_and_get", func(t *testing.T) {
		ctx := context.Background()
		store, err := NewStore(ctx)
		require.NoError(t, err, "should create store")

		// Test loading a custom grammar
		customGrammar := []byte(`{
			"name": "Custom Grammar",
			"scopeName": "source.custom",
			"patterns": []
		}`)

		err = store.LoadCustomGrammar(ctx, "custom", customGrammar)
		require.NoError(t, err, "should load custom grammar")

		// Test retrieving the grammar
		grammar, err := store.GetGrammar("custom")
		require.NoError(t, err, "should get custom grammar")
		assert.Equal(t, "Custom Grammar", grammar.Name)
		assert.Equal(t, "source.custom", grammar.ScopeName)
	})

	// Test case: getting non-existent grammar
	t.Run("test_get_nonexistent_grammar", func(t *testing.T) {
		ctx := context.Background()
		store, err := NewStore(ctx)
		require.NoError(t, err, "should create store")

		_, err = store.GetGrammar("nonexistent")
		assert.Error(t, err, "should error on nonexistent grammar")
	})
}
