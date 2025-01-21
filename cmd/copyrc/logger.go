package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
)

// FileStatus represents the state of a file
type FileStatus struct {
	Symbol rune
	Style  StatusStyle
	Text   string
}

type StatusStyle struct {
	SymbolColor color.Attribute
	TextColor   color.Attribute
}

// FileInfo represents a file with its status and metadata
type FileInfo struct {
	Name         string
	Status       FileStatus
	IsSpecial    bool
	IsUntracked  bool
	Replacements *int // Number of replacements made to this file
}

// FileType represents the source/type of a file
type FileType string

const (
	FileTypeManaged FileType = "managed"
	FileTypeLocal   FileType = "local"
	FileTypeCopy    FileType = "copy"
)

// RepoDisplay represents how a repository should be displayed
type RepoDisplay struct {
	Name        string
	Ref         string
	Destination string
	IsArchive   bool
	Files       []FileInfo
}

// Status definitions
var (
	RegularFile = FileStatus{
		Symbol: '•',
		Style: StatusStyle{
			SymbolColor: color.Faint,
			TextColor:   color.Faint,
		},
		Text: "no change",
	}

	SpecialFile = FileStatus{
		Symbol: '•',
		Style: StatusStyle{
			SymbolColor: color.FgCyan,
			TextColor:   color.Faint,
		},
		Text: "no change",
	}

	UntrackedFile = FileStatus{
		Symbol: '-',
		Style: StatusStyle{
			SymbolColor: color.FgYellow,
			TextColor:   color.Faint,
		},
		Text: "",
	}

	NewFile = FileStatus{
		Symbol: '✓',
		Style: StatusStyle{
			SymbolColor: color.FgGreen,
			TextColor:   color.Faint,
		},
		Text: "NEW FILE",
	}

	UpdatedFile = FileStatus{
		Symbol: '⟳',
		Style: StatusStyle{
			SymbolColor: color.FgBlue,
			TextColor:   color.Faint,
		},
		Text: "UPDATED",
	}

	RemovedFile = FileStatus{
		Symbol: '✗',
		Style: StatusStyle{
			SymbolColor: color.FgRed,
			TextColor:   color.Faint,
		},
		Text: "REMOVED",
	}
)

// Display configuration
const (
	fileIndent  = 4  // spaces to indent file entries
	nameWidth   = 35 // Base width for filename
	typeWidth   = 15 // Width for file type
	statusWidth = 15 // Width for status text
)

type Logger struct {
	zlog        zerolog.Logger
	out         io.Writer
	mu          sync.Mutex
	lastWasRepo bool
	currentRepo *RepoDisplay
	repoMu      sync.Mutex
}

func NewLogger(out io.Writer) *Logger {
	// Configure zerolog to write to a discarded writer in tests
	// This ensures our test assertions only see the formatted output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog := zerolog.New(io.Discard).With().Timestamp().Caller().Logger()

	return &Logger{
		zlog: zlog,
		out:  out,
		mu:   sync.Mutex{},
	}
}

func (l *Logger) formatFileStatus(file FileInfo) string {
	var status FileStatus
	var fileType FileType
	switch {
	case file.IsUntracked:
		status = UntrackedFile
		fileType = FileTypeLocal
	case file.IsSpecial || strings.HasSuffix(file.Name, "embed.gen.go"):
		status = SpecialFile
		fileType = FileTypeManaged
	case strings.HasSuffix(file.Name, ".tar.gz"), strings.HasSuffix(file.Name, ".copy.go"), strings.HasSuffix(file.Name, ".copy.md"):
		fileType = FileTypeCopy
		status = file.Status
	default:
		fileType = FileTypeManaged
		status = file.Status
	}

	// Build filename part
	namePart := fmt.Sprintf("%-*s", nameWidth, file.Name)

	// Build type part with optional replacements
	var typePart string
	if file.Replacements != nil && *file.Replacements > 0 {
		replacementText := fmt.Sprintf(" [%d]", *file.Replacements)
		typePart = fmt.Sprintf("%-*s", typeWidth-2, string(fileType)+replacementText)
	} else {
		typePart = fmt.Sprintf("%-*s", typeWidth-2, string(fileType))
	}

	// Build status part
	statusPart := fmt.Sprintf("%-*s", statusWidth, status.Text)

	return fmt.Sprintf("%s%s %-*s %-*s %s",
		strings.Repeat(" ", fileIndent),
		color.New(status.Style.SymbolColor).Sprint(string(status.Symbol)),
		nameWidth, namePart,
		typeWidth, color.New(status.Style.SymbolColor).Sprint(typePart),
		color.New(status.Style.TextColor).Sprint(statusPart))
}

func (l *Logger) formatArchiveTag(isArchive bool) string {
	if !isArchive {
		return ""
	}
	return fmt.Sprintf(" %s",
		color.New(color.FgCyan).Sprint("[archive]"))
}

