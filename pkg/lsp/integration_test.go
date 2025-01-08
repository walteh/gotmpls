package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
)

func setupTestWorkspace(t *testing.T) string {
	dir, err := os.MkdirTemp("", "lsp-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	// Create a test template file
	tmplContent := `{{- /*gotype: example.com/test/types.Person */ -}}
{{- define "header" -}}
# Person Information
{{- end -}}

{{template "header"}}

Name: {{.Name}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}
`
	err = os.WriteFile(filepath.Join(dir, "test.tmpl"), []byte(tmplContent), 0644)
	require.NoError(t, err)

	// Create a Go module
	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module example.com/test

go 1.21

`), 0644)
	require.NoError(t, err)

	// Create types package
	typesDir := filepath.Join(dir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	// Create types.go file
	typesContent := `package types

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street string
	City   string
}
`
	err = os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(typesContent), 0644)
	require.NoError(t, err)

	return dir
}

func uriFromPath(path string) string {
	return "file://" + path
}

func TestIntegration_BasicLSPFlow(t *testing.T) {
	// Create a test workspace
	workspaceDir := setupTestWorkspace(t)

	// Create a mock read-write connection
	rwc := &mockRWC{
		readBuf:  bytes.NewBuffer(nil),
		writeBuf: bytes.NewBuffer(nil),
	}

	// Create a server with real components
	server := NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	// Start the server in a goroutine
	go server.Start(context.Background(), rwc, rwc)

	t.Run("initialize", func(t *testing.T) {
		// Send initialize request
		id := int64(1)
		rwc.writeMessage(t, "initialize", &id, InitializeParams{
			RootURI: uriFromPath(workspaceDir),
		})

		// Wait for initialize response
		_, respID, result, err := rwc.readMessage(t)
		require.NoError(t, err)
		require.Equal(t, id, *respID)

		// Verify capabilities
		var initResult InitializeResult
		resultBytes, err := json.Marshal(result)
		require.NoError(t, err)
		err = json.Unmarshal(resultBytes, &initResult)
		require.NoError(t, err)
		require.True(t, initResult.Capabilities.HoverProvider)
		require.NotNil(t, initResult.Capabilities.TextDocumentSync)
		require.Equal(t, 1, initResult.Capabilities.TextDocumentSync.Change)
	})

	// Send initialized notification
	rwc.writeMessage(t, "initialized", nil, nil)

	t.Run("textDocument/didOpen", func(t *testing.T) {
		// Send didOpen notification
		rwc.writeMessage(t, "textDocument/didOpen", nil, DidOpenTextDocumentParams{
			TextDocument: TextDocumentItem{
				URI:     uriFromPath(filepath.Join(workspaceDir, "test.tmpl")),
				Version: 1,
				Text: `{{- /*gotype: example.com/test/types.Person */ -}}
{{- define "header" -}}
# Person Information
{{- end -}}

{{template "header"}}

Name: {{.Name}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}
`,
			},
		})

		// Wait for publishDiagnostics notification
		method, _, result, err := rwc.readMessage(t)
		require.NoError(t, err)
		require.Equal(t, "textDocument/publishDiagnostics", method)

		// Verify diagnostics
		var diagParams PublishDiagnosticsParams
		resultBytes, err := json.Marshal(result)
		require.NoError(t, err)
		err = json.Unmarshal(resultBytes, &diagParams)
		require.NoError(t, err)
		require.Equal(t, uriFromPath(filepath.Join(workspaceDir, "test.tmpl")), diagParams.URI)
		require.Len(t, diagParams.Diagnostics, 1)
		require.Equal(t, "no go.mod found in directory", diagParams.Diagnostics[0].Message)
	})
}
