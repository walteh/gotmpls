package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"gitlab.com/tozd/go/errors"
	"gopkg.in/yaml.v3"
)

// 📝 Config file structure
type CopyConfig struct {
	// 🎯 Global settings
	DefaultBranch  string `json:"default_branch,omitempty"`  // Default branch to use if not specified
	FallbackBranch string `json:"fallback_branch,omitempty"` // Fallback branch if default branch doesn't exist
	StatusFile     string `json:"status_file,omitempty"`     // Name of the status file (defaults to .copy-status)

	// 🔧 Default settings block
	Defaults *DefaultsBlock `json:"defaults,omitempty"`

	// 📝 Copy configurations
	Copies []*CopyEntry `json:"copies"`
}

// 🔧 Default settings that apply to all copies
type DefaultsBlock struct {
	Source      *CopyEntry_Source      `json:"source,omitempty" yaml:"source,omitempty" hcl:"source,block"`
	Destination *CopyEntry_Destination `json:"destination,omitempty" yaml:"destination,omitempty" hcl:"destination,block"`
	Options     *CopyEntry_Options     `json:"options,omitempty" yaml:"options,omitempty" hcl:"options,block"`
}

// 🎯 Source configuration
type CopyEntry_Source struct {
	Repo           string `json:"repo" yaml:"repo" hcl:"repo,attr"`
	Ref            string `json:"ref,omitempty" yaml:"ref,omitempty" hcl:"ref,optional"`
	Path           string `json:"path" yaml:"path" hcl:"path,attr"`
	FallbackBranch string `json:"fallback_branch,omitempty" yaml:"fallback_branch,omitempty" hcl:"fallback_branch,optional"`
}

// 📦 Destination configuration
type CopyEntry_Destination struct {
	Path string `json:"path" yaml:"path" hcl:"path,attr"`
}

// 🔧 Processing options (internal)
type CopyEntry_Options struct {
	Replacements []string `json:"replacements,omitempty" yaml:"replacements,omitempty" hcl:"replacements,optional"`
	IgnoreFiles  []string `json:"ignore_files,omitempty" yaml:"ignore_files,omitempty" hcl:"ignore_files,optional"`
}

// 📝 Individual copy entry
type CopyEntry struct {
	Source      CopyEntry_Source      `json:"source" yaml:"source" hcl:"source,block"`
	Destination CopyEntry_Destination `json:"destination" yaml:"destination" hcl:"destination,block"`
	Options     *CopyEntry_Options    `json:"options,omitempty" yaml:"options,omitempty" hcl:"options,block"`
}

// 🔧 Processing options for YAML/HCL
type configOptions struct {
	Replacements []string `yaml:"replacements,omitempty" hcl:"replacements,optional"`
	IgnoreFiles  []string `yaml:"ignore_files,omitempty" hcl:"ignore_files,optional"`
}

// 📝 Individual copy entry for YAML/HCL
type configEntry struct {
	Source      CopyEntry_Source      `yaml:"source" hcl:"source,block"`
	Destination CopyEntry_Destination `yaml:"destination" hcl:"destination,block"`
	Options     *configOptions        `yaml:"options,omitempty" hcl:"options,block"`
}

// 📝 Config file structure for YAML/HCL
type configFile struct {
	DefaultBranch  string         `yaml:"default_branch,omitempty" hcl:"default_branch,optional"`
	FallbackBranch string         `yaml:"fallback_branch,omitempty" hcl:"fallback_branch,optional"`
	StatusFile     string         `yaml:"status_file,omitempty" hcl:"status_file,optional"`
	Defaults       *DefaultsBlock `yaml:"defaults,omitempty" hcl:"defaults,block"`
	Copies         []*CopyEntry   `yaml:"copies" hcl:"copy,block"`
}

// 🔧 HCL-specific schema
type hclConfig struct {
	DefaultBranch  string        `hcl:"default_branch,optional"`
	FallbackBranch string        `hcl:"fallback_branch,optional"`
	StatusFile     string        `hcl:"status_file,optional"`
	Defaults       *hclDefaults  `hcl:"defaults,block"`
	Copies         []*hclCopyDef `hcl:"copy,block"`
}

type hclDefaults struct {
	Source      *hclDefaultSource `hcl:"source,block"`
	Destination *hclDestination   `hcl:"destination,block"`
	Options     *hclOptions       `hcl:"options,block"`
}

