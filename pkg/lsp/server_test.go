package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formatLSPMessage(method string, id *int64, params interface{}) (string, error) {
	msg := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      *int64      `json:"id,omitempty"`
		Method  string      `json:"method,omitempty"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data), nil
}

func TestMessageFormatting(t *testing.T) {
	t.Run("initialize response has correct format", func(t *testing.T) {
		result := InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: TextDocumentSyncKind{
					Change: 1,
				},
				HoverProvider: true,
				CompletionProvider: CompletionOptions{
					TriggerCharacters: []string{"."},
				},
			},
		}

		id := int64(1)
		msg, err := formatLSPMessage("", &id, result)
		require.NoError(t, err)
		t.Log("Message:", msg)

		// Verify format
		assert.Contains(t, msg, "Content-Length: ")
		assert.Contains(t, msg, "\r\n\r\n")
		assert.Contains(t, msg, `"jsonrpc":"2.0"`)
		assert.Contains(t, msg, `"id":1`)
		assert.Contains(t, msg, `"capabilities"`)

		// Verify structure
		var parsed struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      int64           `json:"id"`
			Result  json.RawMessage `json:"result"`
		}
		parts := bytes.Split([]byte(msg), []byte("\r\n\r\n"))
		require.Len(t, parts, 2)
		err = json.Unmarshal(parts[1], &parsed)
		require.NoError(t, err)
		assert.Equal(t, "2.0", parsed.JSONRPC)
		assert.Equal(t, int64(1), parsed.ID)
	})

	t.Run("diagnostic notification has correct format", func(t *testing.T) {
		diagnostics := []Diagnostic{{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 10},
			},
			Severity: 1,
			Message:  "test error",
		}}

		params := PublishDiagnosticsParams{
			URI:         "file:///test.tmpl",
			Diagnostics: diagnostics,
		}

		msg, err := formatLSPMessage("textDocument/publishDiagnostics", nil, params)
		require.NoError(t, err)
		t.Log("Message:", msg)

		// Verify format
		assert.Contains(t, msg, "Content-Length: ")
		assert.Contains(t, msg, "\r\n\r\n")
		assert.Contains(t, msg, `"jsonrpc":"2.0"`)
		assert.Contains(t, msg, `"method":"textDocument/publishDiagnostics"`)
		assert.Contains(t, msg, `"params"`)

		// Verify structure
		var parsed struct {
			JSONRPC string                   `json:"jsonrpc"`
			Method  string                   `json:"method"`
			Params  PublishDiagnosticsParams `json:"params"`
		}
		parts := bytes.Split([]byte(msg), []byte("\r\n\r\n"))
		require.Len(t, parts, 2)
		err = json.Unmarshal(parts[1], &parsed)
		require.NoError(t, err)
		assert.Equal(t, "2.0", parsed.JSONRPC)
		assert.Equal(t, "textDocument/publishDiagnostics", parsed.Method)
		assert.Equal(t, "file:///test.tmpl", parsed.Params.URI)
		assert.Len(t, parsed.Params.Diagnostics, 1)
	})
}
