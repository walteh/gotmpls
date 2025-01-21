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

// 📦 Status file structure
type StatusFile struct {
	LastUpdated time.Time              `json:"last_updated"`
	CommitHash  string                 `json:"commit_hash"`
	Branch      string                 `json:"branch"`
	Entries     map[string]StatusEntry `json:"entries"`
	Warnings    []string               `json:"warnings,omitempty"`
	// 📝 Store command arguments for change detection
	Args struct {
		SrcRepo      string   `json:"src_repo"`
		SrcRef       string   `json:"src_ref"`
		SrcPath      string   `json:"src_path"`
		Replacements []string `json:"replacements"`
		IgnoreFiles  []string `json:"ignore_files"`
	} `json:"args"`
}

// 🔄 Replacement represents a string replacement
type Replacement struct {
	Old string
	New string
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
}

// 🌐 RepoProvider interface for different Git providers
type RepoProvider interface {
	// ListFiles returns a list of files in the given path
	ListFiles(ctx context.Context) ([]string, error)
	// GetFile downloads a specific file
	GetFile(ctx context.Context, path string) ([]byte, error)
	// GetCommitHash returns the commit hash for the current ref
	GetCommitHash(ctx context.Context) (string, error)
	// GetPermalink returns a permanent link to the file
	GetPermalink(path, commitHash string) string
	// GetSourceInfo returns a string describing the source (e.g. "github.com/org/repo@hash")
	GetSourceInfo(commitHash string) string
}

// 🏭 Provider factory
func NewProvider(repo, ref, path string) (RepoProvider, error) {
	// Support mock provider in tests
	if strings.HasPrefix(repo, "github.com/org/repo") {
		mock := NewMockProvider()
		mock.ref = ref
		mock.path = path
		return mock, nil
	}

	if strings.HasPrefix(repo, "github.com/") {
		return NewGithubProvider(repo, ref, path)
	}
	return nil, errors.Errorf("unsupported repository host: %s", repo)
}

// 📦 Config holds the processed configuration
type Config struct {
	Provider     RepoProvider
	DestPath     string
	Replacements []Replacement
	IgnoreFiles  []string
	Clean        bool   // Whether to clean destination directory
	Status       bool   // Whether to check local status
	RemoteStatus bool   // Whether to check remote status
	Force        bool   // Whether to force update even if status is ok
	SrcRef       string // Source branch/ref
}

func NewConfigFromInput(input Input) (*Config, error) {
	provider, err := NewProvider(input.SrcRepo, input.SrcRef, input.SrcPath)
	if err != nil {
		return nil, errors.Errorf("creating provider: %w", err)
	}

	// Set fallback branch if using GithubProvider
	if gh, ok := provider.(*GithubProvider); ok {
		gh.fallbackBranch = "master" // Default fallback
	}

	replacements := make([]Replacement, 0, len(input.Replacements))
	for _, r := range input.Replacements {
		parts := strings.SplitN(r, ":", 2)
		if len(parts) == 2 {
			replacements = append(replacements, Replacement{Old: parts[0], New: parts[1]})
		}
	}

	return &Config{
		Provider:     provider,
		DestPath:     input.DestPath,
		Replacements: replacements,
		IgnoreFiles:  []string(input.IgnoreFiles),
		Clean:        input.Clean,
		Status:       input.Status,
		RemoteStatus: input.RemoteStatus,
		Force:        input.Force,
		SrcRef:       input.SrcRef,
	}, nil
}

// 🏗️ Github implementation
type GithubProvider struct {
	org            string // Parsed from repo URL
	repo           string // Parsed from repo URL
	ref            string
	path           string
	fallbackBranch string // Fallback branch if ref doesn't exist
}

func NewGithubProvider(repo, ref, path string) (*GithubProvider, error) {
	// Remove github.com/ prefix
	repoPath := strings.TrimPrefix(repo, "github.com/")
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return nil, errors.Errorf("invalid github repository format: %s", repo)
	}

	return &GithubProvider{
		org:            parts[0],
		repo:           parts[1],
		ref:            ref,
		path:           path,
		fallbackBranch: "master", // Default fallback
	}, nil
}

func (g *GithubProvider) ListFiles(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		g.org, g.repo, g.path, g.ref)

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

