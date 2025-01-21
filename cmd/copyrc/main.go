package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
)

// 🔄 Replacement represents a string replacement
type Replacement struct {
	Old  string  `json:"old" hcl:"old" yaml:"old" cty:"old"`
	New  string  `json:"new" hcl:"new" yaml:"new" cty:"new"`
	File *string `json:"file,omitempty" hcl:"file,optional" yaml:"file,omitempty"`
}

// 📦 Input represents raw command line input
type Input struct {
	SrcRepo      string     // Full repo URL (e.g. github.com/org/repo)
	SrcRef       string     // Branch or tag
	SrcPath      string     // Path within repo
	DestPath     string     // Local destination path
	Replacements arrayFlags // String replacements
	IgnoreFiles  arrayFlags // Files to ignore
	Clean        bool       // Whether to clean destination directory
	Status       bool       // Whether to check local status
	RemoteStatus bool       // Whether to check remote status
	Force        bool       // Whether to force update even if status is ok
	UseTarball   bool       // Whether to use tarball-based file access
	Async        bool       // Whether to process files asynchronously
}

// 🌐 RepoProvider interface for different Git providers
type RepoProvider interface {
	// ListFiles returns a list of files in the given path
	ListFiles(ctx context.Context, args ProviderArgs) ([]string, error)
	// GetCommitHash returns the commit hash for the current ref
	GetCommitHash(ctx context.Context, args ProviderArgs) (string, error)
	// GetPermalink returns a permanent link to the file
	GetPermalink(ctx context.Context, args ProviderArgs, commitHash string, file string) (string, error)
	// GetSourceInfo returns a string describing the source (e.g. "github.com/org/repo@hash")
	GetSourceInfo(ctx context.Context, args ProviderArgs, commitHash string) (string, error)
	// GetArchiveUrl returns the URL to download the repository archive
	GetArchiveUrl(ctx context.Context, args ProviderArgs) (string, error)
}

type ConfigCopyArgs struct {
	Replacements []Replacement `hcl:"replacements" yaml:"replacements" json:"replacements"`
	IgnoreFiles  []string      `hcl:"ignore_files" yaml:"ignore_files" json:"ignore_files"`
}

type ConfigArchiveArgs struct {
	GoEmbed bool `hcl:"go_embed" yaml:"go_embed"`
}

// 📦 Config holds the processed configuration
type Config struct {
	ProviderArgs ProviderArgs
	DestPath     string
	CopyArgs     *ConfigCopyArgs
	ArchiveArgs  *ConfigArchiveArgs
	Clean        bool // Whether to clean destination directory
	Status       bool // Whether to check local status
	RemoteStatus bool // Whether to check remote status
	Force        bool // Whether to force update even if status is ok
	Async        bool // Whether to process files asynchronously
}

// 🏭 Create config from input (backward compatibility)
func NewConfigFromInput(input Input, provider RepoProvider) (*Config, error) {
	replacements := make([]Replacement, 0, len(input.Replacements))
	for _, r := range input.Replacements {
		parts := strings.SplitN(r, ":", 2)
		if len(parts) == 2 {
			replacements = append(replacements, Replacement{Old: parts[0], New: parts[1]})
		}
	}

	return &Config{
		ProviderArgs: ProviderArgs{
			Repo: input.SrcRepo,
			Ref:  input.SrcRef,
			Path: input.SrcPath,
		},
		DestPath: input.DestPath,
		CopyArgs: &ConfigCopyArgs{
			Replacements: replacements,
			IgnoreFiles:  []string(input.IgnoreFiles),
		},
		Clean:        input.Clean,
		Status:       input.Status,
		RemoteStatus: input.RemoteStatus,
		Force:        input.Force,
		Async:        input.Async,
	}, nil
}

type ProviderArgs struct {
	Repo string
	Ref  string
	Path string
}