type hclDefaultSource struct {
	Repo           string `hcl:"repo,optional"`
	Ref            string `hcl:"ref,optional"`
	Path           string `hcl:"path,optional"`
	FallbackBranch string `hcl:"fallback_branch,optional"`
}

type hclSource struct {
	Repo           string `hcl:"repo"`
	Ref            string `hcl:"ref,optional"`
	Path           string `hcl:"path"`
	FallbackBranch string `hcl:"fallback_branch,optional"`
}

type hclDestination struct {
	Path string `hcl:"path"`
}

type hclOptions struct {
	Replacements []string `hcl:"replacements,optional"`
	IgnoreFiles  []string `hcl:"ignore_files,optional"`
}

type hclCopyDef struct {
	Source      *hclSource      `hcl:"source,block"`
	Destination *hclDestination `hcl:"destination,block"`
	Options     *hclOptions     `hcl:"options,block"`
}

// 🔄 Parse replacement from various formats
func parseReplacements(replacements []interface{}) ([]string, error) {
	result := make([]string, 0, len(replacements))

	for _, r := range replacements {
		switch v := r.(type) {
		case string:
			// Handle old:new format
			parts := strings.SplitN(v, ":", 2)
			if len(parts) != 2 {
				return nil, errors.Errorf("invalid replacement format: %s", v)
			}
			result = append(result, v)
		case map[interface{}]interface{}:
			// Handle {from: xyz, to: xyz} format
			from, ok1 := v["from"].(string)
			to, ok2 := v["to"].(string)
			if !ok1 || !ok2 {
				return nil, errors.New("replacement must have 'from' and 'to' as strings")
			}
			result = append(result, fmt.Sprintf("%s:%s", from, to))
		case map[string]interface{}:
			// Handle {from: xyz, to: xyz} format
			from, ok1 := v["from"].(string)
			to, ok2 := v["to"].(string)
			if !ok1 || !ok2 {
				return nil, errors.New("replacement must have 'from' and 'to' as strings")
			}
			result = append(result, fmt.Sprintf("%s:%s", from, to))
		default:
			return nil, errors.Errorf("replacement must be string or map, got %T", r)
		}
	}

	return result, nil
}

// 📝 Load config from file (supports YAML and HCL)
func LoadConfig(path string) (*CopyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("reading config file: %w", err)
	}

	// Try YAML first
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		var cfg configFile
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil {
			return nil, errors.Errorf("parsing YAML: %w", err)
		}
		return convertConfig(&cfg)
	} else if strings.HasSuffix(path, ".hcl") {
		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCL(data, path)
		if diags.HasErrors() {
			return nil, errors.Errorf("parsing HCL: %s", diags.Error())
		}

		// Create evaluation context
		ctx := &hcl.EvalContext{
			Variables: map[string]cty.Value{},
		}

		// Decode HCL into our HCL-specific schema
		var cfg hclConfig
		diags = gohcl.DecodeBody(hclFile.Body, ctx, &cfg)
		if diags.HasErrors() {
			return nil, errors.Errorf("decoding HCL: %s", diags.Error())
		}

		// Convert to internal format
		return convertHCLConfig(&cfg)
	}

	return nil, errors.New("unsupported config file format (use .yaml, .yml, or .hcl)")
}

