package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

func TestHover(t *testing.T) {
	ctx := context.Background()

	server := lsp.NewServer(
		parser.Parse,
		ast.NewDefaultPackageAnalyzer(),
		true,
	)

	t.Run("simple variable hover", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"go.mod":    "module test",
			"test.go": `
package test
type Person struct {
	Name string
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over .Name
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "**Variable**: Person.Name\n**Type**: string", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, 1, hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, 1, hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, 3, hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
		require.Equal(t, 8, hoverResult.Range.End.Character, "range should end at the end of .Name")
	})

	t.Run("nested field hover", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Address.Street }}",
			"go.mod":    "module test",
			"test.go": `
package test
type Person struct {
	Address Address
}
type Address struct {
	Street string
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over .Address.Street
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 12},
		})
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "**Variable**: Person.Address.Street\n**Type**: string", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, 1, hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, 1, hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, 3, hoverResult.Range.Start.Character, "range should start at the beginning of .Address.Street")
		require.Equal(t, 17, hoverResult.Range.End.Character, "range should end at the end of .Address.Street")
	})

	t.Run("invalid type hint", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": "{{- /*gotype: test.InvalidType*/ -}}\n{{ .Name }}",
			"go.mod":    "module test",
			"test.go": `
package test
type Person struct {
	Name string
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over .Name with invalid type hint
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.Nil(t, hoverResult, "hover should return nil for invalid type")
	})

	t.Run("invalid field path", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .InvalidField }}",
			"go.mod":    "module test",
			"test.go": `
package test
type Person struct {
	Name string
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over .InvalidField
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.Nil(t, hoverResult, "hover should return nil for invalid field")
	})
}
