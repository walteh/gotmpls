package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		validate    func(t *testing.T, cfg *CopyConfig)
	}{
		{
			name: "valid_config",
			config: `
copies:
  - source:
      repo: org/repo
      ref: main
      path: /src
    destination:
      path: /dest
    options:
      replacements:
        - {old: "foo", new: "bar"}
        - {old: "xyz", new: "yyz"}
      ignore_files:
        - "*.txt"
`,
			validate: func(t *testing.T, cfg *CopyConfig) {
				require.Len(t, cfg.Copies, 1)
				require.Equal(t, "org/repo", cfg.Copies[0].Source.Repo)
				require.Equal(t, "main", cfg.Copies[0].Source.Ref)
				require.Equal(t, "/src", cfg.Copies[0].Source.Path)
				require.Equal(t, "/dest", cfg.Copies[0].Destination.Path)
				require.NotNil(t, cfg.Copies[0].Options)
				require.Len(t, cfg.Copies[0].Options.Replacements, 2)
				require.Equal(t, "foo", cfg.Copies[0].Options.Replacements[0].Old)
				require.Equal(t, "bar", cfg.Copies[0].Options.Replacements[0].New)
				require.Equal(t, "xyz", cfg.Copies[0].Options.Replacements[1].Old)
				require.Equal(t, "yyz", cfg.Copies[0].Options.Replacements[1].New)
				require.Len(t, cfg.Copies[0].Options.IgnoreFiles, 1)
				require.Equal(t, "*.txt", cfg.Copies[0].Options.IgnoreFiles[0])
			},
		},

		{
			name: "invalid_replacement_format",
			config: `
copies:
  - source:
      repo: org/repo
      path: /src
    destination:
      path: /dest
    options:
      replacements:
        - "invalid"
`,
			expectError: true,
			validate: func(t *testing.T, cfg *CopyConfig) {
				require.Error(t, errors.New("copy entry 0, replacement 0: invalid format"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			cfg, err := LoadConfig(configPath)
			if tt.expectError {
				require.Error(t, err)
				if tt.validate != nil {
					tt.validate(t, nil)
				}
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
	mock := NewMockProvider(t)
	mock.ClearFiles() // Start with a clean slate
	mock.AddFile("test.go", []byte(`package foo

func Bar() {}`))
	mock.AddFile("other.go", []byte(`package foo

func Other() {}`))

	ctx := context.Background()
	logger := newTestLogger(t)
	ctx = NewLoggerInContext(ctx, logger)

	// Create test directories
	dir := t.TempDir()
	dest1 := filepath.Join(dir, "dest1")
	dest2 := filepath.Join(dir, "dest2")
	require.NoError(t, os.MkdirAll(dest1, 0755))
	require.NoError(t, os.MkdirAll(dest2, 0755))

	// Create config
	cfg := &CopyConfig{
		Defaults: &DefaultsBlock{},
		Copies: []*CopyEntry{
			{
				Source: CopyEntry_Source{
					Repo: mock.GetFullRepo(),
					Ref:  mock.ref,
					Path: mock.path,
				},
				Destination: CopyEntry_Destination{
					Path: dest1,
				},
				Options: CopyEntry_Options{
					Replacements: []Replacement{
						{Old: "Bar", New: "Baz"},
						{Old: "foo", New: "bar"},
					},
					IgnoreFiles: []string{"*.tmp", "*.bak"},
				},
			},
			{
				Source: CopyEntry_Source{
					Repo: mock.GetFullRepo(),
					Ref:  mock.ref,
					Path: mock.path,
				},
				Destination: CopyEntry_Destination{
					Path: dest2,
				},
			},
		},
	}

	// Run all copies
	require.NoError(t, cfg.RunAll(ctx, false, false, false, false, mock))

	// list out files in tmp dir
	files, err := os.ReadDir(dest1)
	t.Logf("files: %+v", files)
	require.NoError(t, err)

	// Verify first copy
	p1 := filepath.Join(dest1, "test.copy.go")
	require.FileExists(t, p1)
	content, err := os.ReadFile(p1)
	require.NoError(t, err)
	assert.Contains(t, string(content), "func Baz()")

	// Verify second copy
	p2 := filepath.Join(dest2, "test.copy.go")
	require.FileExists(t, p2)
	content, err = os.ReadFile(p2)
	require.NoError(t, err)
	assert.Contains(t, string(content), "func Bar()")

	// Test status check
	require.NoError(t, cfg.RunAll(ctx, false, true, false, false, mock))

	// Test remote status check
	require.NoError(t, cfg.RunAll(ctx, false, false, true, false, mock))

	// Create a patch file to test clean behavior
	patchPath := filepath.Join(dest1, "test.copy.patch.go")
	require.NoError(t, os.WriteFile(patchPath, []byte("patch content"), 0644))

	// Create initial status
	status := &StatusFile{
		CommitHash: mock.commitHash,
		Ref:        mock.ref,
		Args: StatusFileArgs{
			SrcRepo:  mock.GetFullRepo(),
			SrcRef:   mock.ref,
			SrcPath:  mock.path,
			CopyArgs: &ConfigCopyArgs{},
		},
		CoppiedFiles: make(map[string]StatusEntry),
	}
	require.NoError(t, writeStatusFile(ctx, status, dir))

	// Test clean
	require.NoError(t, cfg.RunAll(ctx, true, false, false, false, mock))
	require.NoFileExists(t, filepath.Join(dest1, "test.copy.go"))
	require.NoFileExists(t, filepath.Join(dest1, ".copyrc.lock"))
	require.FileExists(t, patchPath)
}

func TestLoadHCLConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		validate    func(t *testing.T, cfg *CopyConfig)
	}{
		{
			name: "valid_hcl_config",
			config: `

# Default settings
defaults {
}

# Copy configuration
copy {
  source {
    repo = "org/repo"
    ref = "main"
    path = "/src"
  }
  destination {
    path = "/dest"
  }
  options {
    replacements = [
		{old: "foo", new: "bar"},
		{old: "xyz", new: "yyz"},
    ]
    ignore_files = [
      "*.txt"
    ]
  }
}
`,
			validate: func(t *testing.T, cfg *CopyConfig) {
				require.NotNil(t, cfg.Defaults)
				require.Len(t, cfg.Copies, 1)
				require.Equal(t, "org/repo", cfg.Copies[0].Source.Repo)
				require.Equal(t, "main", cfg.Copies[0].Source.Ref)
				require.Equal(t, "/src", cfg.Copies[0].Source.Path)
				require.Equal(t, "/dest", cfg.Copies[0].Destination.Path)
				require.NotNil(t, cfg.Copies[0].Options)
				require.Len(t, cfg.Copies[0].Options.Replacements, 2)
				require.Equal(t, "foo", cfg.Copies[0].Options.Replacements[0].Old)
				require.Equal(t, "bar", cfg.Copies[0].Options.Replacements[0].New)
				require.Equal(t, "xyz", cfg.Copies[0].Options.Replacements[1].Old)
				require.Equal(t, "yyz", cfg.Copies[0].Options.Replacements[1].New)
				require.Len(t, cfg.Copies[0].Options.IgnoreFiles, 1)
				require.Equal(t, "*.txt", cfg.Copies[0].Options.IgnoreFiles[0])
			},
		},
		{
			name: "invalid_hcl_syntax",
			config: `
copy {
  source {
    repo = org/repo" # Missing quote
    path = "/src"
  }
  destination {
    path = "/dest"
  }
}
`,
			expectError: true,
		},
		{
			name: "missing_required_fields",
			config: `
copy {
  source {
    repo = "org/repo"
    # Missing path
  }
  destination {
    path = "/dest"
  }
}
`,
			expectError: true,
		},
		{
			name: "no_copies",
			config: `
`,
			expectError: false,
		},
		{
			name: "invalid_replacement_format",
			config: `
copy {
  source {
    repo = "org/repo"
    path = "/src"
  }
  destination {
    path = "/dest"
  }
  options {
    replacements = ["invalid"]
  }
}
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.hcl")
			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			cfg, err := LoadConfig(configPath)
			if tt.expectError {
				require.Error(t, err)
				if tt.validate != nil {
					tt.validate(t, nil)
				}
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
