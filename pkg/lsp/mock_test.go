package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
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
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockRWC(t *testing.T) (*mockRWC, context.Context) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: io.Discard, NoColor: true}).Level(zerolog.WarnLevel)
	ctx := logger.WithContext(context.Background())

	rwc := &mockRWC{
		readBuf:  bytes.NewBuffer(nil),
		writeBuf: bytes.NewBuffer(nil),
	}

	return rwc, ctx
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

	return n, err
}

func (m *mockRWC) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, fmt.Errorf("write on closed connection")
	}

	return m.writeBuf.Write(p)
}

func (m *mockRWC) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	// Clear buffers on close
	m.readBuf.Reset()
	m.writeBuf.Reset()
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

	m.mu.Lock()
	defer m.mu.Unlock()

	_, err = m.readBuf.WriteString(header)
	require.NoError(t, err)

	_, err = m.readBuf.Write(data)
	require.NoError(t, err)
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
		return "", nil, nil, fmt.Errorf("invalid header: %q", header)
	}
	contentLength, err := strconv.Atoi(match[1])
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid content length: %q", match[1])
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
		return "", msg.ID, nil, msg.Error
	}

	result = msg.Result
	if result == nil {
		result = msg.Params
	}

	return msg.Method, msg.ID, result, nil
}
