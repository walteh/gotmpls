package nvim_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration/nvim"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

func TestNeovimBasic(t *testing.T) {
	ctx := context.Background()

	files := map[string]string{
		"main.go": `package main

type Person struct {
	Name string
}

func main() {
	p := Person{Name: "test"}
	_ = p
}`,
		"go.mod": "module test",
	}

	si, err := protocol.NewGoplsServerInstance(ctx)
	require.NoError(t, err)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err)

	// Test hover at Person struct
	mainFile := runner.TmpFilePathOf("main.go")
	hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(mainFile, protocol.Position{Line: 7, Character: 8}))
	require.NoError(t, err)
	assert.NotNil(t, hoverResult)
	assert.Contains(t, hoverResult.Contents.Value, "type Person struct")

	// Save and quit
	output, err := runner.SaveAndQuitWithOutput()
	require.NoError(t, err)
	require.NotEmpty(t, output)
}

func TestEditMethods(t *testing.T) {
	ctx := context.Background()

	files := map[string]string{
		"main.go": `package main

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{Name: "test"}
	_ = p
}`,
		"go.mod": "module test",
	}

	si, err := protocol.NewGoplsServerInstance(ctx)
	require.NoError(t, err)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err)

	mainFile := runner.TmpFilePathOf("main.go")

	// Open the file and ensure LSP is attached
	buf, err := runner.OpenFile(mainFile)
	require.NoError(t, err, "opening file should succeed")

	err = runner.AttachLSP(buf)
	require.NoError(t, err, "attaching LSP should succeed")

	// Wait for initial LSP setup
	time.Sleep(1 * time.Second)

	// Test 1: Edit with save should trigger diagnostics
	err = runner.ApplyEditWithSave(t, mainFile, `package main

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{InvalidField: "test"}  // This should cause an error
	_ = p
}`)
	require.NoError(t, err, "applying edit with save should succeed")

	expectedDiag := []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 8, Character: 12},
				End:   protocol.Position{Line: 8, Character: 23},
			},
			Severity: protocol.SeverityError,
			Message:  "unknown field InvalidField in struct literal",
		},
	}
	err = runner.CheckDiagnostics(t, mainFile, expectedDiag, 5*time.Second)
	require.NoError(t, err, "should get diagnostics after save")

	// Test 2: Edit without save should also trigger diagnostics
	err = runner.ApplyEditWithoutSave(t, mainFile, `package main

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{AnotherInvalidField: 42}  // This should cause an error
	_ = p
}`)
	require.NoError(t, err, "applying edit without save should succeed")

	expectedDiag = []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 8, Character: 12},
				End:   protocol.Position{Line: 8, Character: 30},
			},
			Severity: protocol.SeverityError,
			Message:  "unknown field AnotherInvalidField in struct literal",
		},
	}
	err = runner.CheckDiagnostics(t, mainFile, expectedDiag, 5*time.Second)
	require.NoError(t, err, "should get diagnostics without save")

	// Test 3: Edit with save should persist after reopening
	validContent := `package main

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{Age: 42}  // This should be valid
	_ = p
}`
	err = runner.ApplyEditWithSave(t, mainFile, validContent)
	require.NoError(t, err, "applying valid edit should succeed")

	// Verify content persists and has no diagnostics
	err = runner.CheckDiagnostics(t, mainFile, []protocol.Diagnostic{}, 5*time.Second)
	require.NoError(t, err, "should have no diagnostics for valid content")

	// Test hover on the valid field
	hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(mainFile, protocol.Position{Line: 8, Character: 12}))
	require.NoError(t, err, "hover should succeed")
	require.NotNil(t, hoverResult, "hover result should not be nil")
	require.Contains(t, hoverResult.Contents.Value, "Age int", "hover should show Age field type")
}
