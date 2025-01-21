package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/tozd/go/errors"
)

// 🧪 Mock provider for testing
type MockProvider struct {
	files      map[string][]byte
	commitHash string
	ref        string
	org        string
	repo       string
	path       string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		files:      make(map[string][]byte), // Create a new map for each instance
		commitHash: "abc123",
		ref:        "main",
		org:        "org",
		repo:       "repo",
		path:       "path/to/files",
	}
}

// Helper methods for testing
func (m *MockProvider) AddFile(name string, content []byte) {
	m.files[name] = content
}

func (m *MockProvider) ClearFiles() {
	m.files = make(map[string][]byte)
}

func (m *MockProvider) ListFiles(ctx context.Context, args ProviderArgs) ([]string, error) {
	// Return all files in the map
	files := make([]string, 0, len(m.files))
	for f := range m.files {
		files = append(files, f)
	}
	fmt.Printf("🧪 Mock provider listing %d files: %v\n", len(files), files)
	return files, nil
}

func (m *MockProvider) GetFile(ctx context.Context, args ProviderArgs, path string) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return content, nil
}

func (m *MockProvider) GetCommitHash(ctx context.Context, args ProviderArgs) (string, error) {
	return m.commitHash, nil
}

func (m *MockProvider) GetPermalink(args ProviderArgs, commitHash string, file string) string {
	return "mock://" + file + "@" + commitHash
}

func (m *MockProvider) GetSourceInfo(args ProviderArgs, commitHash string) string {
	return "mock@" + commitHash
}

func (m *MockProvider) GetFullRepo() string {
	return "github.com/" + m.org + "/" + m.repo
}

// GetArchiveUrl returns a mock URL for testing
func (m *MockProvider) GetArchiveUrl(ctx context.Context, args ProviderArgs) (string, error) {
	// Create a temporary file with the archive data
	data, err := m.GetArchiveData()
	if err != nil {
		return "", errors.Errorf("creating archive data: %w", err)
	}

	// Create a temporary file
	f, err := os.CreateTemp("", "mock-archive-*.tar.gz")
	if err != nil {
		return "", errors.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	// Write the archive data
	if _, err := f.Write(data); err != nil {
		return "", errors.Errorf("writing archive data: %w", err)
	}

	// Return a file:// URL to the temporary file
	return "file://" + f.Name(), nil
}

// GetArchiveData returns a mock archive for testing
func (m *MockProvider) GetArchiveData() ([]byte, error) {
	// Create a tar.gz archive in memory
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add each file to the archive
	for name, content := range m.files {
		// Create tar header
		header := &tar.Header{
			Name:    filepath.Join(m.path, name), // Include the path prefix
			Size:    int64(len(content)),
			Mode:    0644,
			ModTime: time.Now(),
		}

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return nil, errors.Errorf("writing tar header: %w", err)
		}

		// Write content
		if _, err := tw.Write(content); err != nil {
			return nil, errors.Errorf("writing tar content: %w", err)
		}
	}

	// Close writers
	if err := tw.Close(); err != nil {
		return nil, errors.Errorf("closing tar writer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, errors.Errorf("closing gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
