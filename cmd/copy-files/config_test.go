package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *CopyConfig)
	}{
		{
			name: "valid_config",
			config: `
default_branch: main
status_file: .copy-status
copies:
  - source:
      repo: github.com/org/repo
      ref: main
      path: path/to/files
    destination:
      path: ./pkg/dest
    options:
      replacements:
        - "old:new"
        - this: foo
          with: bar
      ignore_files:
        - "*.tmp"
        - "*.bak"
  - source:
      repo: github.com/other/repo
      path: other/path
    destination:
      path: ./pkg/other
`,
			validate: func(t *testing.T, cfg *CopyConfig) {
				require.Len(t, cfg.Copies, 2)
				assert.Equal(t, "main", cfg.DefaultBranch)
				assert.Equal(t, ".copy-status", cfg.StatusFile)

				// First copy entry
				assert.Equal(t, "github.com/org/repo", cfg.Copies[0].Source.Repo)
				assert.Equal(t, "main", cfg.Copies[0].Source.Ref)
				assert.Equal(t, "path/to/files", cfg.Copies[0].Source.Path)
				assert.Equal(t, "./pkg/dest", cfg.Copies[0].Destination.Path)

				// Check replacements (converted to string format)
				require.Len(t, cfg.Copies[0].Options.Replacements, 2)
				assert.Equal(t, "old:new", cfg.Copies[0].Options.Replacements[0])
				assert.Equal(t, "foo:bar", cfg.Copies[0].Options.Replacements[1])

				assert.Equal(t, []string{"*.tmp", "*.bak"}, cfg.Copies[0].Options.IgnoreFiles)

				// Second copy entry
				assert.Equal(t, "github.com/other/repo", cfg.Copies[1].Source.Repo)
				assert.Equal(t, "main", cfg.Copies[1].Source.Ref) // Default branch
				assert.Equal(t, "other/path", cfg.Copies[1].Source.Path)
				assert.Equal(t, "./pkg/other", cfg.Copies[1].Destination.Path)
			},
		},
		{
			name: "valid_config_with_fallback",
			config: `
default_branch: main
fallback_branch: develop
status_file: .copy-status
copies:
  - source:
      repo: github.com/org/repo
      ref: main
      path: path/to/files
      fallback_branch: master
    destination:
      path: ./pkg/dest
    options:
      replacements:
        - "old:new"
        - this: xyz
          with: yyz
      ignore_files:
        - "*.tmp"
        - "*.bak"
`,
			validate: func(t *testing.T, cfg *CopyConfig) {
				require.Len(t, cfg.Copies, 1)
				assert.Equal(t, "main", cfg.DefaultBranch)
				assert.Equal(t, "develop", cfg.FallbackBranch)
				assert.Equal(t, ".copy-status", cfg.StatusFile)

				// Check source config
				assert.Equal(t, "github.com/org/repo", cfg.Copies[0].Source.Repo)
				assert.Equal(t, "main", cfg.Copies[0].Source.Ref)
				assert.Equal(t, "path/to/files", cfg.Copies[0].Source.Path)
				assert.Equal(t, "master", cfg.Copies[0].Source.FallbackBranch)

				// Check replacements
				require.Len(t, cfg.Copies[0].Options.Replacements, 2)
				assert.Equal(t, "old:new", cfg.Copies[0].Options.Replacements[0])
				// The structured replacement should be converted to string format
				assert.Equal(t, "xyz:yyz", cfg.Copies[0].Options.Replacements[1])
			},
		},
		{
			name: "missing_required_fields",
			config: `
copies:
  - source:
      repo: github.com/org/repo
    destination:
      path: ./pkg/dest
`,
			wantErr:     true,
			errContains: "source path is required",
		},
		{
			name: "no_copies",
			config: `
default_branch: main
status_file: .copy-status
`,
			wantErr:     true,
			errContains: "no copy entries defined",
		},
		{
			name: "invalid_yaml",
			config: `
default_branch: main
copies:
  - source:
    - invalid
`,
			wantErr:     true,
			errContains: "parsing config file",
		},
		{
			name: "invalid_replacement_format",
			config: `
copies:
  - source:
      repo: github.com/org/repo
      path: path/to/files
    destination:
      path: ./pkg/dest
    options:
      replacements:
        - this: foo
          invalid: field
`,
			wantErr:     true,
			errContains: "must have 'this' and 'with' as strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			dir := t.TempDir()
			path := filepath.Join(dir, ".copyrc")
			require.NoError(t, os.WriteFile(path, []byte(tt.config), 0644))

			// Load config
			cfg, err := LoadConfig(path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestRunAll(t *testing.T) {
	// Setup mock provider with test files
	mock := NewMockProvider()
	mock.ClearFiles() // Start with a clean slate
	mock.AddFile("test.go", []byte(`package foo

func Bar() {}`))
	mock.AddFile("other.go", []byte(`package foo

func Other() {}`))

	// Create test directories
	dir := t.TempDir()
	dest1 := filepath.Join(dir, "dest1")
	dest2 := filepath.Join(dir, "dest2")
	require.NoError(t, os.MkdirAll(dest1, 0755))
	require.NoError(t, os.MkdirAll(dest2, 0755))

	// Create config
	cfg := &CopyConfig{
		DefaultBranch:  "main",
		FallbackBranch: "master",
		StatusFile:     ".copy-status",
		Copies: []CopyEntry{
			{
				Source: struct {
					Repo           string "yaml:\"repo\""
					Ref            string "yaml:\"ref\""
					Path           string "yaml:\"path\""
					FallbackBranch string "yaml:\"fallback_branch,omitempty\""
				}{
					Repo: mock.GetFullRepo(),
					Ref:  mock.ref,
					Path: mock.path,
				},
				Destination: struct {
					Path string "yaml:\"path\""
				}{
					Path: dest1,
				},
				Options: struct {
					Replacements []interface{} "yaml:\"replacements,omitempty\""
					IgnoreFiles  []string      "yaml:\"ignore_files,omitempty\""
				}{
					Replacements: []interface{}{
						"Bar:Baz",
						map[string]interface{}{
							"this": "foo",
							"with": "bar",
						},
					},
				},
			},
			{
				Source: struct {
					Repo           string "yaml:\"repo\""
					Ref            string "yaml:\"ref\""
					Path           string "yaml:\"path\""
					FallbackBranch string "yaml:\"fallback_branch,omitempty\""
				}{
					Repo: mock.GetFullRepo(),
					Path: mock.path,
				},
				Destination: struct {
					Path string "yaml:\"path\""
				}{
					Path: dest2,
				},
			},
		},
	}

	// Run all copies
	require.NoError(t, cfg.RunAll(false, false, false, false))

	// Verify first copy
	content, err := os.ReadFile(filepath.Join(dest1, "test.copy.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "func Baz()")

	// Verify second copy
	content, err = os.ReadFile(filepath.Join(dest2, "test.copy.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "func Bar()")

	// Test status check
	require.NoError(t, cfg.RunAll(false, true, false, false))

	// Test remote status check
	require.NoError(t, cfg.RunAll(false, false, true, false))

	// Create a patch file to test clean behavior
	patchPath := filepath.Join(dest1, "test.copy.patch.go")
	require.NoError(t, os.WriteFile(patchPath, []byte("patch content"), 0644))

	// Test clean
	require.NoError(t, cfg.RunAll(true, false, false, false))
	_, err = os.Stat(filepath.Join(dest1, "test.copy.go"))
	assert.True(t, os.IsNotExist(err), "file should be removed by clean")
	_, err = os.Stat(patchPath)
	assert.NoError(t, err, "patch file should still exist")

	// Clean up
	mock.ClearFiles()
}
