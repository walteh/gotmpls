package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"
)

// 🏗️ Github implementation
type GithubProvider struct {
}

func NewGithubProvider() (*GithubProvider, error) {
	return &GithubProvider{}, nil
}

func parseGithubRepo(repo string) (org string, name string, err error) {
	// Remove "From " prefix if present
	repo = strings.TrimPrefix(repo, "From ")

	// Remove @ref suffix if present
	if idx := strings.LastIndex(repo, "@"); idx != -1 {
		repo = repo[:idx]
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 3 || parts[0] != "github.com" {
		return "", "", errors.Errorf("invalid github repository format: %s (expected github.com/org/repo)", repo)
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

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("fetching file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}

	if resp.StatusCode == http.StatusForbidden {
		// Check if we're rate limited
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			resetTime := resp.Header.Get("X-RateLimit-Reset")
			resetTimestamp, err := strconv.ParseInt(resetTime, 10, 64)
			if err == nil {
				waitDuration := time.Until(time.Unix(resetTimestamp, 0))
				if waitDuration > 0 {
					time.Sleep(waitDuration)
					return g.ListFiles(ctx, args) // Retry after waiting
				}
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body once
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("reading response body: %w", err)
	}

	// Try to decode as array first
	var files []struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &files); err == nil {
		result := make([]string, 0, len(files))
		for _, f := range files {
			result = append(result, f.Path)
		}
		return result, nil
	}

	// If array decode fails, try single file object
	var file struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, errors.Errorf("decoding response: %w", err)
	}

	return []string{file.Path}, nil
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
