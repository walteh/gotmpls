// Package hover provides functionality for generating hover information.
package hover

import (
	"context"
	"fmt"
	"strings"

	"go/types"

	"github.com/rs/zerolog"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// HoverInfo represents the information to be displayed in a hover tooltip
type HoverInfo struct {
	// Content is the markdown content to display
	Content []string
	// Range is the range in the document that this hover applies to
	Position position.RawPosition
}

// FormatHoverResponse formats a hover response for a variable or function
func FormatHoverResponse(ctx context.Context, variable *parser.VariableLocation, method *ast.TemplateMethodInfo, typeInfo *ast.FieldInfo) (*HoverInfo, error) {
	if variable == nil {
		return nil, errors.New("variable cannot be nil")
	}

	var content []string

	// If it's a function call with method info
	if method != nil {
		// Function signature
		content = append(content, "### Template Function\n")

		// Function signature in Go style
		sig := fmt.Sprintf("```go\nfunc %s(", method.Name)
		params := make([]string, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = fmt.Sprintf("arg%d %s", i+1, param.String())
		}
		sig += strings.Join(params, ", ")
		sig += ")"

		if len(method.Results) > 0 {
			if len(method.Results) == 1 {
				sig += fmt.Sprintf(" %s", method.Results[0].String())
			} else {
				sig += " ("
				results := make([]string, len(method.Results))
				for i, result := range method.Results {
					results[i] = result.String()
				}
				sig += strings.Join(results, ", ")
				sig += ")"
			}
		}
		sig += "\n```"
		content = append(content, sig)

		// Template usage example
		usage := "### Template Usage\n```go-template\n"
		if len(variable.PipeArguments) > 0 {
			usage += variable.Position.Text + " | " + method.Name
			for _, arg := range variable.PipeArguments {
				if arg.Type != nil {
					usage += fmt.Sprintf(" %q", arg.Type.String())
				}
			}
		} else {
			usage += method.Name + " " + variable.Position.Text
		}
		usage += "\n```"
		content = append(content, usage)

	} else if typeInfo != nil {
		if typeInfo.Type.Func != nil {
			// Method Information
			content = append(content, "### Method Information\n")

			// Method signature
			sig := typeInfo.NestedMultiLineTypeString()
			content = append(content, sig)

			// Return type info
			if sig, ok := typeInfo.Type.Type().(*types.Signature); ok && sig.Results().Len() > 0 {
				content = append(content, "\n### Return Type\n```go")
				for i := 0; i < sig.Results().Len(); i++ {
					content = append(content, sig.Results().At(i).Type().String())
				}
				content = append(content, "```")
			}

			// Template usage
			content = append(content, "\n### Template Usage\n```go-template")
			content = append(content, variable.Position.Text)
			content = append(content, "```")

		} else {
			// Type Information for fields
			content = append(content, "### Type Information\n")

			// Add the nested type visualization
			typeStr := typeInfo.NestedMultiLineTypeString()
			if typeStr == "" {
				typeStr = "```go\n// Type information not available\n```"
			}
			content = append(content, typeStr)

			// Show the template access path
			templatePath := "\n### Template Access\n```go-template\n"
			templatePath += variable.Position.Text
			templatePath += "\n```"
			content = append(content, templatePath)
		}
	} else {
		// Fallback for when we don't have type info
		content = append(content, "### Template Reference\n")
		content = append(content, "```go-template\n"+variable.Position.Text+"\n```")
		content = append(content, "\n*Type information not available*")
	}

	zerolog.Ctx(ctx).Debug().
		Str("variable", variable.Position.Text).
		Bool("has_method", method != nil).
		Bool("has_type_info", typeInfo != nil).
		Bool("is_method", typeInfo != nil && typeInfo.Type.Func != nil).
		Msg("formatted hover response")

	return &HoverInfo{
		Content:  []string{strings.Join(content, "\n")},
		Position: variable.Position,
	}, nil
}

func BuildHoverResponseFromParse(ctx context.Context, info *parser.ParsedTemplateFile, hoverPosition position.RawPosition, registry *ast.Registry) (*HoverInfo, error) {
	for _, block := range info.Blocks {
		if block.TypeHint == nil {
			continue
		}

		zerolog.Ctx(ctx).Debug().Msgf("checking block %s against type hint %s (vars: %d)", block.Name, block.TypeHint.TypePath, len(block.Variables))

		for _, function := range block.Functions {
			zerolog.Ctx(ctx).Debug().Any("function", function).Any("hover", hoverPosition).Msg("checking overlap")
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", hoverPosition.Text, hoverPosition.Offset, function.Position.Text, function.Position.Offset)
			if hoverPosition.HasRangeOverlapWith(function.Position) {
				zerolog.Ctx(ctx).Debug().Msgf("function %s at %v overlaps with position %v", function.Name(), function.Position, hoverPosition)
				method, err := ast.GenerateFunctionCallInfoFromPosition(ctx, function.Position)
				if err != nil {
					return nil, errors.Errorf("generating function call info: %w", err)
				}

				hoverInfo, err := FormatHoverResponse(ctx, &function, method, nil)
				if err != nil {
					return nil, errors.Errorf("formatting hover response: %w", err)
				}

				return hoverInfo, nil
			}
		}

		thd, err := ast.BuildTypeHintDefinitionFromRegistry(ctx, block.TypeHint.TypePath, registry)
		if err != nil {
			return nil, errors.Errorf("building type hint definition: %w", err)
		}

		for _, variable := range block.Variables {
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", hoverPosition.Text, hoverPosition.Offset, variable.Position.Text, variable.Position.Offset)
			if hoverPosition.HasRangeOverlapWith(variable.Position) {
				zerolog.Ctx(ctx).Debug().Msgf("variable %s at %v overlaps with position %v", variable.Name(), variable.Position, hoverPosition)

				typeInfo, err := ast.GenerateFieldInfoFromPosition(ctx, thd, variable.Position)
				if err != nil {
					// If the field doesn't exist, return nil hover info instead of an error
					if strings.Contains(err.Error(), "field not found") {
						return nil, nil
					}
					return nil, errors.Errorf("generating field info: %w", err)
				}

				hoverInfo, err := FormatHoverResponse(ctx, &variable, nil, typeInfo)
				if err != nil {
					return nil, errors.Errorf("formatting hover response: %w", err)
				}

				return hoverInfo, nil
			}
		}

	}

	return nil, nil
}