func main() {
	logger := NewLogger(os.Stdout)

	// 🎯 Parse command line flags
	var input Input
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (.copyrc)")
	flag.StringVar(&input.SrcRepo, "src-repo", "", "Source repository (e.g. github.com/org/repo)")
	flag.StringVar(&input.SrcRef, "ref", "main", "Source branch/ref")
	flag.StringVar(&input.SrcPath, "src-path", "", "Source path within repository")
	flag.StringVar(&input.DestPath, "dest-path", "", "Destination path")
	flag.Var(&input.Replacements, "replacements", "JSON array or comma-separated list of replacements in old:new format")
	flag.Var(&input.IgnoreFiles, "ignore", "JSON array or comma-separated list of files to ignore")
	flag.BoolVar(&input.Clean, "clean", false, "Clean destination directory before copying")
	flag.BoolVar(&input.Status, "status", false, "Check if files are up to date (local check only)")
	flag.BoolVar(&input.RemoteStatus, "remote-status", false, "Check if files are up to date (includes remote check)")
	flag.BoolVar(&input.Force, "force", false, "Force update even if status is ok")
	flag.BoolVar(&input.Async, "async", false, "Process files asynchronously")
	flag.BoolVar(&input.UseTarball, "use-tarball", false, "Whether to use tarball-based file access")
	flag.Parse()

	gh, err := NewGithubProvider()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// 🔍 Check if using config file
	if configFile != "" {
		cfg, err := LoadConfig(configFile)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		if err := cfg.RunAll(input.Clean, input.Status, input.RemoteStatus, input.Force, gh); err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		return
	}

	// 🔍 Validate required flags
	var missingFlags []string
	if input.SrcRepo == "" {
		missingFlags = append(missingFlags, "src-repo")
	}
	if input.SrcPath == "" {
		missingFlags = append(missingFlags, "src-path")
	}
	if input.DestPath == "" {
		missingFlags = append(missingFlags, "dest-path")
	}

	if len(missingFlags) > 0 {
		logger.Error(fmt.Sprintf("Required flags missing: %s", strings.Join(missingFlags, ", ")))
		flag.Usage()
		os.Exit(1)
	}

	// 🚀 Run the copy operation
	cfg, err := NewConfigFromInput(input, gh)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	if err := run(cfg, gh); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(cfg *Config, provider RepoProvider) error {
	logger := NewLogger(os.Stdout)

	// Determine status file location based on mode
	var statusFile string
	if cfg.ArchiveArgs != nil {
		repoDir := filepath.Join(cfg.DestPath, filepath.Base(cfg.ProviderArgs.Repo))
		statusFile = filepath.Join(repoDir, ".copyrc.lock")
		// Create repo directory if it doesn't exist
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return errors.Errorf("creating repo directory: %w", err)
		}
	} else {
		statusFile = filepath.Join(cfg.DestPath, ".copyrc.lock")
	}

	status, err := loadStatusFile(statusFile)
	if err != nil || status == nil {
		status = &StatusFile{
			CoppiedFiles:   make(map[string]StatusEntry),
			GeneratedFiles: make(map[string]GeneratedFileEntry),
			Args: StatusFileArgs{
				SrcRepo: cfg.ProviderArgs.Repo,
				SrcRef:  cfg.ProviderArgs.Ref,
				SrcPath: cfg.ProviderArgs.Path,
			},
		}
	}

	// Compare arguments
	argsAreSame := status.Args.SrcRepo == cfg.ProviderArgs.Repo &&
		status.Args.SrcRef == cfg.ProviderArgs.Ref &&
		status.Args.SrcPath == cfg.ProviderArgs.Path

	// Compare copy arguments if they exist
	if status.Args.CopyArgs != nil && cfg.CopyArgs != nil {
		// Compare replacements
		if len(status.Args.CopyArgs.Replacements) != len(cfg.CopyArgs.Replacements) {
			argsAreSame = false
		} else {
			for i, r := range status.Args.CopyArgs.Replacements {
				if r.Old != cfg.CopyArgs.Replacements[i].Old ||
					r.New != cfg.CopyArgs.Replacements[i].New {
					argsAreSame = false
					break
				}
			}
		}

		// Compare ignore files
		if len(status.Args.CopyArgs.IgnoreFiles) != len(cfg.CopyArgs.IgnoreFiles) {
			argsAreSame = false
		} else {
			for i, f := range status.Args.CopyArgs.IgnoreFiles {
				if f != cfg.CopyArgs.IgnoreFiles[i] {
					argsAreSame = false
					break
				}
			}
		}
	}

	// Check if arguments have changed
	if (cfg.Status || cfg.RemoteStatus) && !cfg.Force {
		if !argsAreSame {
			return errors.New("configuration has changed")
		}
		// For local status check, we're done
		if cfg.Status && !cfg.RemoteStatus {
			return nil
		}
	}

	if cfg.Clean {
		for _, entry := range status.CoppiedFiles {
			logger.Operation(fmt.Sprintf("  Removing %s", entry.File))
		}
		for _, entry := range status.GeneratedFiles {
			logger.Operation(fmt.Sprintf("  Removing %s", entry.File))
		}
		if err := cleanDestination(status, cfg.DestPath); err != nil {
			return errors.Errorf("cleaning destination: %w", err)
		}
		return nil
	}

	commitHash, err := provider.GetCommitHash(context.Background(), cfg.ProviderArgs)
	if err != nil {
		return errors.Errorf("getting commit hash: %w", err)
	}

	if !cfg.Force && !cfg.Clean && status.CommitHash != "" {
		if status.CommitHash == commitHash && argsAreSame {
			return nil
		}
		if cfg.Status || cfg.RemoteStatus {
			return errors.New("files are out of date")
		}
	}

	// Reset processed files map for each repository
	processedFiles = sync.Map{}

	var mu sync.Mutex
	if err := processDirectory(context.Background(), provider, cfg, commitHash, status, &mu); err != nil {
		return errors.Errorf("processing directory: %w", err)
	}

	status.LastUpdated = time.Now()
	status.CommitHash = commitHash
	status.Ref = cfg.ProviderArgs.Ref
	status.Args = StatusFileArgs{
		SrcRepo:     cfg.ProviderArgs.Repo,
		SrcRef:      cfg.ProviderArgs.Ref,
		SrcPath:     cfg.ProviderArgs.Path,
		CopyArgs:    cfg.CopyArgs,
		ArchiveArgs: cfg.ArchiveArgs,
	}

	if err := writeStatusFile(statusFile, status); err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	// Add final newline
	fmt.Println()

	return nil
}

// processDirectory is defined in process.go

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	// Try to parse as JSON array first
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		var arr []string
		d := json.NewDecoder(strings.NewReader(value))
		d.UseNumber() // To prevent numbers from being converted to float64
		if err := d.Decode(&arr); err != nil {
			return errors.Errorf("unmarshalling json: %w", err)
		}
		*i = arr
		return nil
	}

	// If not JSON, treat as comma-separated list
	if strings.Contains(value, ",") {
		*i = strings.Split(value, ",")
		return nil
	}

	// Single value
	*i = append(*i, value)
	return nil
}
