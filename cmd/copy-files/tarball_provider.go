package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"gitlab.com/tozd/go/errors"
)

// 📦 TarballOptions configures tarball extraction behavior
type TarballOptions struct {
	CacheDir string // Directory where tarballs are cached
}

// 📥 GetFileFromTarball downloads and extracts a specific file from a repository tarball
func GetFileFromTarball(ctx context.Context, provider RepoProvider, args ProviderArgs) ([]byte, error) {

	// Download tarball if needed
	data, err := getArchiveData(ctx, provider, args)
	if err != nil {
		return nil, errors.Errorf("getting archive data: %w", err)
	}

	return data, nil
}

// 🔄 getArchiveData downloads and caches the repository tarball
func getArchiveData(ctx context.Context, provider RepoProvider, args ProviderArgs) ([]byte, error) {
	// Get archive URL
	url, err := provider.GetArchiveUrl(ctx, args)
	if err != nil {
		return nil, errors.Errorf("getting archive url: %w", err)
	}

	// Read data based on URL scheme
	var data []byte
	if strings.HasPrefix(url, "file://") {
		// Local file URL
		path := strings.TrimPrefix(url, "file://")
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, errors.Errorf("reading local archive: %w", err)
		}
	} else if strings.HasPrefix(url, "https://") {
		// Remote HTTPS URL
		resp, err := http.Get(url)
		if err != nil {
			return nil, errors.Errorf("downloading archive: %w", err)
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Errorf("reading response: %w", err)
		}
	} else {
		return nil, errors.Errorf("unsupported URL scheme: %s", url)
	}

	return data, nil
}
