package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		ref         string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_github_repo",
			repo: "github.com/org/repo",
			ref:  "main",
			path: "path/to/files",
		},
		{
			name:        "invalid_github_repo",
			repo:        "github.com/org",
			ref:         "main",
			path:        "path/to/files",
			wantErr:     true,
			errContains: "invalid github repository format",
		},
		{
			name:        "unsupported_provider",
			repo:        "gitlab.com/org/repo",
			ref:         "main",
			path:        "path/to/files",
			wantErr:     true,
			errContains: "unsupported repository host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.repo, tt.ref, tt.path)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err, "unexpected error")
			require.NotNil(t, provider, "provider should not be nil")
		})
	}
}

func TestGithubProvider(t *testing.T) {
	provider, err := NewGithubProvider("github.com/org/repo", "main", "path/to/files")
	require.NoError(t, err, "creating provider")

	t.Run("GetSourceInfo", func(t *testing.T) {
		info := provider.GetSourceInfo("abc123")
		assert.Equal(t, "github.com/org/repo@abc123", info)
	})

	t.Run("GetPermalink", func(t *testing.T) {
		link := provider.GetPermalink("file.go", "abc123")
		assert.Equal(t, "https://github.com/org/repo/blob/abc123/file.go", link)
	})
}

func TestNewConfigFromInput(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_input",
			input: Input{
				SrcRepo:  "github.com/org/repo",
				SrcRef:   "main",
				SrcPath:  "path/to/files",
				DestPath: "/tmp/dest",
				Replacements: []string{
					"old:new",
					"foo:bar",
				},
				IgnoreFiles: []string{
					"*.tmp",
					"*.bak",
				},
			},
		},
		{
			name: "invalid_repo",
			input: Input{
				SrcRepo:  "github.com/org",
				SrcRef:   "main",
				SrcPath:  "path/to/files",
				DestPath: "/tmp/dest",
			},
			wantErr:     true,
			errContains: "invalid github repository format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigFromInput(tt.input)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err, "unexpected error")
			require.NotNil(t, cfg, "config should not be nil")
			require.NotNil(t, cfg.Provider, "provider should not be nil")
			assert.Equal(t, tt.input.DestPath, cfg.DestPath)
			assert.Len(t, cfg.Replacements, len(tt.input.Replacements))
			assert.Len(t, cfg.IgnoreFiles, len(tt.input.IgnoreFiles))
		})
	}
}

// 🧪 Mock provider for testing
type mockProvider struct {
	files      map[string][]byte
	commitHash string
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		files:      make(map[string][]byte),
		commitHash: "abc123",
	}
}

func (m *mockProvider) ListFiles(ctx context.Context) ([]string, error) {
	files := make([]string, 0, len(m.files))
	for f := range m.files {
		files = append(files, f)
	}
	return files, nil
}

func (m *mockProvider) GetFile(ctx context.Context, path string) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return content, nil
}

func (m *mockProvider) GetCommitHash(ctx context.Context) (string, error) {
	return m.commitHash, nil
}

func (m *mockProvider) GetPermalink(path, commitHash string) string {
	return "mock://" + path + "@" + commitHash
}

func (m *mockProvider) GetSourceInfo(commitHash string) string {
	return "mock@" + commitHash
}

func TestProcessFile(t *testing.T) {
	// Setup mock provider with test files
	mock := newMockProvider()
	mock.files["test.go"] = []byte(`package foo

func Bar() {}`)
	mock.files["other.go"] = []byte(`package foo

func Other() {}`)

	cfg := &Config{
		Provider: mock,
		DestPath: t.TempDir(),
		Replacements: []Replacement{
			{Old: "Bar", New: "Baz"},
		},
	}

	// Create temporary status file
	statusFile := filepath.Join(cfg.DestPath, ".copy-status")
	require.NoError(t, initStatusFile(statusFile))

	t.Run("normal file", func(t *testing.T) {
		// Process the file
		var mu sync.Mutex
		err := processFile(context.Background(), cfg, "test.go", mock.commitHash, statusFile, &mu)
		require.NoError(t, err)

		// Verify the output file
		content, err := os.ReadFile(filepath.Join(cfg.DestPath, "test.copy.go"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "func Baz()")
		assert.Contains(t, string(content), "// 📦 Generated from: mock@abc123")
	})

	t.Run("file with patch", func(t *testing.T) {
		// Create a patch file
		patchPath := filepath.Join(cfg.DestPath, "other.copy.patch.go")
		require.NoError(t, os.WriteFile(patchPath, []byte("patch content"), 0644))

		// Process the file
		var mu sync.Mutex
		err := processFile(context.Background(), cfg, "other.go", mock.commitHash, statusFile, &mu)
		require.NoError(t, err)

		// Verify the file was not created
		_, err = os.Stat(filepath.Join(cfg.DestPath, "other.copy.go"))
		assert.True(t, os.IsNotExist(err), "file should not exist")
	})

	t.Run("file with patch already exists", func(t *testing.T) {
		// Create both patch and copy files
		patchPath := filepath.Join(cfg.DestPath, "both.copy.patch.go")
		copyPath := filepath.Join(cfg.DestPath, "both.copy.go")
		require.NoError(t, os.WriteFile(patchPath, []byte("patch content"), 0644))
		require.NoError(t, os.WriteFile(copyPath, []byte("copy content"), 0644))

		// Add file to mock
		mock.files["both.go"] = []byte(`package foo

func Both() {}`)

		// Process the file
		var mu sync.Mutex
		err := processFile(context.Background(), cfg, "both.go", mock.commitHash, statusFile, &mu)
		require.NoError(t, err)

		// Verify the copy file was not modified
		content, err := os.ReadFile(copyPath)
		require.NoError(t, err)
		assert.Equal(t, "copy content", string(content), "copy file should not be modified when patch exists")
	})
}
