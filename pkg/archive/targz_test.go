package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTarGz(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	err := tw.Close()
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	return buf.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	testFiles := map[string]string{
		"dir1/file1.txt":         "content1",
		"dir1/dir2/file2.txt":    "content2",
		"dir1/dir2/dir3/file.go": "package main",
	}

	data := createTestTarGz(t, testFiles)

	tmpDir, err := os.MkdirTemp("", "targz-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = ExtractTarGz(data, tmpDir)
	require.NoError(t, err)

	// Verify files were extracted correctly
	for name, expectedContent := range testFiles {
		content, err := os.ReadFile(filepath.Join(tmpDir, name))
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	}
}

func TestExtractTarGzWithOptions(t *testing.T) {
	testFiles := map[string]string{
		"prefix/dir1/file1.txt":      "content1",
		"prefix/dir1/dir2/file2.txt": "content2",
		"prefix/skip/file3.txt":      "content3",
	}

	data := createTestTarGz(t, testFiles)

	tmpDir, err := os.MkdirTemp("", "targz-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	opts := ExtractOptions{
		StripComponents: 1,
		FileMode:        0600,
		DirMode:         0700,
		Filter: func(header *tar.Header) bool {
			return header.Name != "prefix/skip/file3.txt"
		},
	}

	err = ExtractTarGzWithOptions(data, tmpDir, opts)
	require.NoError(t, err)

	// Verify files were extracted correctly with stripped prefix
	expectedFiles := map[string]string{
		"dir1/file1.txt":      "content1",
		"dir1/dir2/file2.txt": "content2",
	}

	for name, expectedContent := range expectedFiles {
		path := filepath.Join(tmpDir, name)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))

		// Check file mode
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}

	// Verify filtered file was not extracted
	_, err = os.Stat(filepath.Join(tmpDir, "skip/file3.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{
			path:     "dir1/dir2/file.txt",
			expected: []string{"dir1", "dir2", "file.txt"},
		},
		{
			path:     "file.txt",
			expected: []string{"file.txt"},
		},
		{
			path:     "dir1/dir2/",
			expected: []string{"dir1", "dir2"},
		},
		{
			path:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := splitPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