// 🔄 Convert config from YAML/HCL format to internal format
func convertConfig(cfg *configFile) (*CopyConfig, error) {
	result := &CopyConfig{
		DefaultBranch:  cfg.DefaultBranch,
		FallbackBranch: cfg.FallbackBranch,
		StatusFile:     cfg.StatusFile,
		Defaults:       cfg.Defaults,
		Copies:         make([]*CopyEntry, 0, len(cfg.Copies)),
	}

	// Copy entries
	result.Copies = append(result.Copies, cfg.Copies...)

	// Set defaults if not provided
	if result.DefaultBranch == "" {
		result.DefaultBranch = "main"
	}
	if result.FallbackBranch == "" {
		result.FallbackBranch = "master"
	}
	if result.StatusFile == "" {
		result.StatusFile = ".copy-status"
	}

	// Apply defaults from defaults block
	if result.Defaults != nil {
		for _, entry := range result.Copies {
			if entry.Source.Ref == "" && result.Defaults.Source != nil {
				entry.Source.Ref = result.Defaults.Source.Ref
			}
			if entry.Source.FallbackBranch == "" && result.Defaults.Source != nil {
				entry.Source.FallbackBranch = result.Defaults.Source.FallbackBranch
			}
			if entry.Options == nil && result.Defaults.Options != nil {
				entry.Options = &CopyEntry_Options{
					Replacements: result.Defaults.Options.Replacements,
					IgnoreFiles:  result.Defaults.Options.IgnoreFiles,
				}
			}
		}
	}

	// Validate required fields
	for i, entry := range result.Copies {
		if entry.Source.Repo == "" {
			return nil, errors.Errorf("copy entry %d: source repo is required", i)
		}
		if entry.Source.Path == "" {
			return nil, errors.Errorf("copy entry %d: source path is required", i)
		}
		if entry.Destination.Path == "" {
			return nil, errors.Errorf("copy entry %d: destination path is required", i)
		}

		// Validate replacements
		if entry.Options != nil {
			for j, r := range entry.Options.Replacements {
				parts := strings.SplitN(r, ":", 2)
				if len(parts) != 2 {
					return nil, errors.Errorf("copy entry %d, replacement %d: invalid format", i, j)
				}
			}
		}
	}

	// Validate at least one copy entry
	if len(result.Copies) == 0 {
		return nil, errors.New("no copy entries defined")
	}

	return result, nil
}

// 🔄 Convert HCL config to internal format
func convertHCLConfig(cfg *hclConfig) (*CopyConfig, error) {
	result := &CopyConfig{
		DefaultBranch:  cfg.DefaultBranch,
		FallbackBranch: cfg.FallbackBranch,
		StatusFile:     cfg.StatusFile,
	}

	// Convert defaults if present
	if cfg.Defaults != nil {
		result.Defaults = &DefaultsBlock{}
		if cfg.Defaults.Source != nil {
			result.Defaults.Source = &CopyEntry_Source{
				Repo:           cfg.Defaults.Source.Repo,
				Ref:            cfg.Defaults.Source.Ref,
				Path:           cfg.Defaults.Source.Path,
				FallbackBranch: cfg.Defaults.Source.FallbackBranch,
			}
		}
		if cfg.Defaults.Destination != nil {
			result.Defaults.Destination = &CopyEntry_Destination{
				Path: cfg.Defaults.Destination.Path,
			}
		}
		if cfg.Defaults.Options != nil {
			result.Defaults.Options = &CopyEntry_Options{
				Replacements: cfg.Defaults.Options.Replacements,
				IgnoreFiles:  cfg.Defaults.Options.IgnoreFiles,
			}
		}
	}

	// Convert copy entries
	result.Copies = make([]*CopyEntry, 0, len(cfg.Copies))
	for _, copy := range cfg.Copies {
		entry := &CopyEntry{
			Source: CopyEntry_Source{
				Repo:           copy.Source.Repo,
				Ref:            copy.Source.Ref,
				Path:           copy.Source.Path,
				FallbackBranch: copy.Source.FallbackBranch,
			},
			Destination: CopyEntry_Destination{
				Path: copy.Destination.Path,
			},
		}
		if copy.Options != nil {
			entry.Options = &CopyEntry_Options{
				Replacements: copy.Options.Replacements,
				IgnoreFiles:  copy.Options.IgnoreFiles,
			}
		}
		result.Copies = append(result.Copies, entry)
	}

	// Set defaults and validate
	return finalizeConfig(result)
}

// 🔍 Set defaults and validate config
func finalizeConfig(cfg *CopyConfig) (*CopyConfig, error) {
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
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
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
			parts := strings.SplitN(r, ":", 2)
			if len(parts) != 2 {
				return errors.Errorf("copy entry %d, replacement %d: invalid format", i, j)
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

		// Copy replacements
		if copyEntry.Options != nil {
			input.Replacements = make(arrayFlags, len(copyEntry.Options.Replacements))
			copy(input.Replacements, copyEntry.Options.Replacements)

			// Copy ignore files
			input.IgnoreFiles = make(arrayFlags, len(copyEntry.Options.IgnoreFiles))
			copy(input.IgnoreFiles, copyEntry.Options.IgnoreFiles)
		}

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
