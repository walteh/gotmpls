//go:build wasip1

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/walteh/gotmpls/cmd/gotmpls-wasi/lsp"
)

func main() {
	fmt.Fprintf(os.Stderr, "🚀 Starting WASI timeout-based test...\n")

	ctx := context.Background()

	if err := lsp.ServeLSP(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error serving LSP: %v\n", err)
		os.Exit(1)
	}
}

// type NonBlockingReader struct {
// 	input     io.Reader
// 	buffer    chan []byte
// 	errChan   chan error
// 	closeOnce sync.Once
// }

// func NewNonBlockingReader(input io.Reader) *NonBlockingReader {
// 	r := &NonBlockingReader{
// 		input:   input,
// 		buffer:  make(chan []byte, 100),
// 		errChan: make(chan error, 1),
// 	}
// 	go r.readLoop()
// 	return r
// }

// func (r *NonBlockingReader) readLoop() {
// 	fmt.Fprintf(os.Stderr, "🔄 Starting WASI read loop\n")
// 	var totalBytes int64
// 	var pollCount int64
// 	var emptyReads int64

// 	// Start with large chunks for initial LSP message
// 	chunkSize := 1024

// 	// Debug info about stdin
// 	if f, ok := r.input.(*os.File); ok {
// 		fmt.Fprintf(os.Stderr, "📋 WASI Stdin info: fd=%d, name=%s\n", f.Fd(), f.Name())
// 	}

// 	for {
// 		chunk := make([]byte, chunkSize)

// 		// Direct WASI read from stdin (fd 0)
// 		n, err := syscall.Read(0, chunk)

// 		if err != nil {
// 			if err == syscall.EAGAIN {
// 				runtime.Gosched()
// 				continue
// 			}
// 			fmt.Fprintf(os.Stderr, "❌ WASI read error: %v\n", err)
// 			r.errChan <- err
// 			return
// 		}

// 		if n > 0 {
// 			emptyReads = 0
// 			totalBytes += int64(n)
// 			data := make([]byte, n)
// 			copy(data, chunk[:n])

// 			fmt.Fprintf(os.Stderr, "📦 WASI read: %d bytes [total: %d]\n", n, totalBytes)

// 			select {
// 			case r.buffer <- data:
// 			default:
// 				runtime.Gosched()
// 			}

// 			if chunkSize > 1 {
// 				fmt.Fprintf(os.Stderr, "📉 Switching to smaller chunks\n")
// 				chunkSize = 1
// 			}
// 		} else {
// 			emptyReads++
// 			if emptyReads%100 == 0 {
// 				fmt.Fprintf(os.Stderr, "⏳ Empty reads: %d\n", emptyReads)
// 			}
// 			runtime.Gosched()
// 			time.Sleep(time.Millisecond)
// 		}

// 		pollCount++
// 		if pollCount%1000 == 0 {
// 			fmt.Fprintf(os.Stderr, "🔄 WASI loop: poll=%d, bytes=%d, empty=%d\n",
// 				pollCount, totalBytes, emptyReads)
// 		}
// 	}
// }

// func (r *NonBlockingReader) Read(p []byte) (n int, err error) {
// 	select {
// 	case err := <-r.errChan:
// 		return 0, err
// 	case data := <-r.buffer:
// 		n = copy(p, data)
// 		return n, nil
// 	case <-time.After(time.Millisecond):
// 		return 0, nil
// 	}
// }

// func main() {
// 	fmt.Fprintf(os.Stderr, "🚀 Starting WASI timeout-based test...\n")

// 	reader := NewNonBlockingReader(os.Stdin)

// 	// Ultra aggressive timer
// 	go func() {
// 		count := 0
// 		for {
// 			count++
// 			if count%100 == 0 {
// 				fmt.Fprintf(os.Stderr, "⏰ Timer tick %d\n", count)
// 			}
// 			runtime.Gosched()
// 		}
// 	}()

// 	// Ultra aggressive writer
// 	go func() {
// 		count := 0
// 		for {
// 			count++
// 			if count%100 == 0 {
// 				fmt.Fprintf(os.Stderr, "📝 Writer tick %d\n", count)
// 			}
// 			runtime.Gosched()
// 		}
// 	}()

// 	// Main loop
// 	buf := make([]byte, 256)
// 	var totalRead int64
// 	for {
// 		n, err := reader.Read(buf)
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "❌ Error reading: %v\n", err)
// 			os.Exit(1)
// 		}
// 		if n > 0 {
// 			totalRead += int64(n)
// 			fmt.Fprintf(os.Stderr, "📥 Read %d bytes [total: %d]\n", n, totalRead)
// 		}
// 		runtime.Gosched()
// 	}
// }
