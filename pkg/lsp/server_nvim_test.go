package lsp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/lsp"
	"github.com/walteh/gotmpls/pkg/lsp/integration/nvim"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
)

// Helper to open a document in the server
func OpenDocument(ctx context.Context, t *testing.T, server *lsp.Server, uri protocol.DocumentURI, content string) error {
	t.Helper()
	return server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "gotmpl",
			Version:    1,
			Text:       content,
		},
	})
}

func TestNvimServer(t *testing.T) {
	t.Parallel()

	t.Run("server_initialization", func(t *testing.T) {
		t.Parallel()

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
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)
		_, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		// The fact that setupNeovimTest succeeded means the server initialized correctly
		// and we were able to establish LSP communication
	})

	t.Run("server_handles_multiple_files", func(t *testing.T) {
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
		t.Parallel()
		ctx := context.Background()
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		// Test hover in first file
		file1 := runner.TmpFilePathOf("file1.tmpl")
		hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(file1, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```gotmpl\n.Name\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
		require.Equal(t, uint32(8), hoverResult.Range.End.Character, "range should end at the end of .Name")

		// Test hover in second file
		file2 := runner.TmpFilePathOf("file2.tmpl")
		hoverResult, rpcs = runner.Hover(t, ctx, protocol.NewHoverParams(file2, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tAge int\n}\n```\n\n### Template Access\n```gotmpl\n.Age\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Age")
		require.Equal(t, uint32(7), hoverResult.Range.End.Character, "range should end at the end of .Age")
	})

	t.Run("mock_server_handles_multiple_files", func(t *testing.T) {
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
		t.Parallel()

		ctx := context.Background()
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		// Test hover in first file
		file1 := runner.TmpFilePathOf("file1.tmpl")
		hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(file1, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```gotmpl\n.Name\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
		require.Equal(t, uint32(8), hoverResult.Range.End.Character, "range should end at the end of .Name")

		// Test hover in second file
		file2 := runner.TmpFilePathOf("file2.tmpl")
		hoverResult, rpcs = runner.Hover(t, ctx, protocol.NewHoverParams(file2, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tAge int\n}\n```\n\n### Template Access\n```gotmpl\n.Age\n```", hoverResult.Contents.Value)
		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
		require.Equal(t, uint32(1), hoverResult.Range.Start.Line, "range should start on line 1")
		require.Equal(t, uint32(1), hoverResult.Range.End.Line, "range should end on line 1")
		require.Equal(t, uint32(3), hoverResult.Range.Start.Character, "range should start at the beginning of .Age")
		require.Equal(t, uint32(7), hoverResult.Range.End.Character, "range should end at the end of .Age")
	})

	t.Run("server_handles_file_changes", func(t *testing.T) {
		t.Parallel()

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
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("test.tmpl")

		hoverw := &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```gotmpl\n.Name\n```",
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 3},
				End:   protocol.Position{Line: 1, Character: 8},
			},
		}

		// Test initial hover
		hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.Equal(t, hoverw, hoverResult, "hover result should match expected")

		rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .Age }}", true)
		require.Len(t, rpcs, 1, "should have 1 rpcs")

		// Test hover after change
		hoverResult, rpcs = runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.Nil(t, hoverResult, "hover should return nil for non-existent field")
	})

	t.Run("hover_should_show_method_signature", func(t *testing.T) {
		t.Parallel()

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
		ctx := context.Background()
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("test.tmpl")
		hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Method Information\n\n```go\nfunc (*Person) GetName() (string)\n```\n\n### Return Type\n```go\nstring\n```\n\n### Template Usage\n```gotmpl\n.GetName\n```", hoverResult.Contents.Value)
	})

	t.Run("server_verifies_hover_ranges", func(t *testing.T) {
		t.Parallel()

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
		ctx := context.Background()
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

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
				hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 2, Character: uint32(pos.character)}))
				require.Len(t, rpcs, 2, "should have 2 rpcs")

				if pos.expected {
					require.NotNil(t, hoverResult, "hover result should not be nil")
					require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tAddress struct {\n\t\tStreet string\n\t}\n}\n```\n\n### Template Access\n```gotmpl\n.Address.Street\n```", hoverResult.Contents.Value)
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
		t.Parallel()

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
		ctx := context.Background()
		ser := lsp.NewServer(ctx)
		si := protocol.NewServerInstance(ctx, ser, nil)

		runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
		require.NoError(t, err, "setup should succeed")

		testFile := runner.TmpFilePathOf("subdir/test.tmpl")
		hoverResult, rpcs := runner.Hover(t, ctx, protocol.NewHoverParams(testFile, protocol.Position{Line: 1, Character: 3}))
		require.Len(t, rpcs, 2, "should have 2 rpcs")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```gotmpl\n.Name\n```", hoverResult.Contents.Value)
	})
}

func TestNvimDiagnosticsAfterFileChange(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"test.tmpl": "{{- /*gotype: test.Person*/ -}}\n{{ .Namex }}",
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
	ser := lsp.NewServer(ctx)
	si := protocol.NewServerInstance(ctx, ser, nil)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
	require.NoError(t, err, "setup should succeed")

	testFile := runner.TmpFilePathOf("test.tmpl")

	// Verify we get diagnostics for the invalid field
	diags, rpcs := runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.NotEmpty(t, diags, "should have diagnostics for invalid field")
	require.Contains(t, diags[0].Message, "field not found", "diagnostic should mention the invalid field")

	rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}", true)
	require.Len(t, rpcs, 1, "should have 1 rpcs")

	// Verify diagnostics are cleared
	diags, rpcs = runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.Empty(t, diags, "diagnostics should be cleared after fixing the error")

	// Make another change that introduces an error
	rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .AnotherInvalidField }}", true)
	require.Len(t, rpcs, 1, "should have 1 rpcs")

	// Verify we get diagnostics for the new invalid field
	diags, rpcs = runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.NotEmpty(t, diags, "should have diagnostics for new invalid field")
	require.Contains(t, diags[0].Message, "AnotherInvalidField", "diagnostic should mention the new invalid field")
}

func TestNvimDiagnosticHarness(t *testing.T) {

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
	t.Parallel()

	ctx := context.Background()
	ser := lsp.NewServer(ctx)
	si := protocol.NewServerInstance(ctx, ser, nil)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
	require.NoError(t, err, "setup should succeed")

	testFile := runner.TmpFilePathOf("test.tmpl")

	// Test Case 1: Valid template should have no diagnostics
	diags, rpcs := runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.Empty(t, diags, "diagnostics should be nil for valid template")

	// Test Case 2: Invalid field should show diagnostic
	rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .InvalidField }}", true)
	require.Len(t, rpcs, 1, "should have 1 rpcs")

	expectedDiag := []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 3},
				End:   protocol.Position{Line: 1, Character: 16},
			},
			Severity: protocol.SeverityError,
			Message:  "field not found [ InvalidField ] in type [ Person ]",
			Code:     "",
		},
	}
	diags, rpcs = runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	for _, rpc := range rpcs {
		if rpc.Response != nil {
			t.Logf("rpc: %+v", rpc.Response.ResultString())
		}
	}

	require.Len(t, rpcs, 2, "should hav	e 2 rpcs")
	require.Len(t, diags, len(expectedDiag), "should have 1 diagnostic")
	require.ElementsMatch(t, expectedDiag, diags, "diagnostics should match expected")

	rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .Field1 }}\n{{ .Field2 }}", true)
	require.Len(t, rpcs, 1, "should have 1 rpcs")

	expectedDiag = []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 3},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			Severity: protocol.SeverityError,
			Message:  "field not found [ Field1 ] in type [ Person ]",
			Code:     "",
			Source:   "",
		},
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 3},
				End:   protocol.Position{Line: 2, Character: 10},
			},
			Severity: protocol.SeverityError,
			Message:  "field not found [ Field2 ] in type [ Person ]",
			Code:     "",
			Source:   "",
		},
	}
	diags, rpcs = runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.ElementsMatch(t, expectedDiag, diags, "diagnostics should match expected")

	rpcs = runner.ApplyEdit(t, testFile, "{{- /*gotype: test.Person*/ -}}\n{{ .Name }}", true)
	require.Len(t, rpcs, 1, "should have 1 rpcs")

	diags, rpcs = runner.GetDiagnostics(t, testFile, protocol.SeverityError)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	require.Empty(t, diags, "diagnostics should be cleared after fixing errors")
}

func TestNvimSemanticTokens(t *testing.T) {
	t.Parallel()

	// t.Skip("skipping semantic tokens test")

	files := map[string]string{
		"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ if eq .Name "test" }}
	{{ printf "Hello, %s" .Name }}
{{ end }}`,
		"go.mod": "module test",
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
	ser := lsp.NewServer(ctx)
	si := protocol.NewServerInstance(ctx, ser, nil)

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoTemplateConfig{})
	require.NoError(t, err, "setup should succeed")

	testFile := runner.TmpFilePathOf("test.tmpl")

	// Request semantic tokens for the entire file
	tokens, _ := runner.GetSemanticTokensFull(t, ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testFile},
	})
	require.NotNil(t, tokens, "semantic tokens should not be nil")

	// Verify we have the expected number of tokens
	// The template should have tokens for:
	// - delimiters ({{, }}, {{-, -}})
	// - keywords (if, end)
	// - operators (eq)
	// - variables (.Name)
	// - functions (printf)
	// - strings ("test", "Hello, %s")
	require.NotEmpty(t, tokens.Data, "should have semantic tokens")

	// Request semantic tokens for a specific range
	rangeTokens, _ := runner.GetSemanticTokensRange(t, ctx, &protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testFile},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 0},
			End:   protocol.Position{Line: 1, Character: 25},
		},
	})
	require.NotNil(t, rangeTokens, "semantic tokens for range should not be nil")
	require.NotEmpty(t, rangeTokens.Data, "should have semantic tokens for range")

	// 	// Test semantic tokens after file modification
	// 	_ = runner.ApplyEdit(t, testFile, `{{- /*gotype: test.Person*/ -}}
	// {{ if and (eq .Name "test") (gt .Age 18) }}
	// 	{{ printf "Adult: %s" .Name }}
	// {{ end }}`, true)

	// // Request tokens for modified file
	//
	//	newTokens, _ := runner.GetSemanticTokensFull(t, ctx, &protocol.SemanticTokensParams{
	//		TextDocument: protocol.TextDocumentIdentifier{URI: testFile},
	//	})
	//
	// require.NotNil(t, newTokens, "semantic tokens after modification should not be nil")
	// require.NotEmpty(t, newTokens.Data, "should have semantic tokens after modification")
}
