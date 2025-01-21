package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarballFunctions(t *testing.T) {
	// Create mock provider
	mock := NewMockProvider(t)
	mock.AddFile("test.txt", []byte("test content"))
	mock.AddFile("dir/nested.txt", []byte("nested content"))

	t.Run("test_get_file", func(t *testing.T) {
		args := ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "path/to/files",
		}

		data, err := GetFileFromTarball(context.Background(), mock, args)
		require.NoError(t, err, "getting file should succeed")
		assert.Equal(t, []byte{0x1f, 0x8b}, data[0:2], "should be gzipped data")
	})

	t.Run("test_get_nested_file", func(t *testing.T) {
		args := ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "path/to/files",
		}

		data, err := GetFileFromTarball(context.Background(), mock, args)
		require.NoError(t, err, "getting nested file should succeed")
		assert.Equal(t, []byte{0x1f, 0x8b}, data[0:2], "should be gzipped data")
	})

	t.Run("test_file_not_found", func(t *testing.T) {
		args := ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "/invalid/path",
		}

		_, err := GetFileFromTarball(context.Background(), mock, args)
		require.Error(t, err, "getting nonexistent file should fail")
		assert.Contains(t, err.Error(), "invalid path")
	})

	t.Run("test_invalid_cache_dir", func(t *testing.T) {
		args := ProviderArgs{
			Repo: "github.com/org/repo",
			Ref:  "main",
			Path: "/invalid/path",
		}

		_, err := GetFileFromTarball(context.Background(), mock, args)
		require.Error(t, err, "getting file with invalid cache dir should fail")
		assert.Contains(t, err.Error(), "invalid path")
	})
}
