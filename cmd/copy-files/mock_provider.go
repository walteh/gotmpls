package main

import (
	"context"

	"gitlab.com/tozd/go/errors"
)

// 🧪 Mock provider for testing
var mockFiles = make(map[string][]byte)

type MockProvider struct {
	files      map[string][]byte
	commitHash string
	ref        string
	failHash   bool
	org        string
	repo       string
	path       string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		files:      mockFiles,
		commitHash: "abc123",
		ref:        "main",
		org:        "org",
		repo:       "repo",
		path:       "path/to/files",
	}
}

func (m *MockProvider) AddFile(path string, content []byte) {
	mockFiles[path] = content
}

func (m *MockProvider) ClearFiles() {
	mockFiles = make(map[string][]byte)
}

func (m *MockProvider) ListFiles(ctx context.Context) ([]string, error) {
	files := make([]string, 0, len(m.files))
	for f := range m.files {
		files = append(files, f)
	}
	return files, nil
}

func (m *MockProvider) GetFile(ctx context.Context, path string) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return content, nil
}

func (m *MockProvider) GetCommitHash(ctx context.Context) (string, error) {
	if m.failHash {
		return "", errors.New("simulated error")
	}
	return m.commitHash, nil
}

func (m *MockProvider) GetPermalink(path, commitHash string) string {
	return "mock://" + path + "@" + commitHash
}

func (m *MockProvider) GetSourceInfo(commitHash string) string {
	return "mock@" + commitHash
}

func (m *MockProvider) GetFullRepo() string {
	return "github.com/" + m.org + "/" + m.repo
}
