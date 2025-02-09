//go:build wasip1

package lsp

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/lsp"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

// StdioTransport implements a transport layer that uses standard input/output
type StdioTransport struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

// NewStdioTransport creates a new transport that uses stdin/stdout
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader: &debugReader{reader: os.Stdin},
		writer: &debugWriter{writer: os.Stdout},
	}
}

// debugReader wraps an io.Reader and logs all reads to stderr
type debugReader struct {
	reader io.ReadCloser
}

func (d *debugReader) Read(p []byte) (n int, err error) {
	fmt.Fprintf(os.Stderr, "🔍 STDIN Read called with buffer size: %d\n", len(p))
	n, err = d.reader.Read(p)

	// trim to only show the first and last 20 characters
	if n > 40 {
		fmt.Fprintf(os.Stderr, "📥 STDIN Read %d bytes: %q\n", n, string(p[:20])+" ... "+string(p[n-20:]))
	} else {
		fmt.Fprintf(os.Stderr, "📥 STDIN Read %d bytes: %q\n", n, string(p[:n]))
	}

	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "❌ STDIN ERR: %v\n", err)
	}
	return n, err
}

func (d *debugReader) Close() error {
	return d.reader.Close()
}

// debugWriter wraps an io.Writer and logs all writes to stderr
type debugWriter struct {
	writer io.WriteCloser
}

func (d *debugWriter) Write(p []byte) (n int, err error) {
	fmt.Fprintf(os.Stderr, "📤 About to write to STDOUT %d bytes: %s\n", len(p), string(p))
	n, err = d.writer.Write(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ STDOUT ERR: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "✅ Successfully wrote %d bytes to STDOUT\n", n)
	}
	return n, err
}

func (d *debugWriter) Close() error {
	return d.writer.Close()
}

// GetChannelStreams returns io.ReadWriteCloser for LSP communication
func (t *StdioTransport) GetChannelStreams() (io.ReadCloser, io.WriteCloser) {
	return t.reader, t.writer
}

type testRPCLogger struct {
}

func (l *testRPCLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	fmt.Fprintf(os.Stderr, "server received request via stdin: method=%s, params=%s", req.Method(), req.ParamString())
}

func (l *testRPCLogger) LogResponse(ctx context.Context, rsp *jrpc2.Response) {
	fmt.Fprintf(os.Stderr, "server sent response via stdout: id=%s, result=%s", rsp.ID(), rsp.ResultString())
}

// ServeLSP starts the LSP server with stdio transport
func ServeLSP(ctx context.Context) error {
	fmt.Fprintf(os.Stderr, "🚀 ServeLSP starting...\n")

	// Create logger
	logger := zerolog.Ctx(ctx)
	if logger == nil {
		return errors.New("logger not found in context")
	}

	// Create and start server
	server := lsp.NewServer(ctx)

	// Create server options with logging
	opts := &jrpc2.ServerOptions{
		AllowPush:   true, // Allow server to send notifications
		Concurrency: 1,    // Single-threaded for now
		RPCLog:      &testRPCLogger{},
		Logger:      func(text string) { fmt.Fprintf(os.Stderr, "😈 %s\n", text) },
	}

	// Create server instance
	instance := protocol.NewServerInstance(ctx, server, opts)

	// Create LSP channel
	transport := NewStdioTransport()
	ch := channel.LSP(transport.reader, transport.writer)

	// Start server
	srv, err := instance.Instance().StartAndDetach(ch)
	if err != nil {
		return errors.Errorf("starting language server: %w", err)
	}

	server.SetCallbackClient(instance.Instance().ForwardingClient())

	logger.Info().Msg("🎯 Server instance started")

	// Wait for server
	if err := srv.Wait(); err != nil {
		return errors.Errorf("server error: %w", err)
	}

	return nil
}
