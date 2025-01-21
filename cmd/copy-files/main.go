package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

// 🎨 Colors for different types of output
var (
	success = color.New(color.FgGreen, color.Bold).SprintFunc()
	info    = color.New(color.FgCyan).SprintFunc()
	warn    = color.New(color.FgYellow).SprintFunc()
	errfmt  = color.New(color.FgRed, color.Bold).SprintFunc()
	emoji   = color.New(color.FgHiWhite).SprintFunc()
)

// 📝 Status file entry
type StatusEntry struct {
	File       string    `json:"file"`
	Source     string    `json:"source"`
	Permalink  string    `json:"permalink"`
	Downloaded time.Time `json:"downloaded"`
	Changes    []string  `json:"changes,omitempty"`
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
	LastUpdated time.Time              `json:"last_updated"`
	CommitHash  string                 `json:"commit_hash"`
	Ref         string                 `json:"branch"`
	Entries     map[string]StatusEntry `json:"entries"`
	Warnings    []string               `json:"warnings,omitempty"`
	// 📝 Store command arguments for change detection
	Args StatusFileArgs `json:"args"`
}

// 🔄 Replacement represents a string replacement
type Replacement struct {
	Old string `json:"old" hcl:"old" yaml:"old" cty:"old"`
	New string `json:"new" hcl:"new" yaml:"new" cty:"new"`
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
	Replacements []Replacement `hcl:"replacements" yaml:"replacements"`
	IgnoreFiles  []string      `hcl:"ignore_files" yaml:"ignore_files"`
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
}

// 🏭 Provider factory

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
	}, nil
}

type ProviderArgs struct {
	Repo string
	Ref  string
	Path string
}

// 🏗️ Github implementation
type GithubProvider struct {
}

func NewGithubProvider() (*GithubProvider, error) {

	return &GithubProvider{}, nil
}

func parseGithubRepo(repo string) (org string, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 3 {
		return "", "", errors.Errorf("invalid github repository format: %s", repo)
	}
	return parts[1], parts[2], nil
}

func (g *GithubProvider) ListFiles(ctx context.Context, args ProviderArgs) ([]string, error) {
	org, repo, err := parseGithubRepo(args.Repo)
	if err != nil {
		return nil, errors.Errorf("parsing github repository: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		org, repo, args.Path, args.Ref)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("fetching file list: %w", err)
	}
	defer resp.Body.Close()

	var files []struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, errors.Errorf("decoding response: %w", err)
	}

	result := make([]string, 0, len(files))
	for _, f := range files {
		result = append(result, f.Path)
	}
	return result, nil
}

func (g *GithubProvider) GetCommitHash(ctx context.Context, args ProviderArgs) (string, error) {
	// Try the specified ref first
	hash, err := g.tryGetCommitHash(ctx, args)
	if err == nil {
		return hash, nil
	}

	return "", errors.Errorf("getting commit hash: %w", err)
}

