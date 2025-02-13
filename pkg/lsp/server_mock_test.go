package lsp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/mockery"
	"github.com/walteh/gotmpls/pkg/lsp"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
)

// setupMockServer creates a server with a mock client for testing.
// It returns the mock client for setting expectations and the server for making requests.
func setupMockServer(t *testing.T, docs map[string]string) (context.Context, *mockery.MockClient_protocol, *lsp.Server, func(string) protocol.DocumentURI) {
	t.Helper()

	ctx := context.Background()

	ctx = zerolog.New(zerolog.TestWriter{T: t}).With().Str("test", t.Name()).Timestamp().Logger().WithContext(ctx)

	// Create server
	server := lsp.NewServer(ctx)

	// Create mock client and set it up
	mockClient := mockery.NewMockClient_protocol(t)
	server.SetCallbackClient(mockClient)

	tmpDir := t.TempDir()

	for uri, content := range docs {
		var langID string
		if strings.HasSuffix(uri, ".tmpl") {
			langID = "gotmpl"
		} else {
			langID = "go"
		}
		docURI := mockToDocURI(filepath.Join(tmpDir, uri))
		server.Documents().Store(docURI, &lsp.Document{
			URI:        string(docURI),
			LanguageID: protocol.LanguageKind(langID),
			Version:    1,
			Content:    content,
		})
		os.WriteFile(filepath.Join(tmpDir, uri), []byte(content), 0644)
	}

	return ctx, mockClient, server, func(uri string) protocol.DocumentURI {
		if strings.HasPrefix(uri, "file://") {
			return protocol.DocumentURI(uri)
		}
		return mockToDocURI(filepath.Join(tmpDir, uri))
	}
}

func mockToDocURI(uri string) protocol.DocumentURI {
	if strings.HasPrefix(uri, "file://") {
		return protocol.DocumentURI(uri)
	}
	return protocol.DocumentURI("file://" + uri)
}

func TestMockServerHover(t *testing.T) {

	t.Run("hover_shows_field_info", func(t *testing.T) {

		files := map[string]string{
			"go.mod": "module test",
			"test.go": `package test

import _ "embed"
//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Name string
}`,
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ .Name }}`,
		}

		ctx, mockClient, server, toDocURI := setupMockServer(t, files)

		// Get hover info
		hoverResult, err := server.Hover(ctx, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: toDocURI("test.tmpl"),
				},
				Position: protocol.Position{Line: 1, Character: 3},
			},
		})
		require.NoError(t, err, "hover should succeed")
		require.NotNil(t, hoverResult, "hover result should not be nil")
		require.Equal(t, "### Type Information\n\n```go\ntype Person struct {\n\tName string\n}\n```\n\n### Template Access\n```gotmpl\n.Name\n```", hoverResult.Contents.Value)

		mockClient.AssertExpectations(t)
	})
}

func TestMockServerDiagnostics(t *testing.T) {

	t.Run("invalid_field_shows_diagnostic", func(t *testing.T) {

		files := map[string]string{
			"go.mod": "module test",
			"test.go": `package test

import _ "embed"
//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Name string
}`,
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ .InvalidField }}`,
		}

		ctx, mockClient, server, toDocURI := setupMockServer(t, files)

		var params *protocol.PublishDiagnosticsParams
		// Set up expectations for diagnostics
		mockClient.EXPECT().PublishDiagnostics(ctx, mock.MatchedBy(func(p *protocol.PublishDiagnosticsParams) bool {
			params = p
			return p.URI == toDocURI("test.tmpl")
		})).Return(nil).Once()

		// Trigger diagnostics by making a change
		err := server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				Version: 1,
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{
					URI: toDocURI("test.tmpl"),
				},
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				{
					Text: "x",
					Range: &protocol.Range{
						Start: protocol.Position{Line: 1, Character: 16},
						End:   protocol.Position{Line: 1, Character: 16},
					},
					RangeLength: 2,
				},
			},
		})
		require.NoError(t, err, "change should succeed")

		mockClient.AssertExpectations(t)
		expectedDiag := []protocol.Diagnostic{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 14},
					End:   protocol.Position{Line: 0, Character: 25},
				},
				Severity: protocol.SeverityInformation,
				Message:  "type hint successfully loaded: test.Person",
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 3},
					End:   protocol.Position{Line: 1, Character: 17},
				},
				Severity: protocol.SeverityError,
				Message:  "field not found [ InvalidFieldx ] in type [ Person ]",
			},
		}
		require.Equal(t, expectedDiag, params.Diagnostics)
	})
}