func (g *GithubProvider) GetFile(ctx context.Context, path string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		g.org, g.repo, g.ref, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (g *GithubProvider) GetCommitHash(ctx context.Context) (string, error) {
	// Try the specified ref first
	hash, err := g.tryGetCommitHash(ctx, g.ref)
	if err == nil {
		return hash, nil
	}

	// If ref is the default branch, try fallback
	if g.ref == "main" || g.ref == "master" {
		hash, err = g.tryGetCommitHash(ctx, g.fallbackBranch)
		if err == nil {
			g.ref = g.fallbackBranch // Update ref to the working one
			return hash, nil
		}
	}

	return "", errors.Errorf("getting commit hash: %w", err)
}

func (g *GithubProvider) tryGetCommitHash(ctx context.Context, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote",
		fmt.Sprintf("https://github.com/%s/%s.git", g.org, g.repo),
		ref)

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

func (g *GithubProvider) GetPermalink(path, commitHash string) string {
	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s",
		g.org, g.repo, commitHash, path)
}

func (g *GithubProvider) GetSourceInfo(commitHash string) string {
	return fmt.Sprintf("github.com/%s/%s@%s", g.org, g.repo, commitHash)
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

	// 🔍 Check if using config file
	if configFile != "" {
		cfg, err := LoadConfig(configFile)
		if err != nil {
			fmt.Printf("%s %v\n", errfmt("❌"), err)
			os.Exit(1)
		}

		if err := cfg.RunAll(input.Clean, input.Status, input.RemoteStatus, input.Force); err != nil {
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
	cfg, err := NewConfigFromInput(input)
	if err != nil {
		fmt.Printf("%s %v\n", errfmt("❌"), err)
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		fmt.Printf("%s %v\n", errfmt("❌"), err)
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	ctx := context.Background()

	// 📝 Load or initialize status file
	statusFile := filepath.Join(cfg.DestPath, ".copy-status")
	status, err := loadStatusFile(statusFile)
	if err != nil {
		status = &StatusFile{
			Entries: make(map[string]StatusEntry),
		}
	}

	// 🔍 Check if arguments have changed
	if cfg.Status || cfg.RemoteStatus {
		if !cfg.Force && status.Args.SrcRepo != "" {
			// Get provider info
			var srcRepo, srcPath string
			switch p := cfg.Provider.(type) {
			case *GithubProvider:
				srcRepo = p.org + "/" + p.repo
				srcPath = p.path
			case *MockProvider:
				srcRepo = p.org + "/" + p.repo
				srcPath = p.path
			default:
				return errors.New("unsupported provider type")
			}

			// Check if any arguments have changed
			if status.Args.SrcRepo != srcRepo ||
				status.Args.SrcRef != cfg.SrcRef ||
				status.Args.SrcPath != srcPath ||
				!stringSlicesEqual(status.Args.Replacements, cfg.Replacements) ||
				!stringSlicesEqual(status.Args.IgnoreFiles, cfg.IgnoreFiles) {
				return errors.New("configuration has changed")
			}

			// For local status check, we're done
			if cfg.Status && !cfg.RemoteStatus {
				return nil
			}
		}
	}

	// 🧹 Clean if requested
	if cfg.Clean {
		if err := cleanDestination(cfg.DestPath); err != nil {
			return errors.Errorf("cleaning destination: %w", err)
		}
	}

	// 🔍 Get commit hash (only for remote status or actual sync)
	if !cfg.Status || cfg.RemoteStatus {
		commitHash, err := cfg.Provider.GetCommitHash(ctx)
		if err != nil {
			if cfg.RemoteStatus {
				fmt.Fprintf(os.Stderr, "%s Unable to check remote status: %v\n", warn("⚠️"), err)
				return nil // Not an error for status check
			}
			return errors.Errorf("getting commit hash: %w", err)
		}

		// Check if files are up to date
		if (cfg.Status || cfg.RemoteStatus) && !cfg.Force {
			if status.CommitHash == commitHash {
				return nil
			}
			return errors.New("files are out of date")
		}

		// 📋 List files
		files, err := cfg.Provider.ListFiles(ctx)
		if err != nil {
			if cfg.RemoteStatus {
				fmt.Fprintf(os.Stderr, "%s Unable to list remote files: %v\n", warn("⚠️"), err)
				return nil // Not an error for status check
			}
			return errors.Errorf("listing files: %w", err)
		}

		// 🔄 Process each file
		g, ctx := errgroup.WithContext(ctx)
		var mu sync.Mutex // For status file access
		for _, file := range files {
			file := file // capture for goroutine
			g.Go(func() error {
				return processFile(ctx, cfg, file, commitHash, status, &mu)
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		// Update status file metadata
		status.LastUpdated = time.Now().UTC()
		status.CommitHash = commitHash
		status.Branch = cfg.SrcRef
	}

	// Update status file arguments
	switch p := cfg.Provider.(type) {
	case *GithubProvider:
		status.Args.SrcRepo = p.org + "/" + p.repo
		status.Args.SrcPath = p.path
	case *MockProvider:
		status.Args.SrcRepo = p.org + "/" + p.repo
		status.Args.SrcPath = p.path
	default:
		return errors.New("unsupported provider type")
	}
	status.Args.SrcRef = cfg.SrcRef
	status.Args.Replacements = make([]string, len(cfg.Replacements))
	for i, r := range cfg.Replacements {
		status.Args.Replacements[i] = r.Old + ":" + r.New
	}
	status.Args.IgnoreFiles = cfg.IgnoreFiles

	// Write final status
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
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling status: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.Errorf("writing status file: %w", err)
	}

	return nil
}

func processFile(ctx context.Context, cfg *Config, file, commitHash string, status *StatusFile, mu *sync.Mutex) error {
	fmt.Printf("%s Processing %s\n", emoji("📥"), info(file))

	// 📥 Download file
	content, err := cfg.Provider.GetFile(ctx, file)
	if err != nil {
		return errors.Errorf("downloading file: %w", err)
	}

	// Get base name and extension
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(filepath.Base(file), ext)

	// 📝 Create status entry
	entry := StatusEntry{
		File:       base + ".copy" + ext,
		Source:     cfg.Provider.GetSourceInfo(commitHash),
		Permalink:  cfg.Provider.GetPermalink(file, commitHash),
		Downloaded: time.Now().UTC(),
	}

	// 📦 Process content
	var buf bytes.Buffer

	// Add file header based on extension
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".cpp", ".c", ".h", ".hpp", ".java", ".scala", ".rs", ".php":
		fmt.Fprintf(&buf, "// 📦 Generated from: %s\n", entry.Source)
		fmt.Fprintf(&buf, "// 🔗 Source: %s\n", entry.Permalink)
		fmt.Fprintf(&buf, "// ⏰ Downloaded at: %s\n", entry.Downloaded.Format(time.RFC3339))
		fmt.Fprintf(&buf, "// ⚠️  This file is auto-generated. See .copy-status for details.\n\n")
	case ".py", ".rb", ".pl", ".sh":
		fmt.Fprintf(&buf, "# 📦 Generated from: %s\n", entry.Source)
		fmt.Fprintf(&buf, "# 🔗 Source: %s\n", entry.Permalink)
		fmt.Fprintf(&buf, "# ⏰ Downloaded at: %s\n", entry.Downloaded.Format(time.RFC3339))
		fmt.Fprintf(&buf, "# ⚠️  This file is auto-generated. See .copy-status for details.\n\n")
	case ".md", ".txt", ".json", ".yaml", ".yml":
		fmt.Fprintf(&buf, "<!--\n")
		fmt.Fprintf(&buf, "📦 Generated from: %s\n", entry.Source)
		fmt.Fprintf(&buf, "🔗 Source: %s\n", entry.Permalink)
		fmt.Fprintf(&buf, "⏰ Downloaded at: %s\n", entry.Downloaded.Format(time.RFC3339))
		fmt.Fprintf(&buf, "⚠️  This file is auto-generated. See .copy-status for details.\n")
		fmt.Fprintf(&buf, "-->\n\n")
	}

	// Add package declaration for Go files
	if ext == ".go" && !bytes.Contains(content, []byte("package ")) {
		pkgName := filepath.Base(cfg.DestPath)
		fmt.Fprintf(&buf, "package %s\n\n", pkgName)
		entry.Changes = append(entry.Changes, fmt.Sprintf("Added package declaration: %s", pkgName))
	}

	// Write original content
	buf.Write(content)

	// Apply replacements for Go files
	if ext == ".go" {
		for _, r := range cfg.Replacements {
			if bytes.Contains(buf.Bytes(), []byte(r.Old)) {
				// Find line numbers for the changes
				lines := bytes.Split(buf.Bytes(), []byte("\n"))
				for i, line := range lines {
					if bytes.Contains(line, []byte(r.Old)) {
						entry.Changes = append(entry.Changes,
							fmt.Sprintf("Line %d: Replaced '%s' with '%s'", i+1, r.Old, r.New))
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
	if _, err := os.Stat(patchPath); err == nil {
		// Skip files that have a .patch version
		fmt.Printf("%s Skipping %s (has .patch file)\n", emoji("⚠️"), warn(entry.File))
		return nil
	}

	// Write the file
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return errors.Errorf("writing file: %w", err)
	}

	// Update status file (with mutex for concurrent access)
	mu.Lock()
	status.Entries[entry.File] = entry
	mu.Unlock()

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
