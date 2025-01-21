package main

import (
	"bytes"
	"os"
	"strings"

	"gitlab.com/tozd/go/errors"
	"gopkg.in/yaml.v3"
)

// 📦 Config file structure
type CopyConfig struct {
	// 🎯 Global settings
	DefaultBranch  string `yaml:"default_branch,omitempty"`  // Default branch to use if not specified
	FallbackBranch string `yaml:"fallback_branch,omitempty"` // Fallback branch if default branch doesn't exist
	StatusFile     string `yaml:"status_file,omitempty"`     // Name of the status file (defaults to .copy-status)

	// 📝 Copy configurations
	Copies []CopyEntry `yaml:"copies"`
}

// 📝 Individual copy entry
type CopyEntry struct {
	// 🎯 Source configuration
	Source struct {
		Repo           string `yaml:"repo"`                      // e.g. github.com/org/repo
		Ref            string `yaml:"ref"`                       // Branch or commit hash
		Path           string `yaml:"path"`                      // Path within repo
		FallbackBranch string `yaml:"fallback_branch,omitempty"` // Override global fallback branch
	} `yaml:"source"`

	// 📦 Destination configuration
	Destination struct {
		Path string `yaml:"path"` // Local path
	} `yaml:"destination"`

	// 🔧 Processing options
	Options struct {
		Replacements []interface{} `yaml:"replacements,omitempty"` // String replacements in old:new format or {this,with} format
		IgnoreFiles  []string      `yaml:"ignore_files,omitempty"` // Files to ignore
	} `yaml:"options"`
}

// 🔄 Parse replacement from various formats
func parseReplacement(r interface{}) (Replacement, error) {
	switch v := r.(type) {
	case string:
		// Handle old:new format
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return Replacement{}, errors.Errorf("invalid replacement format: %s", v)
		}
		return Replacement{Old: parts[0], New: parts[1]}, nil
	case map[string]interface{}:
		// Handle {this: xyz, with: xyz} format
		this, ok1 := v["this"].(string)
		with, ok2 := v["with"].(string)
		if !ok1 || !ok2 {
			return Replacement{}, errors.New("replacement must have 'this' and 'with' as strings")
		}
		return Replacement{Old: this, New: with}, nil
	default:
		return Replacement{}, errors.Errorf("unsupported replacement type: %T", r)
	}
}

// 📝 Load config from file
func LoadConfig(path string) (*CopyConfig, error) {
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("reading config file: %w", err)
	}

	// Parse YAML with strict mode
	var cfg CopyConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Strict mode
	if err := decoder.Decode(&cfg); err != nil {
		return nil, errors.Errorf("parsing config file: %w", err)
	}

	// Set defaults
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = "main"
	}
	if cfg.FallbackBranch == "" {
		cfg.FallbackBranch = "master"
	}
	if cfg.StatusFile == "" {
		cfg.StatusFile = ".copy-status"
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// 🔍 Validate config
func validateConfig(cfg *CopyConfig) error {
	if len(cfg.Copies) == 0 {
		return errors.New("no copy entries defined")
	}

	for i, copy := range cfg.Copies {
		if copy.Source.Repo == "" {
			return errors.Errorf("copy entry %d: source repo is required", i)
		}
		if copy.Source.Path == "" {
			return errors.Errorf("copy entry %d: source path is required", i)
		}
		if copy.Destination.Path == "" {
			return errors.Errorf("copy entry %d: destination path is required", i)
		}

		// Set default ref if not specified
		if copy.Source.Ref == "" {
			cfg.Copies[i].Source.Ref = cfg.DefaultBranch
		}

		// Parse replacements
		for j, r := range copy.Options.Replacements {
			switch v := r.(type) {
			case string:
				// Already in string format
				continue
			case map[string]interface{}:
				// Convert {this: xyz, with: yyz} format to string format
				this, ok1 := v["this"].(string)
				with, ok2 := v["with"].(string)
				if !ok1 || !ok2 {
					return errors.Errorf("copy entry %d, replacement %d: must have 'this' and 'with' as strings", i, j)
				}
				cfg.Copies[i].Options.Replacements[j] = this + ":" + with
			default:
				return errors.Errorf("copy entry %d, replacement %d: unsupported type %T", i, j, v)
			}
		}
	}

	return nil
}

// 🏃 Run all copy operations
func (cfg *CopyConfig) RunAll(clean, status, remoteStatus, force bool) error {
	for _, copyEntry := range cfg.Copies {
		input := Input{
			SrcRepo:      copyEntry.Source.Repo,
			SrcRef:       copyEntry.Source.Ref,
			SrcPath:      copyEntry.Source.Path,
			DestPath:     copyEntry.Destination.Path,
			Clean:        clean,
			Status:       status,
			RemoteStatus: remoteStatus,
			Force:        force,
		}

		// Convert replacements to strings
		input.Replacements = make(arrayFlags, len(copyEntry.Options.Replacements))
		for i, r := range copyEntry.Options.Replacements {
			switch v := r.(type) {
			case string:
				input.Replacements[i] = v
			case map[string]interface{}:
				if this, ok := v["this"].(string); ok {
					if with, ok := v["with"].(string); ok {
						input.Replacements[i] = this + ":" + with
					}
				}
			}
		}

		// Copy ignore files
		input.IgnoreFiles = make(arrayFlags, len(copyEntry.Options.IgnoreFiles))
		copy(input.IgnoreFiles, copyEntry.Options.IgnoreFiles)

		// Create config
		copyConfig, err := NewConfigFromInput(input)
		if err != nil {
			return errors.Errorf("creating config for %s: %w", copyEntry.Source.Repo, err)
		}

		// Set fallback branch if configured
		if gh, ok := copyConfig.Provider.(*GithubProvider); ok {
			// Use entry-specific fallback if set, otherwise use global
			if copyEntry.Source.FallbackBranch != "" {
				gh.fallbackBranch = copyEntry.Source.FallbackBranch
			} else {
				gh.fallbackBranch = cfg.FallbackBranch
			}
		}

		// Run copy operation
		if err := run(copyConfig); err != nil {
			return errors.Errorf("copying %s: %w", copyEntry.Source.Repo, err)
		}
	}

	return nil
}
