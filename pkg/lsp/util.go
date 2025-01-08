package lsp

import "io"

// ReadWriteCloser combines an io.ReadCloser and io.WriteCloser into a single io.ReadWriteCloser
type ReadWriteCloser struct {
	io.ReadCloser
	io.WriteCloser
}

// NewReadWriteCloser creates a new ReadWriteCloser from separate read and write closers
func NewReadWriteCloser(r io.ReadCloser, w io.WriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{r, w}
}

// Close closes both the reader and writer
func (rwc *ReadWriteCloser) Close() error {
	err1 := rwc.ReadCloser.Close()
	err2 := rwc.WriteCloser.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
