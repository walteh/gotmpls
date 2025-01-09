package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

var contentLengthRegexp = regexp.MustCompile(`Content-Length: (\d+)`)

type jsonrpcError struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// mockRWC implements a mock io.ReadWriteCloser for testing
type mockRWC struct {
	readBuf         *bytes.Buffer
	writeBuf        *bytes.Buffer
	closed          bool
	mu              sync.Mutex
	creationContext context.Context
}

func newMockRWC(t *testing.T) (*mockRWC, context.Context) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).
		With().
		Timestamp().
		Str("test", t.Name()).
		Logger()

	ctx := logger.WithContext(context.Background())

	return &mockRWC{
		readBuf:         bytes.NewBuffer(nil),
		writeBuf:        bytes.NewBuffer(nil),
		creationContext: ctx,
	}, ctx
}

func (m *mockRWC) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	for m.readBuf.Len() == 0 {
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
		if m.closed {
			return 0, io.EOF
		}
	}

	n, err = m.readBuf.Read(p)
	if err == io.EOF && n > 0 {
		err = nil
	}

	if m.creationContext == nil {
		panic("what")
	}
	zerolog.Ctx(m.creationContext).
		Debug().
		Int("bytes", n).
		Err(err).
		Bytes("data", p[:n]).
		Send()
	return n, err
}

func (m *mockRWC) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.Errorf("write on closed connection")
	}

	n, err = m.writeBuf.Write(p)
	zerolog.Ctx(m.creationContext).Debug().Int("bytes", n).Err(err).Bytes("data", p).Send()
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

func (m *mockRWC) writeMessage(ctx context.Context, t *testing.T, method string, id *int64, params interface{}) {
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
	zerolog.Ctx(ctx).Debug().
		Str("header", header).
		RawJSON("body", data).
		Msg("writing message")

	m.mu.Lock()
	defer m.mu.Unlock()

	_, err = m.readBuf.WriteString(header)
	require.NoError(t, err)

	_, err = m.readBuf.Write(data)
	require.NoError(t, err)

	zerolog.Ctx(ctx).Debug().Int("bufferLen", m.readBuf.Len()).Msg("message written")
}

func (m *mockRWC) readMessage(ctx context.Context) (method string, id *int64, result interface{}, err error) {
	// Read the header first
	var header string
	for {
		b, err := m.writeBuf.ReadByte()
		if err != nil {
			if err == io.EOF {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return "", nil, nil, err
		}
		header += string(b)
		if strings.HasSuffix(header, "\r\n\r\n") {
			break
		}
	}

	// Parse content length
	match := contentLengthRegexp.FindStringSubmatch(header)
	if match == nil {
		return "", nil, nil, errors.Errorf("invalid header: %q", header)
	}
	contentLength, err := strconv.Atoi(match[1])
	if err != nil {
		return "", nil, nil, errors.Errorf("invalid content length: %q", match[1])
	}

	// Read the content
	content := make([]byte, contentLength)
	_, err = io.ReadFull(m.writeBuf, content)
	if err != nil {
		return "", nil, nil, err
	}

	zerolog.Ctx(ctx).Debug().
		Str("header", header).
		RawJSON("content", content).
		Msg("read message")

	// Parse the message
	var msg struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      *int64        `json:"id,omitempty"`
		Method  string        `json:"method,omitempty"`
		Result  interface{}   `json:"result,omitempty"`
		Params  interface{}   `json:"params,omitempty"`
		Error   *jsonrpcError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(content, &msg); err != nil {
		return "", nil, nil, err
	}

	if msg.Error != nil {
		return "", msg.ID, nil, errors.Errorf("JSON-RPC error: %v", msg.Error)
	}

	result = msg.Result
	if result == nil {
		result = msg.Params
	}

	return msg.Method, msg.ID, result, nil
}

func (m *mockRWC) drainMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readBuf.Reset()
	m.writeBuf.Reset()
}

// setupTestServer creates a new server with debug enabled and a mock connection
func setupTestServer(t *testing.T) (*Server, *mockRWC, context.Context, context.CancelFunc) {
	// Setup test logger with test output

	// Create server with test logger
	server := NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	// Create mock RWC with test logger
	rwc, ctx := newMockRWC(t)

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	return server, rwc, ctx, cancel
}

