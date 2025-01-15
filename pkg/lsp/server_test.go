package lsp_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration/nvim"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	t.Run("server initialization", func(t *testing.T) {
		files := map[string]string{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"go.mod":    "module test",
			"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Name string
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		_, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		// The fact that setupNeovimTest succeeded means the server initialized correctly
		// and we were able to establish LSP communication
	})

	t.Run("server handles multiple files", func(t *testing.T) {
		files := map[string]string{
			"file1.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"file2.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Age }}",
			"go.mod":     "module test",
			"test.go": `
package test
import _ "embed"

//go:embed file1.tmpl
var File1Template string

//go:embed file2.tmpl
var File2Template string

type Person struct {
	Name string
	Age  int
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		// Test hover in first file
		file1 := runner.TmpFilePathOf("file1.tmpl")
		hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(file1, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```go-template\n.Name\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
		require.Equal(t, uint32(8), hoverResult.Range.End.Character, "range should end at the end of .Name")

		// Test hover in second file
		file2 := runner.TmpFilePathOf("file2.tmpl")
		hoverResult, err = runner.Hover(t, ctx, protocol.NewHoverParams(file2, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tAge int\n}\n```\n\n### Template Access\n```go-template\n.Age\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Age")
		require.Equal(t, uint32(7), hoverResult.Range.End.Character, "range should end at the end of .Age")
	})

	t.Run("server_handles_file_changes", func(t *testing.T) {
		files := map[string]string{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"go.mod":    "module test",
			"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string
type Person struct {
	Name string
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("test.tmpl")

		// Test initial hover
		hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```go-template\n.Name\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
		require.Equal(t, uint32(8), hoverResult.Range.End.Character, "range should end at the end of .Name")

		// Save current file before making changes
		err = runner.Command("w")
		require.NoError(t, err, "save should succeed")

		// Change the file content
		err = runner.Command("normal! ggdG")
		require.NoError(t, err, "delete content should succeed")
		err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Age }}")
		require.NoError(t, err, "insert content should succeed")

		// Save the changes
		err = runner.Command("w")
		require.NoError(t, err, "save should succeed")

		// Test hover after change
		hoverResult, err = runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.Nil(t, hoverResult, "hover should return nil for non-existent field")
	})

	t.Run("hover_should_show_method_signature", func(t *testing.T) {
		files := map[string]string{
			"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .GetName }}",
			"go.mod":    "module test",
			"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string
type Person struct {
	Name string
}
	func (p *Person) GetName() string {
		return p.Name
	}
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("test.tmpl")
		hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Method Information\n\n```go\nfunc (*Person) GetName() (string)\n```\n\n### Return Type\n```go\nstring\n```\n\n### Template Usage\n```go-template\n.GetName\n```", hoverResult.Contents.Value)
	})

	t.Run("server_verifies_hover_ranges", func(t *testing.T) {
		files := map[string]string{
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			"go.mod": "module test",
			"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Address struct {
		Street string
	}
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("test.tmpl")

		// Test hover over different parts of .Address.Street
		positions := []struct {
			character int
			name      string
			expected  bool
		}{
			{5, "before address", false},
			{12, "start of Address", true},
			{19, "middle of Street", true},
			{28, "after Street", false},
		}

		for _, pos := range positions {
			t.Run(pos.name, func(t *testing.T) {
				hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 2, Character: uint32(pos.character)}))
				require.NoError(t, err, "hover request should succeed")

				if pos.expected {
					require.NotNil(t, hoverResult, "hover result should not be nil")
					require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tAddress struct {\n\t\tStreet string\n\t}\n}\n```\n\n### Template Access\n```go-template\n.Address.Street\n```", hoverResult.Contents.Value)
					require.NotNil(t, hoverResult.Range, "hover range should not be nil")
					require.Equal(t, uint32(2), hoverResult.Range.Start.Line, "range should start on line 2")
					require.Equal(t, uint32(2), hoverResult.Range.End.Line, "range should end on line 2")
					require.Equal(t, uint32(12), hoverResult.Range.Start.Character, "range should start at the beginning of .Address.Street")
					require.Equal(t, uint32(27), hoverResult.Range.End.Character, "range should end at the end of .Address.Street")
				} else {
					require.Nil(t, hoverResult, "hover should return nil for positions outside variable")
				}
			})
		}
	})

	t.Run("server_handles_submodule", func(t *testing.T) {
		files := map[string]string{
			"subdir/test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
			"subdir/go.mod":    "module test",
			"subdir/test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string
type Person struct {
	Name string
}`,
		}
		si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("subdir/test.tmpl")
		hoverResult, err := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.NoError(t, err, "hover request should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```go-template\n.Name\n```", hoverResult.Contents.Value)
	})
}

func TestDiagnosticsAfterFileChange(t *testing.T) {
	files := map[string]string{
		"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
		"go.mod":    "module test",
		"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string
type Person struct {
	Name string
}`,
	}

	ctx := context.Background()

	si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
	require.NoError(t, err, "setup should succeed")

	testFile := runner.TmpFilePathOf("test.tmpl")

	// Open the file and ensure LSP is attached
	buf, err := runner.OpenFile(testFile)
	require.NoError(t, err, "opening file should succeed")

	err = runner.AttachLSP(buf)
	require.NoError(t, err, "attaching LSP should succeed")

	// First verify we get diagnostics for an invalid field
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .InvalidField }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	// Wait longer for diagnostics to be published
	time.Sleep(500 * time.Millisecond)

	// Verify we get diagnostics for the invalid field
	diags, err := runner.GetDiagnostics(t, testFile, 2*time.Second)
	require.NoError(t, err, "getting diagnostics should succeed")
	require.NotEmpty(t, diags, "should have diagnostics for invalid field")
	require.Contains(t, diags[0].Message, "InvalidField", "diagnostic should mention the invalid field")

	// Now change to a valid field
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Name }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	// Wait longer for diagnostics to be published
	time.Sleep(500 * time.Millisecond)

	// Verify diagnostics are cleared
	diags, err = runner.GetDiagnostics(t, testFile, 2*time.Second)
	require.NoError(t, err, "getting diagnostics should succeed")
	require.Empty(t, diags, "diagnostics should be cleared after fixing the error")

	// Make another change that introduces an error
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .AnotherInvalidField }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	// Wait longer for diagnostics to be published
	time.Sleep(500 * time.Millisecond)

	// Verify we get diagnostics for the new invalid field
	diags, err = runner.GetDiagnostics(t, testFile, 2*time.Second)
	require.NoError(t, err, "getting diagnostics should succeed")
	require.NotEmpty(t, diags, "should have diagnostics for new invalid field")
	require.Contains(t, diags[0].Message, "AnotherInvalidField", "diagnostic should mention the new invalid field")
}

