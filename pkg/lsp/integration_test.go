package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
)

func setupTestWorkspace(t *testing.T) string {
	// Create a temporary directory for the test workspace
	dir, err := os.MkdirTemp("", "lsp-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	// Create a test template file
	tmplContent := `{{- /*gotype: github.com/walteh/go-tmpl-typer/test/types.Person */ -}}
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

	// Create a test types package
	typesDir := filepath.Join(dir, "test", "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

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

func TestIntegration_TemplateValidation(t *testing.T) {
	// Setup test workspace
	workspace := setupTestWorkspace(t)

	// Read the test template file
	tmplContent, err := os.ReadFile(filepath.Join(workspace, "test.tmpl"))
	require.NoError(t, err)

	// Create mocks
	mockParser := mockery.NewMockTemplateParser_parser(t)
	mockValidator := mockery.NewMockValidator_types(t)
	mockAnalyzer := mockery.NewMockPackageAnalyzer_ast(t)
	mockGenerator := mockery.NewMockGenerator_diagnostic(t)

	// Setup expectations
	templateInfo := &parser.TemplateInfo{
		TypeHints: []parser.TypeHint{{
			TypePath: "github.com/walteh/go-tmpl-typer/test/types.Person",
			Line:     1,
			Column:   12,
		}},
		Variables: []parser.VariableLocation{{
			Name:    "Name",
			Line:    3,
			Column:  9,
			EndLine: 3,
			EndCol:  13,
		}, {
			Name:    "Address.Street",
			Line:    4,
			Column:  9,
			EndLine: 4,
			EndCol:  22,
		}},
	}

	// Expect Parse to be called with the template content
	mockParser.EXPECT().
		Parse(mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("string")).
		Run(func(ctx context.Context, content []byte, path string) {
			t.Logf("Parse called with content length: %d", len(content))
			assert.Equal(t, string(tmplContent), string(content), "Content should match template file")
			assert.Contains(t, path, "test.tmpl", "Path should contain template filename")
		}).
		Return(templateInfo, nil).
		Once()

	// Expect AnalyzePackage to be called for the type hint
	registry := ast.NewTypeRegistry()
	mockAnalyzer.EXPECT().
		AnalyzePackage(mock.Anything, mock.AnythingOfType("string")).
		Run(func(ctx context.Context, path string) {
			t.Logf("AnalyzePackage called with path: %s", path)
			assert.Contains(t, path, "test/types", "Path should contain types package")
		}).
		Return(registry, nil).
		Once()

	// Setup validator expectations
	typeInfo := &pkg_types.TypeInfo{
		Name: "Person",
		Fields: map[string]*pkg_types.FieldInfo{
			"Name": {
				Name: "Name",
				Type: types.Typ[types.String],
			},
			"Address": {
				Name: "Address",
				Type: types.NewStruct([]*types.Var{
					types.NewVar(0, nil, "Street", types.Typ[types.String]),
					types.NewVar(0, nil, "City", types.Typ[types.String]),
				}, nil),
			},
		},
	}

	// Expect ValidateType to be called for the type hint
	mockValidator.EXPECT().
		ValidateType(mock.Anything, "github.com/walteh/go-tmpl-typer/test/types.Person", registry).
		Return(typeInfo, nil).
		Once()

	// Expect ValidateField to be called for each variable
	mockValidator.EXPECT().
		ValidateField(mock.Anything, typeInfo, "Name").
		Return(&pkg_types.FieldInfo{
			Name: "Name",
			Type: types.Typ[types.String],
		}, nil).
		Once()

	mockValidator.EXPECT().
		ValidateField(mock.Anything, typeInfo, "Address.Street").
		Return(&pkg_types.FieldInfo{
			Name: "Street",
			Type: types.Typ[types.String],
		}, nil).
		Once()

	// Expect GetRootMethods to be called
	mockValidator.EXPECT().
		GetRootMethods().
		Return(map[string]*pkg_types.MethodInfo{}).
		Once()

	// Expect Generate to be called with the template info and return diagnostics
	diagnosticResult := &diagnostic.Diagnostics{
		Errors: []diagnostic.Diagnostic{{
			Message:  "test error",
			Line:     1,
			Column:   1,
			EndLine:  1,
			EndCol:   10,
			Severity: diagnostic.Error,
		}},
	}
	mockGenerator.EXPECT().
		Generate(mock.Anything, templateInfo, mockValidator, registry).
		Return(diagnosticResult, nil).
		Once()

	// Create server with mocks
	server := NewServer(mockParser, mockValidator, mockAnalyzer, mockGenerator, true)

	// Create mock connection
	rwc := newMockRWC()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Start(ctx, rwc, rwc)
	}()

	// Ensure cleanup
	defer func() {
		cancel()
		rwc.drainMessages()
		select {
		case err := <-serverDone:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("server error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("timeout waiting for server to stop")
		}
	}()

	// Send initialize request
	id := int64(1)
	initParams := InitializeParams{
		RootURI: "file://" + workspace,
	}
	rwc.writeMessage(t, "initialize", &id, initParams)

	// Wait for initialize response
	err = rwc.waitForMessage(t, 1*time.Second)
	require.NoError(t, err)

	// Read and verify initialize response
	_, respID, result, err := rwc.readMessage(t)
	require.NoError(t, err)
	require.NotNil(t, respID)
	require.Equal(t, id, *respID)

	// Verify initialize result
	var initResult InitializeResult
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	err = json.Unmarshal(resultBytes, &initResult)
	require.NoError(t, err)
	assert.NotNil(t, initResult.Capabilities)

	// Send initialized notification
	rwc.writeMessage(t, "initialized", nil, nil)

	// Send didOpen notification
	didOpenParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:     "file://" + filepath.Join(workspace, "test.tmpl"),
			Version: 1,
			Text:    string(tmplContent),
		},
	}
	rwc.writeMessage(t, "textDocument/didOpen", nil, didOpenParams)

	// Wait for diagnostic notification and skip any log messages
	for {
		err := rwc.waitForMessage(t, 1*time.Second)
		require.NoError(t, err)
		method, _, result, err := rwc.readMessage(t)
		require.NoError(t, err)
		if method == "textDocument/publishDiagnostics" {
			// Verify diagnostics
			var diagParams PublishDiagnosticsParams
			resultBytes, err := json.Marshal(result)
			require.NoError(t, err)
			err = json.Unmarshal(resultBytes, &diagParams)
			require.NoError(t, err)

			// We should get the test error from our mock
			require.Len(t, diagParams.Diagnostics, 1)
			assert.Equal(t, "test error", diagParams.Diagnostics[0].Message)
			assert.Equal(t, 1, diagParams.Diagnostics[0].Severity)
			assert.Equal(t, 0, diagParams.Diagnostics[0].Range.Start.Line)
			assert.Equal(t, 0, diagParams.Diagnostics[0].Range.Start.Character)
			assert.Equal(t, 0, diagParams.Diagnostics[0].Range.End.Line)
			assert.Equal(t, 9, diagParams.Diagnostics[0].Range.End.Character)
			break
		}
		t.Logf("Skipping message: method=%s", method)
	}

	// Send shutdown request
	id = 2
	rwc.writeMessage(t, "shutdown", &id, nil)

	// Wait for shutdown response
	err = rwc.waitForMessage(t, 1*time.Second)
	require.NoError(t, err)
	_, respID, _, err = rwc.readMessage(t)
	require.NoError(t, err)
	require.NotNil(t, respID)
	require.Equal(t, id, *respID)

	// Send exit notification
	rwc.writeMessage(t, "exit", nil, nil)
}
