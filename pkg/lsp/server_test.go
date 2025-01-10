package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	t.Run("server initialization", func(t *testing.T) {
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
		require.NoError(t, err)
		defer setup.cleanup()

		// The fact that setupNeovimTest succeeded means the server initialized correctly
		// and we were able to establish LSP communication
	})

	t.Run("server handles multiple files", func(t *testing.T) {
		files := testFiles{
			"file1.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"file2.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Age }}",
			"go.mod":     "module test",
			"test.go": `
package test
type Person struct {
	Name string
	Age  int
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err)
		defer setup.cleanup()

		// Test hover in first file
		file1 := filepath.Join(setup.tmpDir, "file1.tmpl")
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + file1},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Name\n**Type**: string", hoverResult.Contents.Value)

		// Test hover in second file
		file2 := filepath.Join(setup.tmpDir, "file2.tmpl")
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + file2},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Age\n**Type**: int", hoverResult.Contents.Value)
	})

	t.Run("server handles file changes", func(t *testing.T) {
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
		require.NoError(t, err)
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test initial hover
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Name\n**Type**: string", hoverResult.Contents.Value)

		// Save current file before making changes
		err = setup.nvimInstance.Command("w")
		require.NoError(t, err)

		// Change the file content
		err = setup.nvimInstance.Command("normal! ggdG")
		require.NoError(t, err)
		err = setup.nvimInstance.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Age }}")
		require.NoError(t, err)

		// Save the changes
		err = setup.nvimInstance.Command("w")
		require.NoError(t, err)

		// Test hover after change
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.Nil(t, hoverResult, "hover should return nil for non-existent field")
	})
}
