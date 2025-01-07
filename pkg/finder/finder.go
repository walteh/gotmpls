package finder

import (
	"context"

	"gitlab.com/tozd/go/errors"
)

// TemplateFinder is responsible for finding template files in a directory
type TemplateFinder interface {
	// FindTemplates finds all template files in a directory that match the given extensions
	FindTemplates(ctx context.Context, dir string, extensions []string) ([]string, error)
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
func (f *DefaultFinder) FindTemplates(ctx context.Context, dir string, extensions []string) ([]string, error) {
	// TODO: Implement file finding logic
	return nil, errors.Errorf("not implemented")
}
