package finder

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFinder_FindTemplates(t *testing.T) {
	tests := []struct {
		name          string
		dir           string
		extensions    []string
		expectedFiles []string
		wantErr       bool
	}{
		{
			name:       "finds tmpl files",
			dir:        "testdata",
			extensions: []string{".tmpl"},
			expectedFiles: []string{
				filepath.Join("testdata", "sample.tmpl"),
			},
			wantErr: false,
		},
		{
			name:       "finds tmpl.go files",
			dir:        "testdata",
			extensions: []string{".tmpl.go"},
			expectedFiles: []string{
				filepath.Join("testdata", "sample.tmpl.go"),
			},
			wantErr: false,
		},
		{
			name:       "finds both types",
			dir:        "testdata",
			extensions: []string{".tmpl", ".tmpl.go"},
			expectedFiles: []string{
				filepath.Join("testdata", "sample.tmpl"),
				filepath.Join("testdata", "sample.tmpl.go"),
			},
			wantErr: false,
		},
		{
			name:          "no matches",
			dir:           "testdata",
			extensions:    []string{".unknown"},
			expectedFiles: []string{},
			wantErr:       false,
		},
		{
			name:          "invalid directory",
			dir:           "nonexistent",
			extensions:    []string{".tmpl"},
			expectedFiles: nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDefaultFinder()

			got, err := f.FindTemplates(context.Background(), tt.dir, tt.extensions)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedFiles, got)
		})
	}
}
