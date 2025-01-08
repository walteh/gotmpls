package lsp

import (
	"bufio"
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
	n, err := rwc.writer.Write(p)
	if err != nil {
		return n, err
	}
	err = rwc.writer.Flush()
	if err != nil {
		return n, err
	}
	return n, nil
}

// Close closes both the reader and writer
func (rwc *ReadWriteCloser) Close() error {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()
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
