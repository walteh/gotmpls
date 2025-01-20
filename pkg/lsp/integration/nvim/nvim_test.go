package nvim_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/lsp/integration/nvim"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
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
	hover, rpcs := runner.Hover(t, ctx, hoverp)
	require.Len(t, rpcs, 2, "should have 2 rpcs")
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
	// t.Skip() // it stopped working and its not worth debugging right now
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

	time.Sleep(1 * time.Second)

	diags, _ := runner.GetDiagnostics(t, uri, protocol.SeverityError)
	assert.NotNil(t, diags, "diag response should not be nil")

	diagsw := &protocol.FullDocumentDiagnosticReport{
		Kind: "full",
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

	assert.Equal(t, diagsw.Items, diags, "diagnostics should match")
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
	_ = runner.ApplyEdit(t, uri, newContent, true)
	// Verify content was updated
	content, err := runner.GetDocumentText(t, uri)
	require.NoError(t, err, "getting document text should succeed")
	assert.Equal(t, newContent, content, "document content should match")

	// Test diagnostics after edit
	diags, rpcs := runner.GetDiagnostics(t, uri, protocol.SeverityError)
	assert.Len(t, diags, 0, "should have no diagnostics for valid file")
	require.Len(t, rpcs, 2, "should have 2 rpcs")
	// Test formatting

	// Clean up
	require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
}

func TestHoverComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		position protocol.Position
		want     *protocol.Hover
	}{
		{
			name: "hover_over_type",
			files: map[string]string{
				"main.go": `package main

type Person struct {
	Name string
	Age  int
}
`,
			},
			position: protocol.Position{
				Line:      2,
				Character: 6,
			},
			want: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: "```go\ntype Person struct { // size=24 (0x18)\n\tName string\n\tAge  int\n}\n```\n\n---\n\n[`main.Person` on pkg.go.dev](https://pkg.go.dev/command-line-arguments/private{{.TEMP_FILE_NAME}}#Person)",
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 5},
					End:   protocol.Position{Line: 2, Character: 11},
				},
			},
		},
		{
			name: "hover_over_field",
			files: map[string]string{
				"main.go": `package main

type Person struct {
	Name string
	Age  int
}
`,
			},
			position: protocol.Position{
				Line:      3,
				Character: 2,
			},
			want: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: "```go\nfield Name string // size=16 (0x10), offset=0\n```\n\n---\n\n[`(main.Person).Name` on pkg.go.dev](https://pkg.go.dev/command-line-arguments/private{{.TEMP_FILE_NAME}}#Person.Name)",
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 3, Character: 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			si, err := protocol.NewGoplsServerInstance(ctx)
			require.NoError(t, err, "failed to create gopls server instance")

			runner, err := nvim.NewNvimIntegrationTestRunner(t, tt.files, si, &nvim.GoplsConfig{})
			require.NoError(t, err, "failed to create test runner")
			defer func() {
				require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
			}()

			uri := runner.TmpFilePathOf("main.go")
			// runner.OpenFile(uri)
			// runner.WaitForLSP()

			hoverp := &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     tt.position,
				},
			}

			hover, rpcs := runner.Hover(t, ctx, hoverp)
			require.Len(t, rpcs, 2, "should have 2 rpcs")

			tt.want.Contents.Value = strings.ReplaceAll(tt.want.Contents.Value, "{{.TEMP_FILE_NAME}}", uri.Path())

			assert.Equal(t, hover, tt.want, "hover should match")
		})
	}
}

func TestSemanticTokensBasic(t *testing.T) {
	t.Skip()
	// Initialize test files with a more complex Go file that should have semantic tokens
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"strings"
)

// Person represents a human being
type Person struct {
	Name    string
	Age     int
	Address string
}

// GetName returns the person's name
func (p *Person) GetName() string {
	return strings.TrimSpace(p.Name)
}

// GetFormattedInfo returns a formatted string with person's info
func (p *Person) GetFormattedInfo() string {
	return fmt.Sprintf("Name: %s, Age: %d", p.GetName(), p.Age)
}

func main() {
	person := &Person{
		Name: "John Doe",
		Age:  30,
	}
	fmt.Println(person.GetFormattedInfo())
}
`,
	}
	ctx := context.Background()
	si, err := protocol.NewGoplsServerInstance(ctx)
	require.NoError(t, err, "failed to create gopls server instance")

	// Configure gopls with semantic tokens enabled
	runner, err := nvim.NewNvimIntegrationTestRunner(t, files, si, &nvim.GoplsConfig{})
	require.NoError(t, err, "failed to create test runner")

	uri := runner.TmpFilePathOf("main.go")

	// First check that the file is loaded and has no diagnostics
	diags, _ := runner.GetDiagnostics(t, uri, protocol.SeverityError)
	require.Empty(t, diags, "file should have no diagnostics")

	t.Logf("🔍 Triggering semantic tokens refresh for buffer %s", "yo")
	// Test full document semantic tokens
	tokens, _ := runner.GetSemanticTokensFull(t, ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NotNil(t, tokens, "semantic tokens should not be nil")
	require.NotEmpty(t, tokens.Data, "should have semantic tokens")

	// Test range semantic tokens (focusing on the GetFormattedInfo method)
	rangeTokens, _ := runner.GetSemanticTokensRange(t, ctx, &protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 19, Character: 0},
			End:   protocol.Position{Line: 22, Character: 0},
		},
	})
	require.NotNil(t, rangeTokens, "semantic tokens for range should not be nil")
	require.NotEmpty(t, rangeTokens.Data, "should have semantic tokens for range")

	// Test semantic tokens after modification
	newContent := `package main

import (
	"fmt"
	"strings"
)

// Person represents a human being
type Person struct {
	Name     string
	Age      int
	Address  string
	Email    string
}

// GetName returns the person's name
func (p *Person) GetName() string {
	return strings.TrimSpace(p.Name)
}

// GetFormattedInfo returns a formatted string with person's info
func (p *Person) GetFormattedInfo() string {
	return fmt.Sprintf("Name: %s, Age: %d, Email: %s", 
		p.GetName(), 
		p.Age,
		p.Email,
	)
}

func main() {
	person := &Person{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}
	fmt.Println(person.GetFormattedInfo())
}
`
	rpcs := runner.ApplyEdit(t, uri, newContent, true)
	require.Len(t, rpcs, 1, "should have 1 rpc")

	// Get tokens for modified file
	newTokens, _ := runner.GetSemanticTokensFull(t, ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NotNil(t, newTokens, "semantic tokens after modification should not be nil")
	require.NotEmpty(t, newTokens.Data, "should have semantic tokens after modification")

	require.NoError(t, runner.SaveAndQuit(), "cleanup should succeed")
}
