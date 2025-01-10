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

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	t.Run("basic LSP flow", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{- define "header" -}}
# Person Information
{{- end -}}

{{template "header"}}

Name: {{.Name}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}`,
			"go.mod": "module test",
			"test.go": `
package test

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street string
	City   string
}`,
		}

		setup, err := setupNeovimTest(t, server, files)
		require.NoError(t, err)
		defer setup.cleanup()

		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

		// Test hover over .Name
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 7, Character: 8},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Name\n**Type**: string", hoverResult.Contents.Value)

		// Test hover over .Age
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 8, Character: 7},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Age\n**Type**: int", hoverResult.Contents.Value)

		// Test hover over nested field .Address.Street
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 10, Character: 12},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Address.Street\n**Type**: string", hoverResult.Contents.Value)

		// Test hover over nested field .Address.City
		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 11, Character: 10},
		})
		require.NoError(t, err)
		require.NotNil(t, hoverResult)
		require.Equal(t, "**Variable**: Person.Address.City\n**Type**: string", hoverResult.Contents.Value)
	})

	t.Run("missing go.mod", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ .Name }}`,
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

		// Test hover over .Name - should fail because go.mod is missing
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.Nil(t, hoverResult, "hover should return nil when go.mod is missing")
	})

	t.Run("invalid go.mod", func(t *testing.T) {
		files := testFiles{
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ .Name }}`,
			"go.mod": "invalid go.mod content",
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

		// Test hover over .Name - should fail because go.mod is invalid
		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
			Position:     lsp.Position{Line: 1, Character: 5},
		})
		require.NoError(t, err)
		require.Nil(t, hoverResult, "hover should return nil when go.mod is invalid")
	})
}
