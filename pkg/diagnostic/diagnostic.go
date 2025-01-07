package diagnostic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// Generator is responsible for generating diagnostics from template information
type Generator interface {
	// Generate generates diagnostics from template information
	Generate(ctx context.Context, info *parser.TemplateInfo, typeValidator types.Validator, registry *ast.TypeRegistry) (*Diagnostics, error)
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

// DefaultGenerator is the default implementation of Generator
type DefaultGenerator struct{}

// NewDefaultGenerator creates a new DefaultGenerator
func NewDefaultGenerator() *DefaultGenerator {
	return &DefaultGenerator{}
}

// Generate implements Generator
func (g *DefaultGenerator) Generate(ctx context.Context, info *parser.TemplateInfo, typeValidator types.Validator, registry *ast.TypeRegistry) (*Diagnostics, error) {
	if info == nil {
		return nil, errors.Errorf("template info is nil")
	}

	diagnostics := &Diagnostics{
		Errors:   make([]Diagnostic, 0),
		Warnings: make([]Diagnostic, 0),
	}

	// Validate type hint
	if len(info.TypeHints) == 0 {
		diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
			Message:  "No type hint found in template",
			Line:     1,
			Column:   1,
			EndLine:  1,
			EndCol:   1,
			Severity: Warning,
		})
		return diagnostics, nil
	}

	// Get type information for the hinted type
	typeInfo, err := typeValidator.ValidateType(ctx, info.TypeHints[0].TypePath, registry)
	if err != nil {
		diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
			Message:  fmt.Sprintf("Invalid type hint: %v", err),
			Line:     info.TypeHints[0].Line,
			Column:   info.TypeHints[0].Column,
			EndLine:  info.TypeHints[0].Line,
			EndCol:   info.TypeHints[0].Column + len(info.TypeHints[0].TypePath),
			Severity: Error,
		})
		return diagnostics, nil
	}

	// Validate variables
	for _, variable := range info.Variables {
		field, err := typeValidator.ValidateField(ctx, typeInfo, variable.Name)
		if err != nil {
			diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
				Message:  fmt.Sprintf("Invalid field access: %v", err),
				Line:     variable.Line,
				Column:   variable.Column,
				EndLine:  variable.EndLine,
				EndCol:   variable.EndCol,
				Severity: Error,
			})
			continue
		}

		// Add type information as hover info
		diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
			Message:  fmt.Sprintf("Type: %v", field.Type),
			Line:     variable.Line,
			Column:   variable.Column,
			EndLine:  variable.EndLine,
			EndCol:   variable.EndCol,
			Severity: Info,
		})
	}

	// Validate functions
	for _, function := range info.Functions {
		method, err := typeValidator.ValidateMethod(ctx, typeInfo, function.Name)
		if err != nil {
			diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
				Message:  fmt.Sprintf("Invalid method call: %v", err),
				Line:     function.Line,
				Column:   function.Column,
				EndLine:  function.EndLine,
				EndCol:   function.EndCol,
				Severity: Error,
			})
			continue
		}

		// Only validate arguments if the function has them
		if len(method.Parameters) > 0 {
			// If Arguments is nil, treat it as empty slice
			args := function.Arguments
			if args == nil {
				args = []string{}
			}

			if len(args) != len(method.Parameters) {
				diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
					Message:  fmt.Sprintf("Wrong number of arguments: expected %d, got %d", len(method.Parameters), len(args)),
					Line:     function.Line,
					Column:   function.Column,
					EndLine:  function.EndLine,
					EndCol:   function.EndCol,
					Severity: Error,
				})
			}
		}

		// Add method signature as hover info
		params := make([]string, len(method.Parameters))
		for i, p := range method.Parameters {
			params[i] = p.String()
		}
		results := make([]string, len(method.Results))
		for i, r := range method.Results {
			results[i] = r.String()
		}

		diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
			Message:  fmt.Sprintf("Method signature: %s(%s) (%s)", function.Name, strings.Join(params, ", "), strings.Join(results, ", ")),
			Line:     function.Line,
			Column:   function.Column,
			EndLine:  function.EndLine,
			EndCol:   function.EndCol,
			Severity: Info,
		})
	}

	// Validate definitions
	for _, def := range info.Definitions {
		if def.Name == "" {
			continue // Skip unnamed definitions
		}

		// Check if the definition name conflicts with any fields or methods
		if _, err := typeValidator.ValidateField(ctx, typeInfo, def.Name); err == nil {
			diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
				Message:  fmt.Sprintf("Definition name '%s' shadows a field name", def.Name),
				Line:     def.Line,
				Column:   def.Column,
				EndLine:  def.EndLine,
				EndCol:   def.EndCol,
				Severity: Warning,
			})
		}

		if _, err := typeValidator.ValidateMethod(ctx, typeInfo, def.Name); err == nil {
			diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
				Message:  fmt.Sprintf("Definition name '%s' shadows a method name", def.Name),
				Line:     def.Line,
				Column:   def.Column,
				EndLine:  def.EndLine,
				EndCol:   def.EndCol,
				Severity: Warning,
			})
		}
	}

	return diagnostics, nil
}

// Formatter formats diagnostics into different output formats
type Formatter interface {
	// Format formats diagnostics into a specific output format
	Format(diagnostics *Diagnostics) ([]byte, error)
}

// VSCodeFormatter formats diagnostics into VSCode-compatible format
type VSCodeFormatter struct{}

// NewVSCodeFormatter creates a new VSCodeFormatter
func NewVSCodeFormatter() *VSCodeFormatter {
	return &VSCodeFormatter{}
}

// Format implements Formatter
func (f *VSCodeFormatter) Format(diagnostics *Diagnostics) ([]byte, error) {
	if diagnostics == nil {
		return nil, errors.Errorf("diagnostics is nil")
	}

	// VSCode expects diagnostics in this format:
	// {
	//   "severity": 1, // Error = 1, Warning = 2, Information = 3
	//   "message": "message text",
	//   "range": {
	//     "start": { "line": 1, "character": 1 },
	//     "end": { "line": 1, "character": 1 }
	//   }
	// }

	type VSCodeRange struct {
		Start struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
		End struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"end"`
	}

	type VSCodeDiagnostic struct {
		Severity int         `json:"severity"`
		Message  string      `json:"message"`
		Range    VSCodeRange `json:"range"`
	}

	var result []VSCodeDiagnostic

	// Convert errors
	for _, err := range diagnostics.Errors {
		vd := VSCodeDiagnostic{
			Severity: 1, // Error
			Message:  err.Message,
			Range: VSCodeRange{
				Start: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      err.Line - 1,   // VSCode is 0-based
					Character: err.Column - 1, // VSCode is 0-based
				},
				End: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      err.EndLine - 1, // VSCode is 0-based
					Character: err.EndCol - 1,  // VSCode is 0-based
				},
			},
		}
		result = append(result, vd)
	}

	// Convert warnings
	for _, warn := range diagnostics.Warnings {
		vd := VSCodeDiagnostic{
			Severity: 2, // Warning
			Message:  warn.Message,
			Range: VSCodeRange{
				Start: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      warn.Line - 1,   // VSCode is 0-based
					Character: warn.Column - 1, // VSCode is 0-based
				},
				End: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      warn.EndLine - 1, // VSCode is 0-based
					Character: warn.EndCol - 1,  // VSCode is 0-based
				},
			},
		}
		result = append(result, vd)
	}

	return json.Marshal(result)
}
