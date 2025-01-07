package diagnostic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// Generator is responsible for generating diagnostics from template information
type Generator interface {
	// Generate generates diagnostics from template information
	Generate(ctx context.Context, info *parser.TemplateInfo, typeValidator pkg_types.Validator, registry *ast.TypeRegistry) (*Diagnostics, error)
}

// Diagnostics represents diagnostic information that can be formatted in different ways
type Diagnostics struct {
	Errors   []Diagnostic
	Warnings []Diagnostic
	Hints    []Diagnostic
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
	Hint    DiagnosticSeverity = "hint"
)

// DefaultGenerator is the default implementation of Generator
type DefaultGenerator struct{}

// NewDefaultGenerator creates a new DefaultGenerator
func NewDefaultGenerator() *DefaultGenerator {
	return &DefaultGenerator{}
}

// Generate implements Generator
func (g *DefaultGenerator) Generate(ctx context.Context, info *parser.TemplateInfo, typeValidator pkg_types.Validator, registry *ast.TypeRegistry) (*Diagnostics, error) {
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

	fieldMap := make(map[string]pkg_types.FieldInfo)

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
		diagnostics.Hints = append(diagnostics.Hints, Diagnostic{
			Message:  fmt.Sprintf("Type: %v", field.Type),
			Line:     variable.Line,
			Column:   variable.Column,
			EndLine:  variable.EndLine,
			EndCol:   variable.EndCol,
			Severity: Hint,
		})
		fieldMap[variable.Name] = *field
	}

	// Validate functions
	for _, function := range info.Functions {
		method, err := typeValidator.ValidateMethod(ctx, function.Name)
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
			if len(function.MethodArguments) != len(method.Parameters) {
				diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
					Message:  fmt.Sprintf("Wrong number of arguments for method %s: expected %d, got %d", function.Name, len(method.Parameters), len(function.MethodArguments)),
					Line:     function.Line,
					Column:   function.Column,
					EndLine:  function.EndLine,
					EndCol:   function.EndCol,
					Severity: Error,
				})
				continue
			}

			for i, arg := range function.MethodArguments {
				if arg == nil {
					continue
				}

				// If the argument is a variable location, use its field type
				if varLoc, ok := arg.(*parser.VariableLocation); ok {
					field, ok := fieldMap[varLoc.Name]
					if !ok {
						diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
							Message:  fmt.Sprintf("Invalid field reference: %s", varLoc.Name),
							Line:     varLoc.Line,
							Column:   varLoc.Column,
							EndLine:  varLoc.EndLine,
							EndCol:   varLoc.EndCol,
							Severity: Error,
						})
						continue
					}

					// If it's a method call, use its return type
					if field.MethodInfo != nil && len(field.MethodInfo.Results) > 0 {
						if field.MethodInfo.Results[0].String() != method.Parameters[i].String() {
							diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
								Message:  fmt.Sprintf("Argument %d: expected %s, got %s", i+1, method.Parameters[i].String(), field.MethodInfo.Results[0].String()),
								Line:     varLoc.Line,
								Column:   varLoc.Column,
								EndLine:  varLoc.EndLine,
								EndCol:   varLoc.EndCol,
								Severity: Error,
							})
						}
					} else if field.Type.String() != method.Parameters[i].String() {
						diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
							Message:  fmt.Sprintf("Argument %d: expected %s, got %s", i+1, method.Parameters[i].String(), field.Type.String()),
							Line:     varLoc.Line,
							Column:   varLoc.Column,
							EndLine:  varLoc.EndLine,
							EndCol:   varLoc.EndCol,
							Severity: Error,
						})
					}
				} else if arg.String() != method.Parameters[i].String() {
					diagnostics.Errors = append(diagnostics.Errors, Diagnostic{
						Message:  fmt.Sprintf("Argument %d: expected %s, got %s", i+1, method.Parameters[i].String(), arg.String()),
						Line:     function.Line,
						Column:   function.Column,
						EndLine:  function.EndLine,
						EndCol:   function.EndCol,
						Severity: Error,
					})
				}
			}
		}

		// Add type information as hover info
		if method.Results != nil && len(method.Results) > 0 {
			diagnostics.Hints = append(diagnostics.Hints, Diagnostic{
				Message:  fmt.Sprintf("Returns: %v", method.Results[0]),
				Line:     function.Line,
				Column:   function.Column,
				EndLine:  function.EndLine,
				EndCol:   function.EndCol,
				Severity: Hint,
			})
		}
	}

	// // Validate definitions
	// for _, def := range info.Definitions {
	// 	if def.Name == "" {
	// 		continue // Skip unnamed definitions
	// 	}

	// 	// Check if the definition name conflicts with any fields or methods
	// 	if _, err := typeValidator.ValidateField(ctx, typeInfo, def.Name); err == nil {
	// 		diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
	// 			Message:  fmt.Sprintf("Definition name '%s' shadows a field name", def.Name),
	// 			Line:     def.Line,
	// 			Column:   def.Column,
	// 			EndLine:  def.EndLine,
	// 			EndCol:   def.EndCol,
	// 			Severity: Warning,
	// 		})
	// 	}

	// 	if _, err := typeValidator.ValidateMethod(ctx, typeInfo, def.Name); err == nil {
	// 		diagnostics.Warnings = append(diagnostics.Warnings, Diagnostic{
	// 			Message:  fmt.Sprintf("Definition name '%s' shadows a method name", def.Name),
	// 			Line:     def.Line,
	// 			Column:   def.Column,
	// 			EndLine:  def.EndLine,
	// 			EndCol:   def.EndCol,
	// 			Severity: Warning,
	// 		})
	// 	}
	// }

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

	// Convert hints
	for _, hint := range diagnostics.Hints {
		vd := VSCodeDiagnostic{
			Severity: 4, // Hint
			Message:  hint.Message,
			Range: VSCodeRange{
				Start: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      hint.Line - 1,   // VSCode is 0-based
					Character: hint.Column - 1, // VSCode is 0-based
				},
				End: struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				}{
					Line:      hint.EndLine - 1, // VSCode is 0-based
					Character: hint.EndCol - 1,  // VSCode is 0-based
				},
			},
		}
		result = append(result, vd)
	}

	return json.Marshal(result)
}
