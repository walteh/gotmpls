package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
)

// WriteFileOpts represents options for writing a file
type WriteFileOpts struct {
	// Required fields
	Path     string // Path to write the file to
	Contents []byte // Contents to write to the file
	// FileType FileType // Type of file (managed/local/copy)

	// Optional fields
	StatusFile       *StatusFile // Full status file for checking existing entries
	StatusMutex      *sync.Mutex // Mutex for status file access
	ReplacementCount int         // Number of replacements made in the file
	EnsureNewline    bool        // Ensure contents end with a newline
	Source           string      // Source info for status entry
	Permalink        string      // Permalink for status entry
	Changes          []string    // Changes made to the file
	IsStatusFile     bool        // Whether this is a status file
	IsUntracked      bool        // Whether this is an untracked file
	IsManaged        bool        // Whether this is a managed file
}

// writeFile handles all file writing scenarios including status updates and logging.
// It returns true if the file was written, false if no changes were needed.
func writeFile(ctx context.Context, opts WriteFileOpts) (bool, error) {
	// Validate required fields
	if opts.Path == "" {
		return false, errors.New("path is required")
	}

	fileName := filepath.Base(opts.Path)

	if opts.IsUntracked {
		logFileOperation(ctx, FileInfo{
			Name:        fileName,
			IsUntracked: true,
		})

		return false, nil
	}

	if opts.StatusFile == nil && !opts.IsUntracked {
		return false, errors.New("status file is required")
	}

	if len(opts.Contents) == 0 {
		return false, errors.Errorf("contents are required for %s", opts.Path)
	}

	if opts.IsStatusFile {
		opts.IsManaged = true
	}

	// Get base name for status entries

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0755); err != nil {
		return false, errors.Errorf("creating directory for %s: %w", opts.Path, err)
	}

	// Try to read existing file
	existing, err := os.ReadFile(opts.Path)
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Errorf("reading existing file %s: %w", opts.Path, err)
	}

	// Ensure newline at end of file if requested
	contents := opts.Contents
	if opts.EnsureNewline && !bytes.HasSuffix(contents, []byte("\n")) {
		contents = append(contents, '\n')
	}

	logger := loggerFromContext(ctx)
	logger.zlog.Debug().Msgf("👀 Writing file %s with contents length %d, curr len: %d, equal: %t", opts.Path, len(contents), len(existing), bytes.Equal(existing, contents))

	var hasEntry bool = false
	if opts.StatusFile != nil {
		if opts.IsManaged {
			_, hasEntry = opts.StatusFile.GeneratedFiles[fileName]
		} else {
			_, hasEntry = opts.StatusFile.CoppiedFiles[fileName]
		}
	}

	// If file exists and content is the same, and we have an existing status entry, no need to write
	if err == nil && bytes.Equal(existing, contents) && (hasEntry || opts.IsStatusFile) {
		// Log the unchanged status
		logFileOperation(ctx, FileInfo{
			Name:         fileName,
			IsModified:   false,
			IsManaged:    opts.IsManaged,
			Replacements: opts.ReplacementCount,
		})

		// Update status entry timestamps even if content hasn't changed
		// now := time.Now().UTC()
		// if opts.StatusFile != nil && opts.StatusMutex != nil {
		// 	opts.StatusMutex.Lock()
		// 	if opts.IsGenerated {
		// 		entry, ok := opts.StatusFile.GeneratedFiles[fileName]
		// 		if !ok {
		// 			entry = GeneratedFileEntry{
		// 				File: fileName,
		// 			}
		// 		}

		// 		entry.LastUpdated = now
		// 		opts.StatusFile.GeneratedFiles[fileName] = entry
		// 	} else {
		// 		entry, ok := opts.StatusFile.CoppiedFiles[fileName]
		// 		if !ok {
		// 			entry = StatusEntry{
		// 				File: fileName,
		// 			}
		// 		}

		// 		entry.LastUpdated = now
		// 		entry.Source = opts.Source
		// 		entry.Permalink = opts.Permalink
		// 		entry.Changes = opts.Changes
		// 		opts.StatusFile.CoppiedFiles[fileName] = entry
		// 	}
		// 	opts.StatusMutex.Unlock()
		// }

		return false, nil
	}

	if opts.IsStatusFile {
		opts.StatusFile.LastUpdated = time.Now()

		// Marshal status data
		data, err := json.MarshalIndent(opts.StatusFile, "", "\t")
		if err != nil {
			return false, errors.Errorf("marshaling status: %w", err)
		}
		opts.Contents = data
	}

	// fmt.Printf("WRITING FILE %s\n", opts.Path)
	// Write the file
	if err := os.WriteFile(opts.Path, contents, 0644); err != nil {
		return false, errors.Errorf("writing file %s: %w", opts.Path, err)
	}

	// print out dir files
	// files, err := os.ReadDir(filepath.Dir(opts.Path))
	// if err != nil {
	// 	return false, errors.Errorf("reading directory %s: %w", filepath.Dir(opts.Path), err)
	// }
	// fmt.Printf("DIR: %+v\n", files)

	// Update status entries
	now := time.Now().UTC()
	if opts.StatusFile != nil && opts.StatusMutex != nil {
		opts.StatusMutex.Lock()
		if opts.IsManaged {
			entry, ok := opts.StatusFile.GeneratedFiles[fileName]
			if !ok {
				entry = GeneratedFileEntry{
					File: fileName,
				}
			}
			entry.LastUpdated = now
			opts.StatusFile.GeneratedFiles[fileName] = entry
		} else {
			entry, ok := opts.StatusFile.CoppiedFiles[fileName]
			if !ok {
				entry = StatusEntry{
					File: fileName,
				}
			}
			entry.LastUpdated = now
			entry.Source = opts.Source
			entry.Permalink = opts.Permalink
			entry.Changes = opts.Changes
			opts.StatusFile.CoppiedFiles[fileName] = entry
		}
		opts.StatusMutex.Unlock()
	}

	// Log the operation based on what changed
	logFileOperation(ctx, FileInfo{
		Name:         fileName,
		IsNew:        os.IsNotExist(err),
		IsModified:   true,
		IsManaged:    opts.IsManaged,
		Replacements: opts.ReplacementCount,
	})

	return true, nil
}