func (l *Logger) formatRepoDisplay(repo RepoDisplay) {
	// Sort files by name
	sortedFiles := make([]FileInfo, len(repo.Files))
	copy(sortedFiles, repo.Files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Name < sortedFiles[j].Name
	})

	// Print destination header
	destPath := repo.Destination
	if repo.IsArchive {
		destPath = fmt.Sprintf("%s/%s", destPath, filepath.Base(repo.Name))
	}
	fmt.Fprintf(l.out, "[syncing %s]\n",
		color.New(color.FgCyan).Sprint(destPath))

	// Print repo header
	fmt.Fprintf(l.out, "%s %s %s %s\n",
		color.New(color.FgMagenta).Sprint("◆"),
		color.New(color.Bold).Sprint(repo.Name),
		color.New(color.Faint).Sprint("•"),
		color.New(color.FgYellow).Sprint(repo.Ref))

	// Print each file
	for _, file := range sortedFiles {
		fmt.Fprintln(l.out, l.formatFileStatus(file))
	}
}

func (l *Logger) Header(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	copyrcheaderText := color.New(color.Bold, color.FgCyan).Sprintf("copyrc")
	fmt.Fprintf(l.out, "\n%s %s\n\n", copyrcheaderText, color.New(color.Faint).Sprint("• syncing repository files"))
	l.zlog.Info().Msg(msg)
}

func (l *Logger) Operation(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if this is a repository line
	if strings.Contains(msg, "Repository:") {
		// Parse repository line
		parts := strings.Split(msg, "(ref:")
		repo := strings.TrimPrefix(strings.TrimSpace(parts[0]), "Repository:")
		ref := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
		isArchive := strings.Contains(ref, "[archive]")
		ref = strings.TrimSuffix(strings.TrimSpace(ref), "[archive]")
		ref = strings.TrimSuffix(strings.TrimSpace(ref), ")")

		// Extract destination path
		destPath := ""
		if idx := strings.Index(ref, "->"); idx != -1 {
			destPath = strings.TrimSpace(ref[idx+2:])
			ref = strings.TrimSpace(ref[:idx])
		}

		// Create RepoDisplay
		l.currentRepo = &RepoDisplay{
			Name:        strings.TrimSpace(repo),
			Ref:         strings.TrimSpace(ref),
			Destination: destPath,
			IsArchive:   isArchive,
			Files:       make([]FileInfo, 0),
		}

		l.formatRepoDisplay(*l.currentRepo)
		l.zlog.Info().
			Str("repo", l.currentRepo.Name).
			Str("ref", l.currentRepo.Ref).
			Bool("archive", l.currentRepo.IsArchive).
			Str("dest", l.currentRepo.Destination).
			Msg("Processing repository")
		return
	}

	// This is a file operation
	filename := strings.TrimPrefix(msg, "  → ")
	parts := strings.Split(filename, " ")
	filename = parts[0]

	// Determine file status
	var status FileStatus
	isSpecial := strings.HasSuffix(filename, ".copyrc.lock")
	isUntracked := strings.Contains(msg, "[untracked]")

	if len(parts) > 1 {
		switch parts[1] {
		case "FileStatusNew":
			status = NewFile
		case "FileStatusUpdated":
			status = UpdatedFile
		default:
			status = RegularFile
		}
	} else {
		status = RegularFile
	}

	// Create FileInfo
	op := FileInfo{
		Name:        filename,
		Status:      status,
		IsSpecial:   isSpecial,
		IsUntracked: isUntracked,
	}

	// Add to current repo's files
	if l.currentRepo != nil {
		l.currentRepo.Files = append(l.currentRepo.Files, op)
	}

	fmt.Fprintln(l.out, l.formatFileStatus(op))
	l.zlog.Info().
		Str("file", op.Name).
		Str("status", op.Status.Text).
		Bool("special", op.IsSpecial).
		Bool("untracked", op.IsUntracked).
		Msg("Processing file")
}

func (l *Logger) Success(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "✅ %s\n", color.New(color.FgGreen).Sprint(msg))
}

func (l *Logger) Warning(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "⚠️  %s\n", color.New(color.FgYellow).Sprint(msg))
	l.zlog.Warn().Msg(msg)
}

func (l *Logger) Error(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "❌ %s\n", color.New(color.FgRed).Sprint(msg))
	l.zlog.Error().Msg(msg)
}

func (l *Logger) Info(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "ℹ️  %s\n", color.New(color.FgCyan).Sprint(msg))
	l.zlog.Info().Msg(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	l.Warning(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *Logger) Successf(format string, args ...interface{}) {
	l.Success(fmt.Sprintf(format, args...))
}

func (l *Logger) AddFileOperation(op FileInfo) {
	l.repoMu.Lock()
	defer l.repoMu.Unlock()

	// Add to current repo's files
	if l.currentRepo != nil {
		l.currentRepo.Files = append(l.currentRepo.Files, op)
	}

	fmt.Fprintln(l.out, l.formatFileStatus(op))
	l.zlog.Info().
		Str("file", op.Name).
		Str("status", op.Status.Text).
		Msg("Processing file")
}
