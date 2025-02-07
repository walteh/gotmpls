//go:build js && wasm

package lsp

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall/js"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/lsp"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

// JSPipe implements a bidirectional pipe for LSP communication
type JSPipe struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func NewJSPipe() *JSPipe {
	r, w := io.Pipe()
	return &JSPipe{
		reader: r,
		writer: w,
	}
}

// JSTransport implements a transport layer that communicates with JavaScript
type JSTransport struct {
	send     js.Value // JavaScript callback function to send messages
	incoming *JSPipe  // Pipe for incoming messages
	outgoing *JSPipe  // Pipe for outgoing messages
	mu       sync.Mutex
}

// NewJSTransport creates a new transport that uses JavaScript callbacks
func NewJSTransport(sendFunc js.Value, recvName string) *JSTransport {
	t := &JSTransport{
		send:     sendFunc,
		incoming: NewJSPipe(),
		outgoing: NewJSPipe(),
	}

	// Register the receive function in JavaScript
	js.Global().Set(recvName, js.FuncOf(t.Recv))

	// Start forwarding outgoing messages to JavaScript
	go t.forwardOutgoing()

	return t
}

// Send implements the transport interface
func (t *JSTransport) Send(msg []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Printf("📤 Transport sending: %s\n", string(msg))

	// Format message with LSP headers
	contentLength := len(msg)
	headers := fmt.Sprintf("Content-Length: %d\r\n\r\n", contentLength)
	fullMessage := headers + string(msg)

	_, err := t.outgoing.writer.Write([]byte(fullMessage))
	return err
}

// Recv implements the transport interface
func (t *JSTransport) Recv(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	msg := args[0].String()
	fmt.Printf("📨 Transport received from JS: %s\n", msg)

	// Format message with LSP headers
	contentLength := len(msg)
	headers := fmt.Sprintf("Content-Length: %d\r\n\r\n", contentLength)
	fullMessage := headers + msg

	// Write to pipe without holding the lock
	n, err := t.incoming.writer.Write([]byte(fullMessage))
	if err != nil {
		fmt.Printf("❌ Error writing to pipe: %v\n", err)
	} else {
		fmt.Printf("✅ Wrote %d bytes to pipe\n", n)
	}
	return nil
}

// forwardOutgoing reads from the outgoing pipe and sends to JavaScript
func (t *JSTransport) forwardOutgoing() {
	buf := make([]byte, 4096)
	for {
		fmt.Printf("🔄 Waiting for outgoing message...\n")
		n, err := t.outgoing.reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("❌ Error reading from pipe: %v\n", err)
			}
			return
		}
		if n > 0 {
			msg := string(buf[:n])
			// Strip LSP headers before sending to JS
			if idx := strings.Index(msg, "\r\n\r\n"); idx >= 0 {
				msg = msg[idx+4:]
			}
			fmt.Printf("📤 Transport forwarding to JS: %s\n", msg)
			t.send.Invoke(msg)
		}
	}
}

// GetChannelStreams returns io.ReadWriteCloser for LSP communication
func (t *JSTransport) GetChannelStreams() (io.ReadCloser, io.WriteCloser) {
	fmt.Printf("🔌 Getting channel streams...\n")
	return t.incoming.reader, t.outgoing.writer
}

// RPCLogger implements jrpc2.Logger interface
type RPCLogger struct {
	logger zerolog.Logger
}

func NewRPCLogger(logger zerolog.Logger) *RPCLogger {
	return &RPCLogger{
		logger: logger,
	}
}

func (l *RPCLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	fmt.Printf("[LSP Request] method=%s id=%v\n", req.Method(), req.ID())
	l.logger.Debug().
		Str("method", req.Method()).
		Interface("id", req.ID()).
		Msg("LSP request received")
}

func (l *RPCLogger) LogResponse(ctx context.Context, resp *jrpc2.Response) {
	fmt.Printf("[LSP Response] id=%v\n", resp.ID())
	l.logger.Debug().
		Interface("id", resp.ID()).
		Msg("LSP response sent")
}

// ServeLSP starts the LSP server with JavaScript transport
func ServeLSP(ctx context.Context, this js.Value, args []js.Value) (string, error) {
	fmt.Printf("🚀 Serving LSP with %d arguments\n", len(args))
	if len(args) < 1 {
		return "", errors.New("expected at least 1 argument: send_message")
	}

	// Extract arguments
	sendFunc := args[0]

	// Create transport with a default receive name
	transport := NewJSTransport(sendFunc, "gotmpls_receive")

	// Create logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Str("service", "lsp").Logger()
	ctx = logger.WithContext(ctx)

	// Create and start server
	server := lsp.NewServer(ctx)

	// Create server options with logging
	opts := &jrpc2.ServerOptions{
		RPCLog:      NewRPCLogger(logger),
		AllowPush:   true, // Allow server to send notifications
		Concurrency: 1,    // Single-threaded for now
	}

	// Create server instance
	instance := protocol.NewServerInstance(ctx, server, opts)

	// Get reader and writer from transport
	reader, writer := transport.GetChannelStreams()

	// Create basic channel
	ch := channel.LSP(reader, writer)
	fmt.Printf("📡 Created LSP channel\n")

	// Start server
	srv, err := instance.Instance().StartAndDetach(ch)
	if err != nil {
		return "", errors.Errorf("starting language server: %w", err)
	}
	fmt.Printf("🎯 Server instance started\n")

	// Set callback client
	server.SetCallbackClient(instance.Instance().ForwardingClient())
	fmt.Printf("🔗 Callback client set\n")

	// Wait for server in a goroutine
	go func() {
		fmt.Printf("⏳ Server wait starting\n")
		if err := srv.Wait(); err != nil {
			fmt.Printf("❌ Server error: %v\n", err)
		}
		fmt.Printf("👋 Server stopped\n")
	}()

	fmt.Printf("✨ LSP server ready\n")
	return "server started", nil
}
