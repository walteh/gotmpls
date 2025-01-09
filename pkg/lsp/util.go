package lsp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"gitlab.com/tozd/go/errors"
)

// ReadWriteCloser combines an io.ReadCloser and io.WriteCloser into a single io.ReadWriteCloser
type ReadWriteCloser struct {
	reader *bufio.Reader
	writer *bufio.Writer
	closer multiCloser
	mu     sync.Mutex
	buf    bytes.Buffer
}

type multiCloser struct {
	closers []io.Closer
}

func (mc multiCloser) Close() error {
	var firstErr error
	for _, c := range mc.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewReadWriteCloser creates a new ReadWriteCloser from separate read and write closers
func NewReadWriteCloser(r io.ReadCloser, w io.WriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{
		reader: bufio.NewReader(r),
		writer: bufio.NewWriter(w),
		closer: multiCloser{closers: []io.Closer{r, w}},
	}
}

// Read reads data from the underlying reader
func (rwc *ReadWriteCloser) Read(p []byte) (int, error) {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()
	return rwc.reader.Read(p)
}

// Write writes data to the underlying writer
func (rwc *ReadWriteCloser) Write(p []byte) (int, error) {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()

	// Write to our temporary buffer first
	n, err := rwc.buf.Write(p)
	if err != nil {
		return n, err
	}

	// Process all complete messages
	for {
		data := rwc.buf.Bytes()

		// Look for Content-Length header
		headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
		if headerEnd == -1 {
			break // No complete header found
		}

		header := data[:headerEnd]
		if !bytes.Contains(header, []byte("Content-Length: ")) {
			break // Invalid header
		}

		contentLengthStr := bytes.TrimPrefix(bytes.TrimSpace(header), []byte("Content-Length: "))
		contentLength := 0
		_, err := fmt.Sscanf(string(contentLengthStr), "%d", &contentLength)
		if err != nil {
			break // Invalid Content-Length
		}

		// Check if we have the complete message
		totalLength := headerEnd + 4 + contentLength // header + \r\n\r\n + content
		if len(data) < totalLength {
			break // Don't have complete message yet
		}

		// Write the complete message
		message := data[:totalLength]
		_, err = rwc.writer.Write(message)
		if err != nil {
			return n, err
		}

		// Flush after each complete message
		err = rwc.writer.Flush()
		if err != nil {
			return n, err
		}

		// Remove the processed message from buffer
		rwc.buf.Next(totalLength)

		// If there's no more data, break
		if rwc.buf.Len() == 0 {
			break
		}
	}

	return n, nil
}

// Close closes both the reader and writer
func (rwc *ReadWriteCloser) Close() error {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()

	// Write any remaining data
	if rwc.buf.Len() > 0 {
		_, err := rwc.writer.Write(rwc.buf.Bytes())
		if err != nil {
			return err
		}
	}

	// Flush any remaining data
	if err := rwc.writer.Flush(); err != nil {
		return err
	}

	return rwc.closer.Close()
}

// uriToPath converts a URI to a filesystem path
func uriToPath(uri string) (string, error) {
	if !strings.HasPrefix(uri, "file://") {
		return "", errors.Errorf("unsupported URI scheme: %s", uri)
	}

	// Parse the URI
	u, err := url.Parse(uri)
	if err != nil {
		return "", errors.Errorf("failed to parse URI: %w", err)
	}

	// Convert the path to a filesystem path
	path := u.Path
	if path == "" {
		return "", errors.Errorf("empty path in URI: %s", uri)
	}

	// On Windows, remove the leading slash
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Clean the path
	path = filepath.Clean(path)

	return path, nil
}
