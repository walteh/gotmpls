package finder

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/tozd/go/errors"
)

// TemplateFinder is responsible for finding template files in a directory
type TemplateFinder interface {
	// FindTemplates finds all template files in a directory that match the given extensions
	FindTemplates(ctx context.Context, dir string, extensions []string) ([]FileInfo, error)
}

// FileInfo represents information about a found template file
type FileInfo struct {
	Path     string
	Content  []byte
	FileType string
}

// DefaultFinder is the default implementation of TemplateFinder
type DefaultFinder struct{}

// NewDefaultFinder creates a new DefaultFinder
func NewDefaultFinder() *DefaultFinder {
	return &DefaultFinder{}
}

// FindTemplates implements TemplateFinder
func (f *DefaultFinder) FindTemplates(ctx context.Context, dir string, extensions []string) ([]FileInfo, error) {
	if len(extensions) == 0 {
		extensions = []string{".tmpl", ".gotmpl"}
	}

	var files []FileInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Errorf("error accessing path %s: %w", path, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		for _, ext := range extensions {
			if strings.HasSuffix(path, ext) {
				content, err := os.ReadFile(path)
				if err != nil {
					return errors.Errorf("error reading file %s: %w", path, err)
				}

				files = append(files, FileInfo{
					Path:     path,
					Content:  content,
					FileType: ext[1:], // Remove the dot
				})
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, errors.Errorf("error walking directory %s: %w", dir, err)
	}

	return files, nil
}
