package diagnostic

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

// Generator is responsible for generating diagnostics from template information
type Generator interface {
	// Generate generates diagnostics from template information
	Generate(ctx context.Context, info *parser.TemplateInfo) (*Diagnostics, error)
}

// Diagnostics represents diagnostic information that can be formatted in different ways
type Diagnostics struct {
	Errors   []Diagnostic
	Warnings []Diagnostic
}

// Diagnostic represents a single diagnostic message
type Diagnostic struct {
	Message  string
	Line     int
	Column   int
	EndLine  int
	EndCol   int
	Severity DiagnosticSeverity
}

// DiagnosticSeverity represents the severity level of a diagnostic
type DiagnosticSeverity string

const (
	Error   DiagnosticSeverity = "error"
	Warning DiagnosticSeverity = "warning"
	Info    DiagnosticSeverity = "info"
)

// Formatter formats diagnostics into different output formats
type Formatter interface {
	// Format formats diagnostics into a specific output format
	Format(diagnostics *Diagnostics) ([]byte, error)
}