func TestDiagnosticHarness(t *testing.T) {
	files := map[string]string{
		"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}",
		"go.mod":    "module test",
		"test.go": `
package test
import _ "embed"

//go:embed test.tmpl
var TestTemplate string
type Person struct {
	Name string
}`,
	}

	ctx := context.Background()
	si := lsp.NewServer(ctx).BuildServerInstance(ctx, nil)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
	require.NoError(t, err, "setup should succeed")

	testFile := runner.TmpFilePathOf("test.tmpl")

	// Open the file and ensure LSP is attached
	buf, err := runner.OpenFile(testFile)
	require.NoError(t, err, "opening file should succeed")

	err = runner.AttachLSP(buf)
	require.NoError(t, err, "attaching LSP should succeed")

	// Test Case 1: Valid template should have no diagnostics
	err = runner.CheckDiagnostics(t, testFile, []protocol.Diagnostic{}, 2*time.Second)
	require.NoError(t, err, "should have no diagnostics for valid template")

	// Test Case 2: Invalid field should show diagnostic
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .InvalidField }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	expectedDiag := []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 3},
				End:   protocol.Position{Line: 1, Character: 15},
			},
			Severity: protocol.SeverityError,
			Message:  "field InvalidField not found in type Person",
		},
	}
	err = runner.CheckDiagnostics(t, testFile, expectedDiag, 2*time.Second)
	require.NoError(t, err, "should show diagnostic for invalid field")

	// Test Case 3: Multiple diagnostics
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Field1 }}\n{{ .Field2 }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	expectedDiags := []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 3},
				End:   protocol.Position{Line: 1, Character: 9},
			},
			Severity: protocol.SeverityError,
			Message:  "field Field1 not found in type Person",
		},
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 3},
				End:   protocol.Position{Line: 2, Character: 9},
			},
			Severity: protocol.SeverityError,
			Message:  "field Field2 not found in type Person",
		},
	}
	err = runner.CheckDiagnostics(t, testFile, expectedDiags, 2*time.Second)
	require.NoError(t, err, "should show diagnostics for multiple invalid fields")

	// Test Case 4: Fixing errors should clear diagnostics
	err = runner.Command("normal! ggdG")
	require.NoError(t, err, "delete content should succeed")
	err = runner.Command("normal! i{{- /*gotype: test.Person*/ -}}\n{{ .Name }}")
	require.NoError(t, err, "insert content should succeed")
	err = runner.Command("w")
	require.NoError(t, err, "save should succeed")

	err = runner.CheckDiagnostics(t, testFile, []protocol.Diagnostic{}, 2*time.Second)
	require.NoError(t, err, "should have no diagnostics after fixing errors")
}
