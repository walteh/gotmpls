package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/tozd/go/errors"
)

// 📝 Status file entry
type StatusEntry struct {
	File        string    `json:"file"`
	Source      string    `json:"source"`
	Permalink   string    `json:"permalink"`
	LastUpdated time.Time `json:"last_updated"`
	Changes     []string  `json:"changes,omitempty"`
}

type GeneratedFileEntry struct {
	File        string    `json:"file"`
	LastUpdated time.Time `json:"last_updated"`
}

type StatusFileArgs struct {
	SrcRepo     string             `json:"src_repo"`
	SrcRef      string             `json:"src_ref"`
	SrcPath     string             `json:"src_path,omitempty"`
	CopyArgs    *ConfigCopyArgs    `json:"copy_args,omitempty"`
	ArchiveArgs *ConfigArchiveArgs `json:"archive_args,omitempty"`
}

// 📦 Status file structure
type StatusFile struct {
	LastUpdated    time.Time                     `json:"last_updated"`
	CommitHash     string                        `json:"commit_hash"`
	Ref            string                        `json:"branch"`
	CoppiedFiles   map[string]StatusEntry        `json:"coppied_files"`
	GeneratedFiles map[string]GeneratedFileEntry `json:"generated_files"`
	Warnings       []string                      `json:"warnings,omitempty" hcl:"warnings,omitempty" yaml:"warnings,omitempty"`
	Args           StatusFileArgs                `json:"args" hcl:"args" yaml:"args"`
}

// 📝 Load status file
func loadStatusFile(path string) (*StatusFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var status StatusFile
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, errors.Errorf("parsing status file: %w", err)
	}

	if status.CoppiedFiles == nil {
		status.CoppiedFiles = make(map[string]StatusEntry)
	}

	if status.GeneratedFiles == nil {
		status.GeneratedFiles = make(map[string]GeneratedFileEntry)
	}

	return &status, nil
}

// 📝 Write status file
func writeStatusFile(ctx context.Context, status *StatusFile, destPath string) error {
	statusPath := filepath.Join(destPath, ".copyrc.lock")

	// Marshal status data
	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return errors.Errorf("marshaling status: %w", err)
	}

	// Write status file if changed
	_, err = writeFile(ctx, WriteFileOpts{
		Path:         statusPath,
		Contents:     data,
		StatusFile:   status,
		IsStatusFile: true,
		IsManaged:    true,
	})
	if err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	return nil
}