// waitForMessage waits for a specific message type from the server
func waitForMessage(ctx context.Context, rwc *mockRWC, serverErrCh chan error, expectedMethod string, timeout time.Duration) (string, *int64, interface{}, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	msgCh := make(chan struct {
		method string
		id     *int64
		result interface{}
		err    error
	})

	go func() {
		for {
			method, id, result, err := rwc.readMessage(ctx)
			if err != nil {
				msgCh <- struct {
					method string
					id     *int64
					result interface{}
					err    error
				}{"", nil, nil, err}
				return
			}

			if method == "window/logMessage" {
				zerolog.Ctx(ctx).Debug().Interface("message", result).Msg("received log message")
				continue
			}

			// For responses to requests, the method will be empty and we should look at the ID
			if expectedMethod == "" && id != nil {
				msgCh <- struct {
					method string
					id     *int64
					result interface{}
					err    error
				}{"initialize", id, result, nil}
				return
			}

			if expectedMethod != "" && method == expectedMethod {
				msgCh <- struct {
					method string
					id     *int64
					result interface{}
					err    error
				}{method, id, result, nil}
				return
			}
		}
	}()

	select {
	case msg := <-msgCh:
		return msg.method, msg.id, msg.result, msg.err
	case err := <-serverErrCh:
		return "", nil, nil, fmt.Errorf("server error: %w", err)
	case <-timer.C:
		return "", nil, nil, fmt.Errorf("timeout waiting for message: %s", expectedMethod)
	}
}

func TestServer_Initialize(t *testing.T) {
	// Setup test with logging

	// Create server with debug enabled
	server := NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	// Create mock connection with test logger
	rwc, ctx := newMockRWC(t)

	// Start server in background with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Start server in goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start(ctx, rwc, rwc)
	}()

	// Send initialize request
	id := int64(1)
	rwc.writeMessage(ctx, t, "initialize", &id, InitializeParams{
		RootURI: "file:///test",
	})

	// Wait for and verify initialize response
	var initResult InitializeResult
	for {
		select {
		case err := <-serverErrCh:
			require.NoError(t, err)
			return
		case <-ctx.Done():
			t.Fatal("timeout waiting for initialize response")
			return
		default:
			method, respID, result, err := rwc.readMessage(ctx)
			require.NoError(t, err)

			// Skip log messages
			if method == "window/logMessage" {
				zerolog.Ctx(ctx).Debug().Interface("message", result).Msg("received log message")
				continue
			}

			// Found initialize response
			if respID != nil {
				require.Equal(t, id, *respID)
				resultBytes, err := json.Marshal(result)
				require.NoError(t, err)
				err = json.Unmarshal(resultBytes, &initResult)
				require.NoError(t, err)

				// Verify capabilities
				require.True(t, initResult.Capabilities.HoverProvider)
				require.NotNil(t, initResult.Capabilities.TextDocumentSync)
				require.Equal(t, 1, initResult.Capabilities.TextDocumentSync.Change)
				return
			}
		}
	}
}

func TestMessageEncoding(t *testing.T) {
	rwc, ctx := newMockRWC(t)

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
	method, respID, result, err := rwc.readMessage(ctx)
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

func TestServer_DidOpen(t *testing.T) {
	dir := setupTestWorkspace(t)

	server, rwc, ctx, cancel := setupTestServer(t)
	defer cancel()

	// Start server in goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		t.Log("Starting server...")
		serverErrCh <- server.Start(ctx, rwc, rwc)
	}()

	// First send initialize request
	t.Log("Sending initialize request...")
	id := int64(1)
	rwc.writeMessage(ctx, t, "initialize", &id, InitializeParams{
		RootURI: "file://" + dir,
	})

	// Wait for initialize response
	method, respID, result, err := waitForMessage(ctx, rwc, serverErrCh, "", 5*time.Second)
	require.NoError(t, err)
	require.Equal(t, "initialize", method)
	require.NotNil(t, respID)
	require.Equal(t, id, *respID)

	t.Log("Initialize response received")

	// Send initialized notification
	t.Log("Sending initialized notification...")
	rwc.writeMessage(ctx, t, "initialized", nil, struct{}{})

	// Give the server a moment to process the initialized notification
	time.Sleep(100 * time.Millisecond)

	// Send didOpen notification
	t.Log("Sending didOpen notification...")
	tmplContent, err := os.ReadFile(filepath.Join(dir, "test.tmpl"))
	require.NoError(t, err)

	rwc.writeMessage(ctx, t, "textDocument/didOpen", nil, &DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file://" + filepath.Join(dir, "test.tmpl"),
			LanguageID: "go-template",
			Version:    1,
			Text:       string(tmplContent),
		},
	})

	// Wait for publishDiagnostics notification
	t.Log("Waiting for diagnostics...")
	method, _, result, err = waitForMessage(ctx, rwc, serverErrCh, "textDocument/publishDiagnostics", 10*time.Second)
	require.NoError(t, err)
	require.Equal(t, "textDocument/publishDiagnostics", method)

	var diagParams PublishDiagnosticsParams
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	err = json.Unmarshal(resultBytes, &diagParams)
	require.NoError(t, err)

	// Verify diagnostics
	require.NotNil(t, diagParams)
	require.Equal(t, "file://"+filepath.Join(dir, "test.tmpl"), diagParams.URI)
	require.Empty(t, diagParams.Diagnostics, "Expected no diagnostics since go.mod exists and template is valid")

	t.Log("Test completed successfully")
}
