// Package hover provides functionality for generating hover information.
package hover

import (
	"context"
	"fmt"
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
func FormatHoverResponse(ctx context.Context, variable *parser.VariableLocation, method *ast.TemplateMethodInfo, typeInfo *ast.FieldInfo) (*HoverInfo, error) {
	if variable == nil {
		return nil, errors.New("variable cannot be nil")
	}

	var sb strings.Builder

	// If it's a function call with method info
	if method != nil {
		// Function signature
		sb.WriteString(fmt.Sprintf("func %s(", method.Name))

		// Parameters
		params := make([]string, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = param.String()
		}
		sb.WriteString(strings.Join(params, ", "))
		sb.WriteString(")")

		// Return values
		if len(method.Results) > 0 {
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

	} else if typeInfo != nil {
		// Variable section
		// sb.WriteString("**Variable**: ")

		// pp.Println(typeInfo.NestedMultiLineTypeString())

		// fld, ok := typeInfo.Type.Underlying().(*types.Var)
		// if !ok {
		// 	return nil, errors.New("type is not a field")
		// }

		// sb.WriteString(typeInfo.Name)
		// sb.WriteString(".")
		// sb.WriteString(variable.Name)

		// sb.WriteString("\n")

		// sb.WriteString("**Type**: ")
		sb.WriteString(typeInfo.NestedMultiLineTypeString())

	}

	return &HoverInfo{
		Content:  []string{sb.String()},
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