func (g *GithubProvider) tryGetCommitHash(ctx context.Context, args ProviderArgs) (string, error) {
	org, repo, err := parseGithubRepo(args.Repo)
	if err != nil {
		return "", errors.Errorf("parsing github repository: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "ls-remote",
		fmt.Sprintf("https://github.com/%s/%s.git", org, repo),
		args.Ref)

	out, err := cmd.Output()
	if err != nil {
		return "", errors.Errorf("running git ls-remote: %w", err)
	}

	parts := strings.Fields(string(out))
	if len(parts) == 0 {
		return "", errors.New("no commit hash found")
	}

	return parts[0], nil
}

func (g *GithubProvider) GetPermalink(ctx context.Context, args ProviderArgs, commitHash string, file string) (string, error) {
	org, repo, err := parseGithubRepo(args.Repo)
	if err != nil {
		return "", errors.Errorf("parsing github repository: %w", err)
	}
	if file == "" && args.Path == "" {
		// archive permalink
		url, err := g.GetArchiveUrl(ctx, args)
		if err != nil {
			return "", errors.Errorf("getting archive url: %w", err)
		}
		return url, nil
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		org, repo, commitHash, file), nil
}

func (g *GithubProvider) GetSourceInfo(ctx context.Context, args ProviderArgs, commitHash string) (string, error) {
	org, repo, err := parseGithubRepo(args.Repo)
	if err != nil {
		return "", errors.Errorf("parsing github repository: %w", err)
	}
	return fmt.Sprintf("github.com/%s/%s@%s", org, repo, commitHash), nil
}

// GetArchiveUrl returns the URL to download the repository archive
func (g *GithubProvider) GetArchiveUrl(ctx context.Context, args ProviderArgs) (string, error) {
	org, repo, err := parseGithubRepo(args.Repo)
	if err != nil {
		return "", errors.Errorf("parsing github repository: %w", err)
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/%s.tar.gz",
		org, repo, args.Ref), nil
}

func main() {
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
	flag.Parse()

	gh, err := NewGithubProvider()
	if err != nil {
		fmt.Printf("%s %v\n", errfmt("❌"), err)
		os.Exit(1)
	}

	// 🔍 Check if using config file
	if configFile != "" {
		cfg, err := LoadConfig(configFile)
		if err != nil {
			fmt.Printf("%s %v\n", errfmt("❌"), err)
			os.Exit(1)
		}

		if err := cfg.RunAll(input.Clean, input.Status, input.RemoteStatus, input.Force, gh); err != nil {
			fmt.Printf("%s %v\n", errfmt("❌"), err)
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
		fmt.Printf("%s Required flags missing: %s\n", errfmt("❌"), strings.Join(missingFlags, ", "))
		flag.Usage()
		os.Exit(1)
	}

	// 🚀 Run the copy operation
	cfg, err := NewConfigFromInput(input, gh)
	if err != nil {
		fmt.Printf("%s %v\n", errfmt("❌"), err)
		os.Exit(1)
	}

	if err := run(cfg, gh); err != nil {
		fmt.Printf("%s %v\n", errfmt("❌"), err)
		os.Exit(1)
	}
}

func run(cfg *Config, provider RepoProvider) error {
	ctx := context.Background()

	var statusFile string
	if cfg.ArchiveArgs != nil {
		statusFile = filepath.Join(cfg.DestPath, filepath.Base(cfg.ProviderArgs.Repo), ".copy-status")
	} else {
		statusFile = filepath.Join(cfg.DestPath, ".copy-status")
	}

	fmt.Printf("🔍 Loading status file from: %s\n", info(statusFile))
	status, err := loadStatusFile(statusFile)
	if err != nil {
		fmt.Printf("📝 Creating new status file (error loading: %v)\n", warn(err))
		status = &StatusFile{
			Entries: make(map[string]StatusEntry),
		}
	}

	// 🔍 Check if arguments have changed
	if cfg.Status || cfg.RemoteStatus {
		if cfg.Force {
			return errors.New("force flag is set on status check, throwing error")
		}

		// Check if any arguments have changed
		fmt.Printf("🔍 Comparing arguments:\n")
		fmt.Printf("  - Repo: %s vs %s\n", info(status.Args.SrcRepo), info(cfg.ProviderArgs.Repo))
		fmt.Printf("  - Ref: %s vs %s\n", info(status.Args.SrcRef), info(cfg.ProviderArgs.Ref))

		fmt.Printf("  - Path: %s vs %s\n", info(status.Args.SrcPath), info(cfg.ProviderArgs.Path))
		var argsAreSame bool = false
		if status.Args.ArchiveArgs != nil {
			argsAreSame = (status.Args.ArchiveArgs.GoEmbed == cfg.ArchiveArgs.GoEmbed)
			fmt.Printf("  - Archive Args: %v vs %v - %v\n", info(status.Args.ArchiveArgs.GoEmbed), info(cfg.ArchiveArgs.GoEmbed), status.Args.ArchiveArgs.GoEmbed == cfg.ArchiveArgs.GoEmbed)
		}
		if status.Args.CopyArgs != nil {
			argsAreSame = (slices.Equal(status.Args.CopyArgs.Replacements, cfg.CopyArgs.Replacements) &&
				slices.Equal(status.Args.CopyArgs.IgnoreFiles, cfg.CopyArgs.IgnoreFiles))
			fmt.Printf("  - Copy Args: %v vs %v - %v\n", info(status.Args.CopyArgs.Replacements), info(cfg.CopyArgs.Replacements), slices.Equal(status.Args.CopyArgs.Replacements, cfg.CopyArgs.Replacements))
			fmt.Printf("  - Copy Args: %v vs %v - %v\n", info(status.Args.CopyArgs.IgnoreFiles), info(cfg.CopyArgs.IgnoreFiles), slices.Equal(status.Args.CopyArgs.IgnoreFiles, cfg.CopyArgs.IgnoreFiles))
		}

		fmt.Printf("🔍 Args are same: %v\n", argsAreSame)

		if status.Args.SrcRepo != cfg.ProviderArgs.Repo ||
			status.Args.SrcRef != cfg.ProviderArgs.Ref ||
			status.Args.SrcPath != cfg.ProviderArgs.Path ||
			!argsAreSame {
			return errors.New("configuration has changed")
		}

		// For local status check, we're done
		if cfg.Status && !cfg.RemoteStatus {
			return nil
		}

	}

	// 🧹 Clean if requested
	if cfg.Clean {
		fmt.Printf("🧹 Cleaning destination directory: %s\n", info(cfg.DestPath))
		if err := cleanDestination(cfg.DestPath); err != nil {
			return errors.Errorf("cleaning destination: %w", err)
		}
	}

	// 🔍 Get commit hash (only for remote status or actual sync)
	if !cfg.Status || cfg.RemoteStatus {
		fmt.Printf("🔍 Getting commit hash...\n")
		commitHash, err := provider.GetCommitHash(ctx, cfg.ProviderArgs)
		if err != nil {
			if cfg.RemoteStatus {
				fmt.Fprintf(os.Stderr, "%s Unable to check remote status: %v\n", warn("⚠️"), err)
				return nil // Not an error for status check
			}
			return errors.Errorf("getting commit hash: %w", err)
		}
		fmt.Printf("📌 Commit hash: %s\n", info(commitHash))

		// Check if files are up to date
		if (cfg.Status || cfg.RemoteStatus) && !cfg.Force {
			fmt.Printf("🔍 Comparing commit hashes: %s vs %s\n", info(status.CommitHash), info(commitHash))
			if status.CommitHash == commitHash {
				return nil
			}
			return errors.New("files are out of date")
		}

		if cfg.ArchiveArgs != nil {
			var mu sync.Mutex
			err := processFile(ctx, provider, cfg, "", commitHash, status, &mu)
			if err != nil {
				return errors.Errorf("processing file: %w", err)
			}

		} else {
			files, err := provider.ListFiles(ctx, cfg.ProviderArgs)
			if err != nil {
				return errors.Errorf("listing files: %w", err)
			}
			fmt.Printf("📋 Found %d files:\n", len(files))
			for _, f := range files {
				fmt.Printf("  - %s\n", info(f))
			}

			// 🔄 Process each file
			g, ctx := errgroup.WithContext(ctx)
			var mu sync.Mutex // For status file access
			for _, file := range files {
				file := file // capture for goroutine
				g.Go(func() error {
					fmt.Printf("🔄 Starting to process file: %s\n", info(file))
					err := processFile(ctx, provider, cfg, file, commitHash, status, &mu)
					if err != nil {
						fmt.Printf("❌ Error processing %s: %v\n", errfmt(file), err)
					} else {
						fmt.Printf("✅ Successfully processed %s\n", success(file))
					}
					return err
				})
			}

			if err := g.Wait(); err != nil {
				return err
			}
		}

		// Update status file metadata
		status.LastUpdated = time.Now().UTC()
		status.CommitHash = commitHash
		status.Ref = cfg.ProviderArgs.Ref
		fmt.Printf("📝 Updated status file metadata:\n")
		fmt.Printf("  - Last Updated: %s\n", info(status.LastUpdated))
		fmt.Printf("  - Commit Hash: %s\n", info(status.CommitHash))
		fmt.Printf("  - Branch: %s\n", info(status.Ref))
	}

	status.Args.SrcRepo = cfg.ProviderArgs.Repo
	status.Args.SrcPath = cfg.ProviderArgs.Path
	status.Args.SrcRef = cfg.ProviderArgs.Ref
	if cfg.CopyArgs != nil {
		if status.Args.CopyArgs == nil {
			status.Args.CopyArgs = &ConfigCopyArgs{}
		}
		status.Args.CopyArgs.Replacements = make([]Replacement, len(cfg.CopyArgs.Replacements))
		for i, r := range cfg.CopyArgs.Replacements {
			status.Args.CopyArgs.Replacements[i] = Replacement{Old: r.Old, New: r.New}
		}
		if cfg.CopyArgs != nil {
			status.Args.CopyArgs.IgnoreFiles = make([]string, len(cfg.CopyArgs.IgnoreFiles))
			copy(status.Args.CopyArgs.IgnoreFiles, cfg.CopyArgs.IgnoreFiles)
		}
	}

	if cfg.ArchiveArgs != nil {
		if status.Args.ArchiveArgs == nil {
			status.Args.ArchiveArgs = &ConfigArchiveArgs{}
		}
		status.Args.ArchiveArgs.GoEmbed = cfg.ArchiveArgs.GoEmbed
	}

	// Write final status
	fmt.Printf("💾 Writing status file to: %s\n", info(statusFile))
	if err := writeStatusFile(statusFile, status); err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	if !cfg.Status && !cfg.RemoteStatus {
		fmt.Printf("\n%s Successfully processed %d files\n", emoji("✨"), len(status.Entries))
		fmt.Printf("%s See %s for detailed information\n", emoji("📝"), info(statusFile))
	}
	return nil
}

// 🧹 Clean destination directory
func cleanDestination(destPath string) error {
	entries, err := os.ReadDir(destPath)
	if err != nil {
		return errors.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, ".copy.") && !strings.Contains(name, ".patch.") {
			if err := os.Remove(filepath.Join(destPath, name)); err != nil {
				return errors.Errorf("removing file: %w", err)
			}
		}
	}
	return nil
}

// 📝 Load status file
func loadStatusFile(path string) (*StatusFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var status StatusFile
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, errors.Errorf("parsing status file: %w", err)
	}

	return &status, nil
}

