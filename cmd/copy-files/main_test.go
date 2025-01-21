package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGithubProvider()
			require.NoError(t, err, "unexpected error")
			require.NotNil(t, provider, "provider should not be nil")
		})
	}
}

func TestGithubProvider(t *testing.T) {
	provider, err := NewGithubProvider()
	require.NoError(t, err, "creating provider")

	t.Run("GetSourceInfo", func(t *testing.T) {
		info, err := provider.GetSourceInfo(context.Background(), ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "path/to/files",
		}, "abc123")
		require.NoError(t, err, "getting source info")
		assert.Equal(t, "github.com/org/repo@abc123", info)
	})

	t.Run("GetPermalink", func(t *testing.T) {
		link, err := provider.GetPermalink(context.Background(), ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "path/to/files",
		}, "abc123", "file.go")
		require.NoError(t, err, "getting permalink")
		assert.Equal(t, "https://github.com/org/repo/blob/abc123/path/to/files/file.go", link)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock provider for testing
			mock := NewMockProvider()

			cfg, err := NewConfigFromInput(tt.input, mock)
			require.NoError(t, err, "unexpected error")
			require.NotNil(t, cfg, "config should not be nil")
			assert.Equal(t, tt.input.DestPath, cfg.DestPath)
			assert.Len(t, cfg.CopyArgs.Replacements, len(tt.input.Replacements))
			assert.Len(t, cfg.CopyArgs.IgnoreFiles, len(tt.input.IgnoreFiles))
		})
	}
}

