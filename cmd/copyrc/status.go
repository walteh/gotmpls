package main

import (
	"bytes"
	"encoding/json"
	"os"
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
func writeStatusFile(path string, status *StatusFile) error {
	// read the current status file json
	currentStatus, err := os.ReadFile(path)
	if err != nil {
		currentStatus = []byte("invalid")
	}

	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return errors.Errorf("marshaling status: %w", err)
	}

	// if the current status file is the same as the new status file, return
	if bytes.Equal(currentStatus, data) {
		return nil
	}

	status.LastUpdated = time.Now().UTC()
	data, err = json.MarshalIndent(status, "", "\t")
	if err != nil {
		return errors.Errorf("marshaling status: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	return nil
}
