// Package hover provides functionality for generating hover information.
package hover

import (
	"context"
	"fmt"
	"go/types"
	"strings"

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
func FormatHoverResponse(ctx context.Context, variable *parser.VariableLocation, method *ast.TemplateMethodInfo, typeResolver func(parser.VariableLocationOrType) []types.Type) (*HoverInfo, error) {
	if variable == nil {
		return nil, errors.New("variable cannot be nil")
	}

	var sb strings.Builder

	// If it's a function call with method info
	if method != nil {
		// Template Function section
		sb.WriteString("### Template Function\n\n")

		// Visual representation of the chain
		if len(variable.PipeArguments) > 0 {

			piped := variable.GetPipedArguments(typeResolver)
			// Show input chain
			for i, arg := range piped.Arguments {
				if i > 0 {
					sb.WriteString("    │\n")
				}
				if arg.PipedArgument != nil {
					sb.WriteString(arg.PipedArgument.Variable.Position.Text + "\n")
				} else if arg.Type != nil {
					sb.WriteString(arg.Type.String() + "\n")
				}
			}
			sb.WriteString("    │\n    ▼\n")
		}
		sb.WriteString(method.Name + "\n\n")

		// Function signature section
		sb.WriteString("### Signature\n\n")
		sb.WriteString("```go\n")
		sb.WriteString(fmt.Sprintf("func %s(", method.Name))

		// Parameters
		params := make([]string, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = fmt.Sprintf("arg%d %s", i+1, param.String())
		}
		sb.WriteString(strings.Join(params, ", "))
		sb.WriteString(")")

		// Return values
		if len(method.Results) > 0 {
			if len(variable.PipeArguments) > 0 {
				sb.WriteString(" -> ")
				piped := variable.GetPipedArguments(typeResolver)

				for _, arg := range piped.Results {
					sb.WriteString(arg.String() + " ")
				}
			} else {
				if len(method.Results) == 1 {
					sb.WriteString(fmt.Sprintf(" %s", method.Results[0].String()))
				} else {
					sb.WriteString(" (")
					results := make([]string, len(method.Results))
					for i, result := range method.Results {
						results[i] = result.String()
					}
					sb.WriteString(strings.Join(results, ", "))
					sb.WriteString(")")
				}
			}
		}
		sb.WriteString("\n```\n\n")

		// Example usage section
		sb.WriteString("### Template Usage\n\n")
		sb.WriteString("```\n")

		// Build the example usage string
		usage := variable.Position.Text
		if len(variable.PipeArguments) > 0 {
			usage += " | " + method.Name
		}
		sb.WriteString(usage + "\n```")
	} else {
		// Variable section
		sb.WriteString("### Variable\n\n")
		sb.WriteString(variable.Position.Text + "\n")
	}

	return &HoverInfo{
		Content:  []string{sb.String()},
		Position: variable.Position,
	}, nil
}

func BuildHoverResponseFromParse(ctx context.Context, info *parser.ParsedTemplateFile, hoverPosition position.RawPosition, typeResolver func(parser.VariableLocationOrType) []types.Type) (*HoverInfo, error) {
	for _, block := range info.Blocks {
		if block.TypeHint == nil {
			continue
		}

		zerolog.Ctx(ctx).Debug().Msgf("checking block %s against type hint %s (vars: %d)", block.Name, block.TypeHint.TypePath, len(block.Variables))

		for _, function := range block.Functions {
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", hoverPosition.Text, hoverPosition.Offset, function.Position.Text, function.Position.Offset)
			if hoverPosition.HasRangeOverlapWith(function.Position) {
				zerolog.Ctx(ctx).Debug().Msgf("function %s at %v overlaps with position %v", function.Name(), function.Position, hoverPosition)
				method, err := ast.GenerateFunctionCallInfoFromPosition(ctx, function.Position)
				if err != nil {
					return nil, errors.Errorf("generating function call info: %w", err)
				}

				hoverInfo, err := FormatHoverResponse(ctx, &function, method, typeResolver)
				if err != nil {
					return nil, errors.Errorf("formatting hover response: %w", err)
				}

				return hoverInfo, nil
			}
		}

		for _, variable := range block.Variables {
			zerolog.Ctx(ctx).Debug().Msgf("checking overlap of [%s:%d] with [%s:%d]", hoverPosition.Text, hoverPosition.Offset, variable.Position.Text, variable.Position.Offset)
			if hoverPosition.HasRangeOverlapWith(variable.Position) {
				zerolog.Ctx(ctx).Debug().Msgf("variable %s at %v overlaps with position %v", variable.Name(), variable.Position, hoverPosition)

				hoverInfo, err := FormatHoverResponse(ctx, &variable, nil, typeResolver)
				if err != nil {
					return nil, errors.Errorf("formatting hover response: %w", err)
				}

				return hoverInfo, nil
			}
		}

	}

	return nil, nil
}
