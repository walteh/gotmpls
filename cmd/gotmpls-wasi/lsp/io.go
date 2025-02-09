package lsp

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

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
	send     func(msg string)
	incoming *JSPipe // Pipe for incoming messages
	outgoing *JSPipe // Pipe for outgoing messages
	mu       sync.Mutex
}

// NewJSTransport creates a new transport that uses JavaScript callbacks
func NewJSTransport(sendFunc func(msg string)) *JSTransport {
	t := &JSTransport{
		send:     sendFunc,
		incoming: NewJSPipe(),
		outgoing: NewJSPipe(),
	}

	// // Register the receive function in JavaScript
	// js.Global().Set(recvName, js.FuncOf(t.Recv))

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

//go:wasmimport wasi_snapshot_preview1 yo_send
func yo_send(msg string)

var currentTransport *JSTransport

// Recv implements the transport interface
//
//go:wasmexport "yo_recv"
func yo_recv(msg string) {
	fmt.Printf("📨 Transport received from JS: %s\n", msg)

	// Format message with LSP headers
	contentLength := len(msg)
	headers := fmt.Sprintf("Content-Length: %d\r\n\r\n", contentLength)
	fullMessage := headers + msg

	// Write to pipe without holding the lock
	n, err := currentTransport.incoming.writer.Write([]byte(fullMessage))
	if err != nil {
		fmt.Printf("❌ Error writing to pipe: %v\n", err)
	} else {
		fmt.Printf("✅ Wrote %d bytes to pipe\n", n)
	}
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
			t.send(msg)
		}
	}
}

// GetChannelStreams returns io.ReadWriteCloser for LSP communication
func (t *JSTransport) GetChannelStreams() (io.ReadCloser, io.WriteCloser) {
	fmt.Printf("🔌 Getting channel streams...\n")
	return t.incoming.reader, t.outgoing.writer
}