func TestProcessFile(t *testing.T) {
	// Setup mock provider with test files
	mock := NewMockProvider()
	mock.AddFile("test.go", []byte(`package foo

func Bar() {}`))
	mock.AddFile("other.go", []byte(`package foo

func Other() {}`))

	args := ProviderArgs{
		Repo: mock.GetFullRepo(),
		Ref:  mock.ref,
		Path: mock.path,
	}

	cfg := &Config{
		ProviderArgs: args,
		DestPath:     t.TempDir(),
		CopyArgs: &ConfigCopyArgs{
			Replacements: []Replacement{
				{Old: "Bar", New: "Baz"},
			},
			IgnoreFiles: []string{"*.tmp", "*.bak"},
		},
	}

	// Initialize status
	status := &StatusFile{
		Entries: make(map[string]StatusEntry),
	}

	t.Run("normal file", func(t *testing.T) {
		// Process the file
		var mu sync.Mutex
		err := processFile(context.Background(), mock, cfg, "test.go", mock.commitHash, status, &mu)
		require.NoError(t, err)

		// Verify the output file
		content, err := os.ReadFile(filepath.Join(cfg.DestPath, "test.copy.go"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "func Baz()")
		assert.Contains(t, string(content), "// 📦 Generated from: mock@abc123")

		// Verify status entry
		entry, ok := status.Entries["test.copy.go"]
		require.True(t, ok, "status entry should exist")
		assert.Equal(t, "test.copy.go", entry.File)
		assert.Equal(t, "mock@abc123", entry.Source)
	})

	t.Run("file with patch", func(t *testing.T) {
		// Create a patch file
		patchPath := filepath.Join(cfg.DestPath, "other.copy.patch.go")
		require.NoError(t, os.WriteFile(patchPath, []byte("patch content"), 0644))

		// Process the file
		var mu sync.Mutex
		err := processFile(context.Background(), mock, cfg, "other.go", mock.commitHash, status, &mu)
		require.NoError(t, err)

		// Verify the file was not created
		_, err = os.Stat(filepath.Join(cfg.DestPath, "other.copy.go"))
		assert.True(t, os.IsNotExist(err), "file should not exist")
	})

	t.Run("clean destination", func(t *testing.T) {
		// Create test files
		dir := t.TempDir()
		files := []string{
			"file1.copy.go",
			"file2.copy.go",
			"file3.patch.go",
			"file4.copy.patch.go",
			"regular.go",
		}
		for _, f := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644))
		}

		// Clean the directory
		err := cleanDestination(dir)
		require.NoError(t, err)

		// Verify only .copy. files were removed
		for _, f := range files {
			path := filepath.Join(dir, f)
			exists := true
			if _, err := os.Stat(path); os.IsNotExist(err) {
				exists = false
			}

			if strings.Contains(f, ".copy.") && !strings.Contains(f, ".patch.") {
				assert.False(t, exists, "file should be removed: %s", f)
			} else {
				assert.True(t, exists, "file should exist: %s", f)
			}
		}
	})

	t.Run("status check", func(t *testing.T) {
		dir := t.TempDir()
		statusPath := filepath.Join(dir, ".copy-status")

		// Create initial status
		status := &StatusFile{
			CommitHash: mock.commitHash,
			Ref:        mock.ref,
			Args: StatusFileArgs{
				SrcRepo:  args.Repo,
				SrcRef:   args.Ref,
				SrcPath:  args.Path,
				CopyArgs: &ConfigCopyArgs{},
			},
			Entries: make(map[string]StatusEntry),
		}
		require.NoError(t, writeStatusFile(statusPath, status))

		// Test with same commit hash
		cfg := &Config{
			ProviderArgs: args,
			DestPath:     dir,
			RemoteStatus: true,
			CopyArgs:     &ConfigCopyArgs{},
		}
		err := run(cfg, mock)
		require.NoError(t, err)

		// Test with different commit hash
		status.CommitHash = "different"
		require.NoError(t, writeStatusFile(statusPath, status))
		err = run(cfg, mock)
		assert.Error(t, err)
	})

	t.Run("local_status_check", func(t *testing.T) {
		dir := t.TempDir()
		statusPath := filepath.Join(dir, ".copy-status")

		// Create initial status
		status := &StatusFile{
			Entries: make(map[string]StatusEntry),
			Args: StatusFileArgs{
				SrcRepo: args.Repo,
				SrcRef:  args.Ref,
				SrcPath: args.Path,
				CopyArgs: &ConfigCopyArgs{
					Replacements: []Replacement{
						{Old: "Bar", New: "Baz"},
					},
					IgnoreFiles: []string{"*.tmp", "*.bak"},
				},
			},
		}
		require.NoError(t, writeStatusFile(statusPath, status))

		// Test with same arguments
		cfg := &Config{
			ProviderArgs: args,
			DestPath:     dir,
			Status:       true,
			CopyArgs: &ConfigCopyArgs{
				Replacements: []Replacement{
					{Old: "Bar", New: "Baz"},
				},
				IgnoreFiles: []string{"*.tmp", "*.bak"},
			},
		}
		err := run(cfg, mock)
		require.NoError(t, err)

		// Test with different arguments
		status.Args.SrcRef = "different"
		require.NoError(t, writeStatusFile(statusPath, status))
		err = run(cfg, mock)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration has changed")

		// Test with force flag
		// cfg.Force = true
		cfg.ProviderArgs.Ref = "different"
		err = run(cfg, mock)
		require.NoError(t, err)
	})

	t.Run("argument_change_detection", func(t *testing.T) {
		dir := t.TempDir()
		statusPath := filepath.Join(dir, ".copy-status")

		// Create initial status
		status := &StatusFile{
			Entries: make(map[string]StatusEntry),
			Args: StatusFileArgs{
				SrcRepo: args.Repo,
				SrcRef:  args.Ref,
				SrcPath: args.Path,
				CopyArgs: &ConfigCopyArgs{
					Replacements: []Replacement{
						{Old: "Bar", New: "Baz"},
					},
					IgnoreFiles: []string{"*.tmp", "*.bak"},
				},
			},
		}
		require.NoError(t, writeStatusFile(statusPath, status))

		// Test with same arguments
		cfg := &Config{
			ProviderArgs: args,
			DestPath:     dir,
			Status:       true,
			CopyArgs: &ConfigCopyArgs{
				Replacements: []Replacement{
					{Old: "Bar", New: "Baz"},
				},
				IgnoreFiles: []string{"*.tmp", "*.bak"},
			},
		}
		err := run(cfg, mock)
		require.NoError(t, err)

		// Test with different arguments
		cfg.ProviderArgs.Ref = "different"
		err = run(cfg, mock)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration has changed")
	})
}

func TestStatusFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".copy-status")

	// Create test status
	status := &StatusFile{
		LastUpdated: time.Now().UTC(),
		CommitHash:  "abc123",
		Ref:         "main",
		Entries: map[string]StatusEntry{
			"test.copy.go": {
				File:       "test.copy.go",
				Source:     "mock@abc123",
				Permalink:  "mock://test.go@abc123",
				Downloaded: time.Now().UTC(),
				Changes:    []string{"test change"},
			},
		},
	}

	// Write and read back
	require.NoError(t, writeStatusFile(path, status))
	loaded, err := loadStatusFile(path)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, status.CommitHash, loaded.CommitHash)
	assert.Equal(t, status.Ref, loaded.Ref)
	assert.Len(t, loaded.Entries, 1)
	assert.Equal(t, status.Entries["test.copy.go"].File, loaded.Entries["test.copy.go"].File)
	assert.Equal(t, status.Entries["test.copy.go"].Changes, loaded.Entries["test.copy.go"].Changes)
}

func TestTarballMode(t *testing.T) {
	// Create mock provider
	mock := NewMockProvider()
	mock.AddFile("test.txt", []byte("test content"))
	mock.AddFile("dir/nested.txt", []byte("nested content"))

	// Create temp directory for cache
	cacheDir := t.TempDir()

	// Create config with tarball mode enabled
	cfg := &Config{
		ProviderArgs: ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
		},
		DestPath: cacheDir,
		ArchiveArgs: &ConfigArchiveArgs{
			GoEmbed: true,
		},
	}

	// Run the copy operation
	err := run(cfg, mock)
	require.NoError(t, err, "run should succeed")

	// Verify the directory structure
	repoDir := filepath.Join(cacheDir, "repo")
	require.DirExists(t, repoDir, "repo directory should exist")

	// Verify tarball file
	tarballPath := filepath.Join(repoDir, "repo.tar.gz")
	require.FileExists(t, tarballPath, "tarball file should exist")

	// Verify embed.go file
	embedPath := filepath.Join(repoDir, "embed.gen.go")
	require.FileExists(t, embedPath, "embed.gen.go file should exist")

	// Read and verify embed.go contents
	content, err := os.ReadFile(embedPath)
	require.NoError(t, err, "reading embed.gen.go should succeed")
	contentStr := string(content)

	// Check required elements in embed.go
	assert.Contains(t, contentStr, "package repo\n", "should have correct package name")
	assert.Contains(t, contentStr, "import _ \"embed\"\n", "should have embed import")
	assert.Contains(t, contentStr, "//go:embed repo.tar.gz", "should have embed directive")
	assert.Contains(t, contentStr, "var Data []byte", "should have Data variable")
	assert.Contains(t, contentStr, "var (", "should have metadata variables")
	assert.Contains(t, contentStr, "Ref        = \"main\"", "should have correct ref")
	assert.Contains(t, contentStr, "Repository = \"github.com/org/repo\"", "should have correct repository")

	// Verify status file
	statusPath := filepath.Join(repoDir, ".copy-status")
	require.FileExists(t, statusPath, "status file should exist at %s", statusPath)

	// Read and verify status file
	status, err := loadStatusFile(statusPath)
	require.NoError(t, err, "reading status file should succeed")

	sourceInfo, err := mock.GetSourceInfo(context.Background(), cfg.ProviderArgs, mock.commitHash)
	require.NoError(t, err, "getting source info should succeed")

	permalink, err := mock.GetArchiveUrl(context.Background(), cfg.ProviderArgs)
	require.NoError(t, err, "getting archive url should succeed")

	// Check status file entries - should only have one entry for the tarball
	require.Len(t, status.Entries, 1, "should only have one entry in status file")
	entry, ok := status.Entries["repo.tar.gz"]
	require.True(t, ok, "status file should have entry for tarball")
	assert.Equal(t, "repo.tar.gz", entry.File, "should have correct file name")
	assert.Equal(t, []string{"generated embed.gen.go file"}, entry.Changes, "should have correct changes")
	assert.Equal(t, sourceInfo, entry.Source, "should have correct source")
	assert.Equal(t, filepath.Dir(permalink), filepath.Dir(entry.Permalink), "should have correct permalink directory")
	assert.Equal(t, filepath.Ext(permalink), filepath.Ext(entry.Permalink), "should have correct permalink file ext")

	// Run again to verify caching behavior
	err = run(cfg, mock)
	require.NoError(t, err, "second run should succeed")

	// Status file should still only have one entry
	status, err = loadStatusFile(statusPath)
	require.NoError(t, err, "reading status file after second run should succeed")
	require.Len(t, status.Entries, 1, "should still only have one entry in status file")
}
