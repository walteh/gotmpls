package lsp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
)

func TestHandleTextDocumentHover(t *testing.T) {
	ctx := context.Background()

	t.Run("simple variable hover", func(t *testing.T) {
		server := lsp.NewServer(ctx)
		// Setup document with a simple type hint and field
		content := `{{- /*gotype: test.Person*/ -}}
{{ .Name }}`
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Create test files
		err := server.CreateTestFiles(map[string]string{
			"go.mod": "module test",
			"test.go": `package test
type Person struct {
	Name string
}`,
		})
		require.NoError(t, err, "creating test files should succeed")

		// Test hover over .Name
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, result, "hover result should not be nil")
		assert.Equal(t, "**Variable**: Person.Name\n**Type**: string", result.Contents.Value, "hover content should match")
		require.NotNil(t, result.Range, "hover range should not be nil")
		assert.Equal(t, 1, result.Range.Start.Line, "range should start on line 1")
		assert.Equal(t, 1, result.Range.End.Line, "range should end on line 1")
		assert.Equal(t, 3, result.Range.Start.Character, "range should start at the beginning of .Name")
		assert.Equal(t, 7, result.Range.End.Character, "range should end at the end of .Name")
	})

	t.Run("nested field hover", func(t *testing.T) {
		// Setup document with nested fields
		content := `{{- /*gotype: test.Person*/ -}}
{{ .Address.Street }}`
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Create test files
		err := server.CreateTestFiles(map[string]string{
			"go.mod": "module test",
			"test.go": `package test
type Person struct {
	Address Address
}
type Address struct {
	Street string
}`,
		})
		require.NoError(t, err, "creating test files should succeed")

		// Test hover over .Address.Street
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 12},
		})
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, result, "hover result should not be nil")
		assert.Equal(t, "**Variable**: Person.Address.Street\n**Type**: string", result.Contents.Value, "hover content should match")
		require.NotNil(t, result.Range, "hover range should not be nil")
		assert.Equal(t, 1, result.Range.Start.Line, "range should start on line 1")
		assert.Equal(t, 1, result.Range.End.Line, "range should end on line 1")
		assert.Equal(t, 11, result.Range.Start.Character, "range should start at the beginning of Street")
		assert.Equal(t, 17, result.Range.End.Character, "range should end at the end of Street")
	})

	t.Run("invalid type hint", func(t *testing.T) {
		// Setup document with invalid type hint
		content := `{{- /*gotype: test.InvalidType*/ -}}
{{ .Name }}`
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Create test files
		err := server.CreateTestFiles(map[string]string{
			"go.mod": "module test",
			"test.go": `package test
type Person struct {
	Name string
}`,
		})
		require.NoError(t, err, "creating test files should succeed")

		// Test hover over .Name with invalid type hint
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		assert.Nil(t, result, "hover should return nil for invalid type")
	})

	t.Run("invalid field path", func(t *testing.T) {
		// Setup document with invalid field
		content := `{{- /*gotype: test.Person*/ -}}
{{ .InvalidField }}`
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Create test files
		err := server.CreateTestFiles(map[string]string{
			"go.mod": "module test",
			"test.go": `package test
type Person struct {
	Name string
}`,
		})
		require.NoError(t, err, "creating test files should succeed")

		// Test hover over .InvalidField
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		assert.Nil(t, result, "hover should return nil for invalid field")
	})

	t.Run("empty document", func(t *testing.T) {
		// Setup empty document
		content := ""
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Test hover on empty document
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 0, Character: 0},
		})
		require.NoError(t, err, "hover request should succeed")
		assert.Nil(t, result, "hover should return nil for empty document")
	})

	t.Run("no type hints", func(t *testing.T) {
		// Setup document without type hints
		content := "{{ .Name }}"
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Test hover without type hints
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 0, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		assert.Nil(t, result, "hover should return nil when no type hints are present")
	})

	t.Run("hover outside variable", func(t *testing.T) {
		// Setup document
		content := `{{- /*gotype: test.Person*/ -}}
{{ .Name }} some text`
		uri := "file:///test.tmpl"
		server.DidOpen(ctx, &lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:     uri,
				Text:    content,
				Version: 1,
			},
		})

		// Create test files
		err := server.CreateTestFiles(map[string]string{
			"go.mod": "module test",
			"test.go": `package test
type Person struct {
	Name string
}`,
		})
		require.NoError(t, err, "creating test files should succeed")

		// Test hover outside variable
		result, err := server.Hover(ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 15},
		})
		require.NoError(t, err, "hover request should succeed")
		assert.Nil(t, result, "hover should return nil when hovering outside variable")
	})
}
