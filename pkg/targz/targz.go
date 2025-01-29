package targz

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarGz extracts a tar.gz archive from a byte slice to a target directory
func ExtractTarGz(data []byte, targetDir string) error {
	return ExtractTarGzWithOptions(data, targetDir, ExtractOptions{})
}

// ExtractOptions provides configuration for the extraction process
type ExtractOptions struct {
	// StripComponents removes the specified number of leading path components
	// Similar to tar's --strip-components
	StripComponents int

	// FileMode is the mode to use for created files (defaults to 0644)
	FileMode os.FileMode

	// DirMode is the mode to use for created directories (defaults to 0755)
	DirMode os.FileMode

	// Filter allows filtering files during extraction
	// Return true to extract the file, false to skip it
	Filter func(header *tar.Header) bool
}

// ExtractTarGzWithOptions extracts a tar.gz archive with custom options
func ExtractTarGzWithOptions(data []byte, targetDir string, opts ExtractOptions) error {
	if opts.FileMode == 0 {
		opts.FileMode = 0644
	}
	if opts.DirMode == 0 {
		opts.DirMode = 0755
	}

	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Apply strip components
		components := SplitPath(header.Name)
		if len(components) <= opts.StripComponents {
			continue
		}
		strippedPath := filepath.Join(components[opts.StripComponents:]...)

		// Apply filter if provided
		if opts.Filter != nil && !opts.Filter(header) {
			continue
		}

		target := filepath.Join(targetDir, strippedPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, opts.DirMode); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, opts.DirMode); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, opts.FileMode)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			f.Close()
		}
	}

	return nil
}

// SplitPath splits a file path into components
func SplitPath(path string) []string {
	// Remove trailing slashes
	path = strings.TrimRight(path, "/")
	if path == "" {
		return nil
	}

	var components []string
	dir := path
	for dir != "." && dir != "/" && dir != "" {
		components = append([]string{filepath.Base(dir)}, components...)
		dir = filepath.Dir(dir)
	}
	return components
}

// MemoryFileStore stores file contents in memory
type MemoryFileStore struct {
	Files map[string][]byte
}

// LoadOptions provides configuration for loading files into memory
type LoadOptions struct {
	// StripComponents removes the specified number of leading path components
	// Similar to tar's --strip-components
	StripComponents int

	// Filter allows filtering files during loading
	// Return true to load the file, false to skip it
	Filter func(header *tar.Header) bool

	// Transform allows transforming the file contents before storing
	// If nil, the contents are stored as-is
	Transform func([]byte) ([]byte, error)

	// TransformName allows transforming the file name before storing
	// If nil, the original name (after stripping) is used
	// If the transformed name collides with an existing file, an error is returned
	TransformName func(string) string
}

// LoadTarGz loads a tar.gz archive into memory with default options
func LoadTarGz(data []byte) (*MemoryFileStore, error) {
	return LoadTarGzWithOptions(data, LoadOptions{})
}

// LoadTarGzWithOptions loads a tar.gz archive into memory with custom options
func LoadTarGzWithOptions(data []byte, opts LoadOptions) (*MemoryFileStore, error) {
	store := &MemoryFileStore{
		Files: make(map[string][]byte),
	}

	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		// Skip if not a regular file
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Apply strip components
		components := SplitPath(header.Name)
		if len(components) <= opts.StripComponents {
			continue
		}
		strippedPath := filepath.Join(components[opts.StripComponents:]...)

		// Apply filter if provided
		if opts.Filter != nil && !opts.Filter(header) {
			continue
		}

		// Apply name transform if provided
		finalPath := strippedPath
		if opts.TransformName != nil {
			finalPath = opts.TransformName(strippedPath)
		}

		// Check for collisions
		if _, exists := store.Files[finalPath]; exists {
			return nil, fmt.Errorf("file collision: %s (original: %s) already exists in store", finalPath, strippedPath)
		}

		// Read file contents
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, tr); err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", header.Name, err)
		}

		contents := buf.Bytes()

		// Apply transform if provided
		if opts.Transform != nil {
			contents, err = opts.Transform(contents)
			if err != nil {
				return nil, fmt.Errorf("failed to transform file %s: %w", header.Name, err)
			}
		}

		store.Files[finalPath] = contents
	}

	return store, nil
}