func TestMockServerSemanticTokens(t *testing.T) {

	t.Run("semantic_tokens_for_template", func(t *testing.T) {

		files := map[string]string{
			"go.mod": "module test",
			"test.go": `package test

import _ "embed"
//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Name string
}`,
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ if eq .Name "test" }}
	{{ printf "Hello, %s" .Name }}
{{ end }}`,
		}

		ctx, mockClient, server, toDocURI := setupMockServer(t, files)

		// Get semantic tokens for the entire file
		tokens, err := server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: toDocURI("test.tmpl"),
			},
		})
		require.NoError(t, err, "semantic tokens request should succeed")
		require.NotNil(t, tokens, "semantic tokens should not be nil")
		require.NotEmpty(t, tokens.Data, "should have semantic tokens")

		// Get semantic tokens for a specific range
		rangeTokens, err := server.SemanticTokensRange(ctx, &protocol.SemanticTokensRangeParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: toDocURI("test.tmpl"),
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 1, Character: 25},
			},
		})
		require.NoError(t, err, "semantic tokens range request should succeed")
		require.NotNil(t, rangeTokens, "semantic tokens for range should not be nil")
		require.NotEmpty(t, rangeTokens.Data, "should have semantic tokens for range")

		mockClient.AssertExpectations(t)
	})
}

func TestMockServerDocumentLifecycle(t *testing.T) {

	t.Run("document_lifecycle", func(t *testing.T) {

		files := map[string]string{
			"go.mod": "module test",
			"test.go": `package test

import _ "embed"
//go:embed test.tmpl
var TestTemplate string

type Person struct {
	Name string
}`,
			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
{{ .Name }}`,
		}

		ctx, mockClient, server, toDocURI := setupMockServer(t, files)

		// Set up expectations for diagnostics
		mockClient.EXPECT().PublishDiagnostics(ctx, mock.MatchedBy(func(p *protocol.PublishDiagnosticsParams) bool {
			return p.URI == toDocURI("test.tmpl")
		})).Return(nil).Twice()

		mockClient.EXPECT().SemanticTokensRefresh(ctx).Return(nil).Once()

		// Open document
		err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        toDocURI("test.tmpl"),
				LanguageID: "gotmpl",
				Version:    1,
				Text:       files["test.tmpl"],
			},
		})
		require.NoError(t, err, "document open should succeed")

		// Save document
		text := files["test.tmpl"]
		err = server.DidSave(ctx, &protocol.DidSaveTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: toDocURI("test.tmpl"),
			},
			Text: &text,
		})
		require.NoError(t, err, "document save should succeed")

		// Close document
		err = server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: toDocURI("test.tmpl"),
			},
		})
		require.NoError(t, err, "document close should succeed")

		// Verify document is removed from store
		_, exists := server.Documents().GetNoFallback(toDocURI("test.tmpl"))
		require.False(t, exists, "document should be removed after close")

		mockClient.AssertExpectations(t)
	})
}

func TestMockServerLifecycle(t *testing.T) {
	t.Run("server_lifecycle", func(t *testing.T) {
		ctx := context.Background()
		server := lsp.NewServer(ctx)

		// Create mock client
		mockClient := mockery.NewMockClient_protocol(t)
		server.SetCallbackClient(mockClient)

		// Initialize server
		params := &protocol.ParamInitialize{
			XInitializeParams: protocol.XInitializeParams{
				ProcessID: 1,
				RootURI:   protocol.DocumentURI("file:///workspace"),
				Capabilities: protocol.ClientCapabilities{
					TextDocument: protocol.TextDocumentClientCapabilities{
						SemanticTokens: protocol.SemanticTokensClientCapabilities{
							DynamicRegistration: true,
							Requests: protocol.ClientSemanticTokensRequestOptions{
								Range: &protocol.Or_ClientSemanticTokensRequestOptions_range{Value: true},
								Full:  &protocol.Or_ClientSemanticTokensRequestOptions_full{Value: true},
							},
							TokenTypes:     []string{"namespace", "type", "class", "enum", "interface", "struct", "typeParameter", "parameter", "variable", "property", "enumMember", "event", "function", "method", "macro", "keyword", "modifier", "comment", "string", "number", "regexp", "operator", "decorator"},
							TokenModifiers: []string{"declaration", "definition", "readonly", "static", "deprecated", "abstract", "async", "modification", "documentation", "defaultLibrary"},
							Formats:        []protocol.TokenFormat{protocol.Relative},
						},
					},
				},
			},
		}
		initResult, err := server.Initialize(ctx, params)
		require.NoError(t, err, "initialize should succeed")
		require.NotNil(t, initResult, "initialize result should not be nil")
		require.NotNil(t, initResult.Capabilities.HoverProvider, "hover should be supported")
		// We don't check for semantic tokens provider here since we're using dynamic registration

		// Set up expectations for registration
		mockClient.EXPECT().RegisterCapability(ctx, mock.MatchedBy(func(params *protocol.RegistrationParams) bool {
			if len(params.Registrations) != 1 {
				return false
			}
			reg := params.Registrations[0]
			return reg.ID == "semantic-tokens" && reg.Method == "textDocument/semanticTokens"
		})).Return(nil).Once()

		// Call Initialized which should trigger registration
		err = server.Initialized(ctx, &protocol.InitializedParams{})
		require.NoError(t, err, "initialized should succeed")

		// Shutdown server
		err = server.Shutdown(ctx)
		require.NoError(t, err, "shutdown should succeed")

		// Exit server
		err = server.Exit(ctx)
		require.NoError(t, err, "exit should succeed")

		mockClient.AssertExpectations(t)
	})
}

