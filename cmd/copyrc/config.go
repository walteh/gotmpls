package main

import (
	"bytes"
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
	// 🔧 Default settings block
	Defaults *DefaultsBlock `json:"defaults,omitempty" hcl:"defaults,block" yaml:"defaults,omitempty"`

	// 📝 Copy configurations
	Copies []*CopyEntry `json:"copies" hcl:"copy,block" yaml:"copies"`
	// 📝 Archive configurations
	Archives []*ArchiveEntry `json:"archives" hcl:"archive,block" yaml:"archives"`
}

// 🔧 Default settings that apply to all copies
type DefaultsBlock struct {
	CopyOptions    *CopyEntry_Options    `json:"copy_options,omitempty" yaml:"copy_options,omitempty" hcl:"copy_options,block"`
	ArchiveOptions *ArchiveEntry_Options `json:"archive_options,omitempty" yaml:"archive_options,omitempty" hcl:"archive_options,block"`
}

// 🎯 Source configuration
type CopyEntry_Source struct {
	Repo string `json:"repo" yaml:"repo" hcl:"repo,attr"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty" hcl:"ref,attr"`
	Path string `json:"path" yaml:"path" hcl:"path,optional"`
}

// 📦 Destination configuration
type CopyEntry_Destination struct {
	Path string `json:"path" yaml:"path" hcl:"path,attr"`
}

// 🔧 Processing options (internal)
type CopyEntry_Options struct {
	Replacements []Replacement `json:"replacements,omitempty" yaml:"replacements,omitempty" hcl:"replacements,optional" cty:"replacements"`
	IgnoreFiles  []string      `json:"ignore_files,omitempty" yaml:"ignore_files,omitempty" hcl:"ignore_files,optional" cty:"ignore_files"`
}

// 📝 Individual copy entry
type CopyEntry struct {
	Source      CopyEntry_Source      `json:"source" yaml:"source" hcl:"source,block"`
	Destination CopyEntry_Destination `json:"destination" yaml:"destination" hcl:"destination,block"`
	Options     CopyEntry_Options     `json:"options" yaml:"options" hcl:"options,block"`
}

// 📝 Archive entry
type ArchiveEntry struct {
	Source      CopyEntry_Source      `yaml:"source" hcl:"source,block"`
	Destination CopyEntry_Destination `yaml:"destination" hcl:"destination,block"`
	Options     *ArchiveEntry_Options `yaml:"options,omitempty" hcl:"options,block"`
}

type ArchiveEntry_Options struct {
	GoEmbed bool `yaml:"go_embed,omitempty" hcl:"go_embed,optional"`
}

// 📝 Load config from file (supports YAML and HCL)
func LoadConfig(path string) (*CopyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("reading config file: %w", err)
	}

	// Try YAML first
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		var cfg CopyConfig
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil {
			return nil, errors.Errorf("parsing YAML: %w", err)
		}
		return &cfg, nil
	}
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
	var cfg CopyConfig
	diags = gohcl.DecodeBody(hclFile.Body, ctx, &cfg)
	if diags.HasErrors() {
		return nil, errors.Errorf("decoding HCL: %s", diags.Error())
	}

	// Convert to internal format
	return &cfg, nil

}

// 🏃 Run all copy operations
func (cfg *CopyConfig) RunAll(clean, status, remoteStatus, force bool, provider RepoProvider) error {
	for _, copyEntry := range cfg.Copies {
		input := Config{
			ProviderArgs: ProviderArgs{
				Repo: copyEntry.Source.Repo,
				Ref:  copyEntry.Source.Ref,
				Path: copyEntry.Source.Path,
			},
			DestPath:     copyEntry.Destination.Path,
			Clean:        clean,
			Status:       status,
			RemoteStatus: remoteStatus,
			Force:        force,
			CopyArgs: &ConfigCopyArgs{
				Replacements: copyEntry.Options.Replacements,
				IgnoreFiles:  copyEntry.Options.IgnoreFiles,
			},
		}

		// Run copy operation
		if err := run(&input, provider); err != nil {
			return errors.Errorf("copying %s: %w", copyEntry.Source.Repo, err)
		}
	}

	for _, archiveEntry := range cfg.Archives {
		input := Config{
			ProviderArgs: ProviderArgs{
				Repo: archiveEntry.Source.Repo,
				Ref:  archiveEntry.Source.Ref,
				Path: archiveEntry.Source.Path,
			},
			DestPath:     archiveEntry.Destination.Path,
			Clean:        clean,
			Status:       status,
			RemoteStatus: remoteStatus,
			Force:        force,
			ArchiveArgs: &ConfigArchiveArgs{
				GoEmbed: archiveEntry.Options.GoEmbed,
			},
		}

		if err := run(&input, provider); err != nil {
			return errors.Errorf("copying %s: %w", archiveEntry.Source.Repo, err)
		}
	}

	return nil
}
