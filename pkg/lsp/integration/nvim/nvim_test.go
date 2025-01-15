package nvim_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration/nvim"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

func TestHoverBasic(t *testing.T) {
	// Initialize test files
	files := map[string]string{
		"main.go": `package main

type Person struct {
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Name
}
`,
	}
	ctx := context.Background()
	si, err := protocol.NewGoplsServerInstance(ctx)
	require.NoError(t, err, "failed to create gopls server instance")

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err, "failed to create test runner")

	uri := runner.TmpFilePathOf("main.go")

	hoverp := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 7, Character: 21}, // Position at GetName
		},
	}
	// Test hover functionality
	hover, err := runner.Hover(t, ctx, hoverp)
	require.NoError(t, err, "hover request should succeed")
	assert.NotNil(t, hover, "hover response should not be nil")

	// {"range":{"end":{"line":7,"character":24},"start":{"line":7,"character":17}},"contents":{"kind":"markdown","value":"```go\nfunc (p *Person) GetName() string\n```\n\n---\n\n[`(main.Person).GetName` on pkg.go.dev](https:\/\/pkg.go.dev\/command-line-arguments\/private\/var\/folders\/8j\/scdcg3yx02dc5pdf9g6188dm0000gn\/T\/nvim-test-1205161790\/main.go#Person.GetName)"}}
	hoverw := &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: fmt.Sprintf("```go\nfunc (p *Person) GetName() string\n```\n\n---\n\n[`(main.Person).GetName` on pkg.go.dev](https://pkg.go.dev/command-line-arguments/private%s#Person.GetName)", string(uri.Path())),
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 7, Character: 17},
			End:   protocol.Position{Line: 7, Character: 24},
		},
	}

	assert.Equal(t, hoverw, hover, "hover response should match")
	require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
}

func TestDiagnosticsBasic(t *testing.T) {
	// Initialize test files
	files := map[string]string{
		"main.go": `package main

type Person struct {
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Invalid
}
`,
	}

	si, err := protocol.NewGoplsServerInstance(context.Background())
	require.NoError(t, err, "failed to create gopls server instance")

	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err, "failed to create test runner")

	// Test hover functionality
	uri := runner.TmpFilePathOf("main.go")

	diags, err := runner.GetDiagnostics(t, uri, 5*time.Second)
	require.NoError(t, err, "hover request should succeed")
	assert.NotNil(t, diags, "hover response should not be nil")

	diagsw := &protocol.FullDocumentDiagnosticReport{
		Kind: "",
		Items: []protocol.Diagnostic{
			{
				Message: "p.Invalid undefined (type *Person has no field or method Invalid)",
				Range: protocol.Range{
					Start: protocol.Position{Line: 8, Character: 10},
					End:   protocol.Position{Line: 8, Character: 17},
				},
				Severity: protocol.SeverityError,
				Source:   "compiler",
				Code:     "MissingFieldOrMethod",
				CodeDescription: &protocol.CodeDescription{
					Href: "https://pkg.go.dev/golang.org/x/tools/internal/typesinternal#MissingFieldOrMethod",
				},
			},
		},
	}

	assert.Equal(t, diagsw, diags, "diagnostics should match")
	require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
}

func TestEditMethods(t *testing.T) {
	// Initialize test files
	files := map[string]string{
		"main.go": `package main

type Person struct {
	Name string
	Age  int
}
`,
	}
	si, err := protocol.NewGoplsServerInstance(context.Background())
	require.NoError(t, err, "failed to create gopls server instance")

	// Create server instance with mock config
	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err, "failed to create test runner")

	uri := runner.TmpFilePathOf("main.go")

	// Test applying edit with save
	newContent := `package main

type Person struct {
	Name    string
	Age     int
	Address string // Added field
}
`
	err = runner.ApplyEdit(t, uri, newContent, true)
	require.NoError(t, err, "applying edit should succeed")

	// Verify content was updated
	content, err := runner.GetDocumentText(t, uri)
	require.NoError(t, err, "getting document text should succeed")
	assert.Equal(t, newContent, content, "document content should match")

	// Test diagnostics after edit
	diags, err := runner.GetDiagnostics(t, uri, 5*time.Second)
	require.NoError(t, err, "getting diagnostics should succeed")
	assert.Empty(t, diags, "should have no diagnostics for valid file")

	// Test formatting
	formatted, err := runner.GetFormattedDocument(t, context.Background(), uri)
	require.NoError(t, err, "formatting document should succeed")
	assert.NotEmpty(t, formatted, "formatted content should not be empty")

	// Clean up
	require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
}