func TestMockServerSemanticTokenRegistration(t *testing.T) {
	t.Run("semantic_token_registration", func(t *testing.T) {
		ctx := context.Background()
		server := lsp.NewServer(ctx)

		// Create mock client
		mockClient := mockery.NewMockClient_protocol(t)
		server.SetCallbackClient(mockClient)

		// Set up expectations for registration
		mockClient.EXPECT().RegisterCapability(ctx, mock.MatchedBy(func(params *protocol.RegistrationParams) bool {
			if len(params.Registrations) != 1 {
				return false
			}
			reg := params.Registrations[0]
			if reg.ID != "semantic-tokens" || reg.Method != "textDocument/semanticTokens" {
				return false
			}
			opts, ok := reg.RegisterOptions.(*protocol.SemanticTokensRegistrationOptions)
			if !ok {
				return false
			}
			if len(opts.DocumentSelector) != 1 {
				return false
			}
			filter := opts.DocumentSelector[0]
			filterValue, ok := filter.Value.(protocol.Or_TextDocumentFilter)
			if !ok {
				return false
			}
			filterLang, ok := filterValue.Value.(protocol.TextDocumentFilterLanguage)
			if !ok {
				return false
			}
			if filterLang.Language != "gotmpl" {
				return false
			}

			// Verify semantic token options
			if len(opts.Legend.TokenTypes) != 14 || len(opts.Legend.TokenModifiers) != 10 {
				return false
			}
			if opts.Legend.TokenTypes[0] != "variable" || opts.Legend.TokenModifiers[0] != "declaration" {
				return false
			}

			// Verify full and range support
			if opts.Full == nil || opts.Range == nil {
				return false
			}
			fullValue, ok := opts.Full.Value.(bool)
			if !ok || !fullValue {
				return false
			}
			rangeValue, ok := opts.Range.Value.(bool)
			if !ok || !rangeValue {
				return false
			}

			return true
		})).Return(nil).Once()

		// Initialize server with client capabilities
		initResult, err := server.Initialize(ctx, &protocol.ParamInitialize{
			XInitializeParams: protocol.XInitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: protocol.TextDocumentClientCapabilities{
						SemanticTokens: protocol.SemanticTokensClientCapabilities{
							DynamicRegistration: true,
							Requests: protocol.ClientSemanticTokensRequestOptions{
								Range: &protocol.Or_ClientSemanticTokensRequestOptions_range{Value: true},
								Full:  &protocol.Or_ClientSemanticTokensRequestOptions_full{Value: true},
							},
							TokenTypes:     []string{"namespace", "type", "class", "enum", "interface", "struct", "typeParameter", "parameter", "variable", "property", "enumMember", "event", "function", "method", "macro", "keyword", "modifier", "comment", "string", "number", "regexp", "operator", "decorator"},
							TokenModifiers: []string{"declaration", "definition", "readonly", "static", "deprecated", "abstract", "async", "modification", "documentation", "defaultLibrary"},
							Formats:        []protocol.TokenFormat{protocol.Relative},
						},
					},
				},
			},
		})
		require.NoError(t, err, "initialize should succeed")
		require.NotNil(t, initResult, "initialize result should not be nil")

		// Call Initialized which should trigger registration
		err = server.Initialized(ctx, &protocol.InitializedParams{})
		require.NoError(t, err, "initialized should succeed")

		mockClient.AssertExpectations(t)
	})
}
