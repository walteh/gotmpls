package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	server := lsp.NewServer(
		ast.NewDefaultPackageAnalyzer(),
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
		require.NoError(t, err, "setup should succeed")
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
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		// Test hover in first file
		file1 := filepath.Join(setup.tmpDir, "file1.tmpl")
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + file1},
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

		// Test hover in second file
		file2 := filepath.Join(setup.tmpDir, "file2.tmpl")
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + file2},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "**Variable**: Person.Age\n**Type**: int", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, 1, hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, 1, hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, 3, hoverResult.Range.Start.Character, "range should start at the beginning of .Age")
		require.Equal(t, 7, hoverResult.Range.End.Character, "range should end at the end of .Age")
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
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test initial hover
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

		// Save current file before making changes
		err = setup.nvimInstance.Command("w")
		require.NoError(t, err, "save should succeed")

		// Change the file content
		err = setup.nvimInstance.Command("normal! ggdG")
		require.NoError(t, err, "delete content should succeed")
		err = setup.nvimInstance.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Age }}")
		require.NoError(t, err, "insert content should succeed")

		// Save the changes
		err = setup.nvimInstance.Command("w")
		require.NoError(t, err, "save should succeed")

		// Test hover after change
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 3},
		})
		require.NoError(t, err, "hover request should succeed")
		require.Nil(t, hoverResult, "hover should return nil for non-existent field")
	})

	t.Run("server verifies hover ranges", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			"go.mod": "module test",
			"test.go": `
package test
type Person struct {
	Address struct {
		Street string
	}
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err, "setup should succeed")
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over different parts of .Address.Street
		positions := []struct {
			character int
			name      string
			expected  bool
		}{
			{5, "before address", false},
			{12, "start of Address", true},
			{19, "middle of Street", true},
			{25, "after Street", false},
		}

		for _, pos := range positions {
			t.Run(pos.name, func(t *testing.T) {
				hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
					TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
					Position:     lsp.Position{Line: 2, Character: pos.character},
				})
				require.NoError(t, err, "hover request should succeed")

				if pos.expected {
					require.NotNil(t, hoverResult, "hover result should not be nil")
					require.Equal(t, "**Variable**: Person.Address.Street\n**Type**: string", hoverResult.Contents.Value)
					require.NotNil(t, hoverResult.Range, "hover range should not be nil")
					require.Equal(t, 2, hoverResult.Range.Start.Line, "range should start on line 2")
					require.Equal(t, 2, hoverResult.Range.End.Line, "range should end on line 2")
					require.Equal(t, 12, hoverResult.Range.Start.Character, "range should start at the beginning of .Address.Street")
					require.Equal(t, 26, hoverResult.Range.End.Character, "range should end at the end of .Address.Street")
				} else {
					require.Nil(t, hoverResult, "hover should return nil for positions outside variable")
				}
			})
		}
	})
}
