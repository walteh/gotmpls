package nvim_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration/nvim"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

func TestNeovimBasic(t *testing.T) {
	ctx := context.Background()

	files := map[string]string{
		"test.tmpl": "{{- /*gotype: test.Items*/ -}}\n{{ .Value }}",
		"go.mod":    "module test",
		"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var Template string

type Items struct {
	Value string
}`,
	}

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files)
	require.NoError(t, err)

	// // Open test file and set up LSP
	testFile := runner.TmpFilePathOf("test.tmpl")

	// Test hover at the start of the file
	hoverResult, err := runner.RequestHover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 6}))
	// if err != nil && strings.Contains(err.Error(), "client failed to attach") {
	// 	output, errd := runner.SaveAndQuitWithOutput()
	// 	require.NoError(t, errd)
	// 	t.Logf("Output: %s", output)
	// 	require.NoError(t, err)
	// }
	require.NoError(t, err)
	require.NotNil(t, hoverResult)
	// Save and quit
	output, err := runner.SaveAndQuitWithOutput()
	require.NoError(t, err)

	// Verify the output
	require.NotEmpty(t, output, "neovim output should not be empty")

	require.Equal(t, "### Type Information\n\n```go\ntype Items struct {\n\tValue string\n}\n```\n\n### Template Access\n```go-template\n.Value\n```", hoverResult.Contents.Value)

}