// 📝 Write status file
func writeStatusFile(path string, status *StatusFile) error {
	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return errors.Errorf("marshaling status: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	return nil
}

func processFile(ctx context.Context, provider RepoProvider, cfg *Config, file, commitHash string, status *StatusFile, mu *sync.Mutex) error {
	fmt.Printf("%s Processing %s\n", emoji("📥"), info(file))

	if cfg.ArchiveArgs != nil {

		if cfg.ProviderArgs.Path != "" {
			return errors.New("path is not supported in tarball mode")
		}

		// Ensure cache directory exists
		fmt.Printf("📁 Using tarball mode with cache directory: %s\n", info(cfg.DestPath))

		// Create repo-specific directory
		repoName := filepath.Base(cfg.ProviderArgs.Repo)
		repoDir := filepath.Join(cfg.DestPath, repoName)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return errors.Errorf("creating repo directory: %w", err)
		}

		// Download tarball
		data, err := GetFileFromTarball(ctx, provider, cfg.ProviderArgs)
		if err != nil {
			return errors.Errorf("getting file from tarball: %w", err)
		}
		fmt.Printf("📥 Downloaded %d bytes\n", len(data))

		// Save tarball
		tarballPath := filepath.Join(repoDir, repoName+".tar.gz")
		if err := os.WriteFile(tarballPath, data, 0644); err != nil {
			return errors.Errorf("writing tarball: %w", err)
		}

		sourceInfo, err := provider.GetSourceInfo(ctx, cfg.ProviderArgs, commitHash)
		if err != nil {
			return errors.Errorf("getting source info: %w", err)
		}

		permalink, err := provider.GetArchiveUrl(ctx, cfg.ProviderArgs)
		if err != nil {
			return errors.Errorf("getting permalink: %w", err)
		}

		if cfg.ArchiveArgs.GoEmbed {
			// Create embed.go file
			pkgName := strings.ReplaceAll(repoName, "-", "")
			embedPath := filepath.Join(repoDir, "embed.gen.go")
			var buf bytes.Buffer

			fmt.Fprintf(&buf, "// 📦 generated by copyrc. DO NOT EDIT.\n")
			fmt.Fprintf(&buf, "// ℹ️ see .copy-status for more details.\n\n")
			fmt.Fprintf(&buf, "package %s\n\n", pkgName)
			fmt.Fprintf(&buf, "import _ \"embed\"\n\n")
			fmt.Fprintf(&buf, "//go:embed %s.tar.gz\n", repoName)
			fmt.Fprintf(&buf, "var Data []byte\n\n")
			fmt.Fprintf(&buf, "// Metadata about the downloaded repository\n")
			fmt.Fprintf(&buf, "var (\n")
			fmt.Fprintf(&buf, "\tRef        = %q\n", cfg.ProviderArgs.Ref)
			fmt.Fprintf(&buf, "\tCommit     = %q\n", commitHash)
			fmt.Fprintf(&buf, "\tRepository = %q\n", cfg.ProviderArgs.Repo)
			fmt.Fprintf(&buf, "\tPermalink  = %q\n", permalink)
			fmt.Fprintf(&buf, "\tDownloaded = %q\n", time.Now().UTC().Format(time.RFC3339))
			fmt.Fprintf(&buf, ")\n")

			if err := os.WriteFile(embedPath, buf.Bytes(), 0644); err != nil {
				return errors.Errorf("writing embed.gen.go: %w", err)
			}
		}
		// Create status entry
		entry := StatusEntry{
			File:       repoName + ".tar.gz",
			Source:     sourceInfo,
			Permalink:  permalink,
			Downloaded: time.Now().UTC(),
			Changes:    []string{"generated embed.gen.go file"},
		}

		// Update status file (with mutex for concurrent access)
		mu.Lock()
		status.Entries[entry.File] = entry
		mu.Unlock()
		fmt.Printf("📝 Updated status entry for %s\n", info(entry.File))

		return nil
	}

	if cfg.CopyArgs == nil {
		return errors.New("copy args are required")
	}
	sourceInfo, err := provider.GetSourceInfo(ctx, cfg.ProviderArgs, commitHash)
	if err != nil {
		return errors.Errorf("getting source info: %w", err)
	}

	permalink, err := provider.GetPermalink(ctx, cfg.ProviderArgs, commitHash, file)
	if err != nil {
		return errors.Errorf("getting permalink: %w", err)
	}

	var contentz []byte
	if strings.HasPrefix(permalink, "file://") {
		fmt.Printf("📥 Downloading file directly from provider\n")
		contentz, err = os.ReadFile(strings.TrimPrefix(permalink, "file://"))
		if err != nil {
			return errors.Errorf("reading file: %w", err)
		}
	} else {
		fmt.Printf("📥 Downloading file from provider\n")
		req, err := http.NewRequestWithContext(ctx, "GET", permalink, nil)
		if err != nil {
			return errors.Errorf("creating request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Errorf("downloading file: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("downloading file: %s", resp.Status)
		}

		contentz, err = io.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("reading file: %w", err)
		}
	}

	fmt.Printf("📥 Downloaded %d bytes\n", len(contentz))

	// Get base name and extension
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(filepath.Base(file), ext)
	fmt.Printf("📝 Base: %s, Extension: %s\n", info(base), info(ext))

	// 📝 Create status entry
	entry := StatusEntry{
		File:       base + ".copy" + ext,
		Source:     sourceInfo,
		Permalink:  permalink,
		Downloaded: time.Now().UTC(),
	}
	fmt.Printf("📝 Created status entry:\n")
	fmt.Printf("  - Output file: %s\n", info(entry.File))
	fmt.Printf("  - Source: %s\n", info(entry.Source))
	fmt.Printf("  - Permalink: %s\n", info(entry.Permalink))

	// 📦 Process content
	var buf bytes.Buffer

	// Add file header based on extension
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".cpp", ".c", ".h", ".hpp", ".java", ".scala", ".rs", ".php", "jsonc":
		fmt.Printf("📝 Adding header for %s file\n", info(ext))
		fmt.Fprintf(&buf, "// 📦 generated by copyrc. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "// 🔗 source: %s\n", permalink)
		fmt.Fprintf(&buf, "// ℹ️ see .copy-status for more details.\n\n")
	case ".py", ".rb", ".pl", ".sh", ".yaml", ".yml":
		fmt.Printf("📝 Adding header for %s file\n", info(ext))
		fmt.Fprintf(&buf, "# 📦 generated by copyrc. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "# 🔗 source: %s\n", permalink)
		fmt.Fprintf(&buf, "# ℹ️ see .copy-status for more details.\n\n")
	case ".md", ".xml":
		fmt.Printf("📝 Adding header for %s file\n", info(ext))
		fmt.Fprintf(&buf, "<!--\n")
		fmt.Fprintf(&buf, "📦 generated by copyrc. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "🔗 source: %s\n", permalink)
		fmt.Fprintf(&buf, "ℹ️ see .copy-status for more details.\n")
		fmt.Fprintf(&buf, "-->\n\n")
	}

	// Add package declaration for Go files
	if ext == ".go" && !bytes.Contains(contentz, []byte("package ")) {
		pkgName := filepath.Base(cfg.DestPath)
		fmt.Printf("📦 Adding package declaration: %s\n", info(pkgName))
		fmt.Fprintf(&buf, "package %s\n\n", pkgName)
		entry.Changes = append(entry.Changes, fmt.Sprintf("Added package declaration: %s", pkgName))
	}

	// Write original content
	buf.Write(contentz)

	// Apply replacements for Go files
	if ext == ".go" {
		fmt.Printf("🔄 Applying %d replacements\n", len(cfg.CopyArgs.Replacements))
		for _, r := range cfg.CopyArgs.Replacements {
			if bytes.Contains(buf.Bytes(), []byte(r.Old)) {
				// Find line numbers for the changes
				lines := bytes.Split(buf.Bytes(), []byte("\n"))
				for i, line := range lines {
					if bytes.Contains(line, []byte(r.Old)) {
						change := fmt.Sprintf("Line %d: Replaced '%s' with '%s'", i+1, r.Old, r.New)
						fmt.Printf("  - %s\n", info(change))
						entry.Changes = append(entry.Changes, change)
					}
				}

				// Apply the replacement
				newContent := bytes.ReplaceAll(buf.Bytes(), []byte(r.Old), []byte(r.New))
				buf.Reset()
				buf.Write(newContent)
			}
		}
	}

	// Check if file exists and has .patch suffix
	outPath := filepath.Join(cfg.DestPath, entry.File)
	patchPath := filepath.Join(cfg.DestPath, base+".copy.patch"+ext)
	fmt.Printf("📝 Output path: %s\n", info(outPath))
	if _, err := os.Stat(patchPath); err == nil {
		// Skip files that have a .patch version
		fmt.Printf("%s Skipping %s (has .patch file)\n", emoji("⚠️"), warn(entry.File))
		return nil
	}

	// Write the file
	fmt.Printf("💾 Writing %d bytes to %s\n", buf.Len(), info(outPath))
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return errors.Errorf("writing file: %w", err)
	}

	// Update status file (with mutex for concurrent access)
	mu.Lock()
	status.Entries[entry.File] = entry
	mu.Unlock()
	fmt.Printf("📝 Updated status entry for %s\n", info(entry.File))

	fmt.Printf("%s Processed %s\n", emoji("✅"), success(entry.File))
	return nil
}

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

// Helper function to compare string slices
func stringSlicesEqual(a []string, b interface{}) bool {
	switch v := b.(type) {
	case []string:
		if len(a) != len(v) {
			return false
		}
		for i := range a {
			if a[i] != v[i] {
				return false
			}
		}
		return true
	case []Replacement:
		if len(a) != len(v) {
			return false
		}
		for i := range a {
			if a[i] != v[i].Old+":"+v[i].New {
				return false
			}
		}
		return true
	default:
		return false
	}
}
