package archive

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
		components := splitPath(header.Name)
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

// splitPath splits a file path into components
func splitPath(path string) []string {
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
