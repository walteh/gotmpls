package finder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFinder_FindTemplates(t *testing.T) {
	// Create a temporary test directory
	testDir, err := os.MkdirTemp("", "finder-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create test files
	files := map[string]string{
		"template1.tmpl":    "{{.Name}}",
		"template2.gotmpl":  "{{.Age}}",
		"not-a-template.go": "package main",
		"sub/nested.tmpl":   "{{.Address}}",
	}

	for name, content := range files {
		path := filepath.Join(testDir, name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		dir        string
		extensions []string
		want       int // Number of files expected
		wantErr    bool
	}{
		{
			name:       "find all templates",
			dir:        testDir,
			extensions: []string{".tmpl", ".gotmpl"},
			want:       3,
			wantErr:    false,
		},
		{
			name:       "find only .tmpl",
			dir:        testDir,
			extensions: []string{".tmpl"},
			want:       2,
			wantErr:    false,
		},
		{
			name:       "default extensions",
			dir:        testDir,
			extensions: nil,
			want:       3,
			wantErr:    false,
		},
		{
			name:       "non-existent directory",
			dir:        filepath.Join(testDir, "non-existent"),
			extensions: nil,
			want:       0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDefaultFinder()
			got, err := f.FindTemplates(context.Background(), tt.dir, tt.extensions)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.want)

			// Verify file contents are loaded
			for _, file := range got {
				assert.NotEmpty(t, file.Content)
				assert.NotEmpty(t, file.Path)
				assert.NotEmpty(t, file.FileType)
			}
		})
	}
}

func TestDefaultFinder_FindTemplates_Context(t *testing.T) {
	testDir, err := os.MkdirTemp("", "finder-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create a test file
	err = os.WriteFile(filepath.Join(testDir, "test.tmpl"), []byte("{{.Test}}"), 0644)
	require.NoError(t, err)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	f := NewDefaultFinder()
	_, err = f.FindTemplates(ctx, testDir, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
