package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// mockRWC implements a mock io.ReadWriteCloser for testing
type mockRWC struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockRWC() *mockRWC {
	return &mockRWC{
		readBuf:  bytes.NewBuffer(nil),
		writeBuf: bytes.NewBuffer(nil),
	}
}

func (m *mockRWC) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	// If there's no data to read, wait a bit
	if m.readBuf.Len() == 0 {
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
	}

	// Read from the read buffer since this is what the server will read
	n, err = m.readBuf.Read(p)
	if err == io.EOF && n > 0 {
		err = nil
	}
	fmt.Printf("mockRWC Read: %d bytes, err: %v, data: %s\n", n, err, string(p[:n]))
	return n, err
}

func (m *mockRWC) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.Errorf("write on closed connection")
	}

	// Write to the write buffer since this is what the client will read
	n, err = m.writeBuf.Write(p)
	fmt.Printf("mockRWC Write: %d bytes, err: %v, data: %s\n", n, err, string(p))
	return n, err
}

func (m *mockRWC) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	return nil
}

func (m *mockRWC) writeMessage(t *testing.T, method string, id *int64, params interface{}) {
	msg := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      *int64      `json:"id,omitempty"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	t.Logf("Writing message header: %q", header)
	t.Logf("Writing message body: %s", string(data))

	m.mu.Lock()
	defer m.mu.Unlock()

	// Write to the read buffer since this is what the server will read
	_, err = m.readBuf.WriteString(header)
	require.NoError(t, err)

	_, err = m.readBuf.Write(data)
	require.NoError(t, err)

	t.Logf("Read buffer length after write: %d", m.readBuf.Len())
}

func (m *mockRWC) waitForMessage(t *testing.T, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		m.mu.Lock()
		hasData := m.writeBuf.Len() > 0
		if hasData {
			t.Logf("Found message in write buffer (length=%d): %s", m.writeBuf.Len(), m.writeBuf.String())
		} else {
			t.Logf("No data in write buffer yet")
		}
		m.mu.Unlock()

		if hasData {
			return nil
		}

		select {
		case <-timer.C:
			m.mu.Lock()
			readLen := m.readBuf.Len()
			writeLen := m.writeBuf.Len()
			m.mu.Unlock()
			return errors.Errorf("timeout waiting for message (readBuf=%d, writeBuf=%d)", readLen, writeLen)
		case <-time.After(10 * time.Millisecond):
			continue
		}
	}
}

func (m *mockRWC) readMessage(t *testing.T) (method string, id *int64, result interface{}, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		// Read until we find the Content-Length header
		var contentLength int
		for {
			line, err := m.writeBuf.ReadString('\n')
			if err != nil {
				return "", nil, nil, err
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Content-Length: ") {
				length := strings.TrimPrefix(line, "Content-Length: ")
				contentLength, err = strconv.Atoi(length)
				if err != nil {
					return "", nil, nil, err
				}
			} else if line == "" && contentLength > 0 {
				// Empty line after Content-Length header means we're ready to read the body
				break
			}
		}

		if contentLength == 0 {
			return "", nil, nil, errors.Errorf("no Content-Length header found")
		}

		// Read message body
		body := make([]byte, contentLength)
		n, err := io.ReadFull(m.writeBuf, body)
		if err != nil {
			return "", nil, nil, err
		}
		if n != contentLength {
			return "", nil, nil, errors.Errorf("expected to read %d bytes, got %d", contentLength, n)
		}

		t.Logf("Read message: %s", string(body))

		// Parse message
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      *int64          `json:"id,omitempty"`
			Method  string          `json:"method,omitempty"`
			Result  json.RawMessage `json:"result,omitempty"`
			Params  json.RawMessage `json:"params,omitempty"`
		}
		err = json.Unmarshal(body, &msg)
		if err != nil {
			return "", nil, nil, err
		}

		// Skip log messages
		if msg.Method == "window/logMessage" {
			continue
		}

		var resultObj interface{}
		if len(msg.Result) > 0 {
			err = json.Unmarshal(msg.Result, &resultObj)
			if err != nil {
				return "", nil, nil, err
			}
		} else if len(msg.Params) > 0 {
			err = json.Unmarshal(msg.Params, &resultObj)
			if err != nil {
				return "", nil, nil, err
			}
		}

		if msg.Method != "" {
			return msg.Method, msg.ID, resultObj, nil
		}
		return method, msg.ID, resultObj, nil
	}
}

func (m *mockRWC) drainMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readBuf.Reset()
	m.writeBuf.Reset()
}

func createTestWorkspace(t *testing.T) string {
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

func TestServer_Initialize(t *testing.T) {
	// Create server with debug enabled
	server := NewServer(nil, nil, nil, nil, true)

	// Create mock connection
	rwc := newMockRWC()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	go func() {
		err := server.Start(ctx, rwc, rwc)
		require.NoError(t, err)
	}()

	// Send initialize request
	id := int64(1)
	initParams := InitializeParams{
		RootURI: "file:///test",
	}
	rwc.writeMessage(t, "initialize", &id, initParams)

	// Wait for and verify initialize response
	err := rwc.waitForMessage(t, 1*time.Second)
	require.NoError(t, err)

	// Skip any log messages and get the initialize response
	method, respID, result, err := rwc.readMessage(t)
	require.NoError(t, err)
	require.Empty(t, method) // Should be empty for a response
	require.NotNil(t, respID)
	require.Equal(t, id, *respID)

	// Verify initialize result
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	var initResult InitializeResult
	err = json.Unmarshal(resultBytes, &initResult)
	require.NoError(t, err)

	// Verify capabilities
	assert.Equal(t, 1, initResult.Capabilities.TextDocumentSync.Change)
	assert.True(t, initResult.Capabilities.HoverProvider)
	assert.Equal(t, []string{"."}, initResult.Capabilities.CompletionProvider.TriggerCharacters)
}

func TestMessageEncoding(t *testing.T) {
	rwc := newMockRWC()

	// Write a test message to the write buffer directly
	id := int64(1)
	params := InitializeParams{
		RootURI: "file:///test",
	}
	msg := struct {
		JSONRPC string           `json:"jsonrpc"`
		ID      *int64           `json:"id,omitempty"`
		Method  string           `json:"method"`
		Params  InitializeParams `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
		Params:  params,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, err = rwc.writeBuf.WriteString(header)
	require.NoError(t, err)
	_, err = rwc.writeBuf.Write(data)
	require.NoError(t, err)

	// Read it back
	method, respID, result, err := rwc.readMessage(t)
	require.NoError(t, err)
	require.Equal(t, "initialize", method)
	require.NotNil(t, respID)
	require.Equal(t, id, *respID)

	// Verify params
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	var readParams InitializeParams
	err = json.Unmarshal(resultBytes, &readParams)
	require.NoError(t, err)
	require.Equal(t, params.RootURI, readParams.RootURI)
}
