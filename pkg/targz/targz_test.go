package targz_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/targz"
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

	err = targz.ExtractTarGz(data, tmpDir)
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

	opts := targz.ExtractOptions{
		StripComponents: 1,
		FileMode:        0600,
		DirMode:         0700,
		Filter: func(header *tar.Header) bool {
			return header.Name != "prefix/skip/file3.txt"
		},
	}

	err = targz.ExtractTarGzWithOptions(data, tmpDir, opts)
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
			result := targz.SplitPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadTarGz(t *testing.T) {
	testFiles := map[string]string{
		"dir1/file1.txt":         "content1",
		"dir1/dir2/file2.txt":    "content2",
		"dir1/dir2/dir3/file.go": "package main",
	}

	data := createTestTarGz(t, testFiles)

	store, err := targz.LoadTarGz(data)
	require.NoError(t, err, "loading tar.gz should succeed")

	// Verify files were loaded correctly
	for name, expectedContent := range testFiles {
		content, ok := store.Files[name]
		require.True(t, ok, "file %s should exist in store", name)
		assert.Equal(t, expectedContent, string(content), "file %s should have correct content", name)
	}
}

func TestLoadTarGzWithOptions(t *testing.T) {
	testFiles := map[string]string{
		"prefix/dir1/file1.txt":      "content1",
		"prefix/dir1/dir2/file2.txt": "content2",
		"prefix/skip/file3.txt":      "content3",
	}

	data := createTestTarGz(t, testFiles)

	opts := targz.LoadOptions{
		StripComponents: 1,
		Filter: func(header *tar.Header) bool {
			return !strings.Contains(header.Name, "/skip/")
		},
		Transform: func(data []byte) ([]byte, error) {
			return []byte(strings.ToUpper(string(data))), nil
		},
	}

	store, err := targz.LoadTarGzWithOptions(data, opts)
	require.NoError(t, err, "loading tar.gz with options should succeed")

	// Expected files after stripping prefix and applying transform
	expectedFiles := map[string]string{
		"dir1/file1.txt":      "CONTENT1",
		"dir1/dir2/file2.txt": "CONTENT2",
	}

	// Verify files were loaded correctly
	for name, expectedContent := range expectedFiles {
		content, ok := store.Files[name]
		require.True(t, ok, "file %s should exist in store", name)
		assert.Equal(t, expectedContent, string(content), "file %s should have correct content", name)
	}

	// Verify filtered file was not loaded
	_, ok := store.Files["skip/file3.txt"]
	assert.False(t, ok, "filtered file should not exist in store")
}

func TestLoadTarGzWithNameTransform(t *testing.T) {
	testFiles := map[string]string{
		"dir1/file1.txt": "content1",
		"dir2/file2.txt": "content2",
	}

	data := createTestTarGz(t, testFiles)

	t.Run("test_successful_transform", func(t *testing.T) {
		opts := targz.LoadOptions{
			TransformName: func(name string) string {
				return "prefix_" + name
			},
		}

		store, err := targz.LoadTarGzWithOptions(data, opts)
		require.NoError(t, err, "loading tar.gz with name transform should succeed")

		expectedFiles := map[string]string{
			"prefix_dir1/file1.txt": "content1",
			"prefix_dir2/file2.txt": "content2",
		}

		for name, expectedContent := range expectedFiles {
			content, ok := store.Files[name]
			require.True(t, ok, "file %s should exist in store", name)
			assert.Equal(t, expectedContent, string(content), "file %s should have correct content", name)
		}
	})

	t.Run("test_collision_detection", func(t *testing.T) {
		opts := targz.LoadOptions{
			TransformName: func(name string) string {
				// Transform all paths to the same name to force collision
				return "collision.txt"
			},
		}

		_, err := targz.LoadTarGzWithOptions(data, opts)
		require.Error(t, err, "loading tar.gz with colliding names should fail")
		assert.Contains(t, err.Error(), "file collision", "error should mention file collision")
	})

	t.Run("test_transform_with_strip_components", func(t *testing.T) {
		opts := targz.LoadOptions{
			StripComponents: 1,
			TransformName: func(name string) string {
				return "transformed_" + name
			},
		}

		store, err := targz.LoadTarGzWithOptions(data, opts)
		require.NoError(t, err, "loading tar.gz with name transform and strip components should succeed")

		expectedFiles := map[string]string{
			"transformed_file1.txt": "content1",
			"transformed_file2.txt": "content2",
		}

		for name, expectedContent := range expectedFiles {
			content, ok := store.Files[name]
			require.True(t, ok, "file %s should exist in store", name)
			assert.Equal(t, expectedContent, string(content), "file %s should have correct content", name)
		}
	})
}
