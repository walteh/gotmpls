// Package hover provides functionality for generating hover information.
package hover

import (
	"context"
	"fmt"
	"go/types"
	"strings"

	"github.com/k0kubun/pp/v3"
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

		// Always show the input variable first
		sb.WriteString(variable.Position.Text + "\n")

		// Get piped arguments and print for debugging
		piped := variable.GetPipedArguments(typeResolver)
		pp.Printf("Piped arguments: %+v\n", piped)

		// Visual representation of the chain
		if len(variable.PipeArguments) > 0 {
			sb.WriteString("    │\n")
			// Add intermediate functions in the chain
			for _, arg := range variable.PipeArguments {
				if arg.Variable != nil {
					sb.WriteString(arg.Variable.Position.Text)
					sb.WriteString("\n    │\n")
				}
			}
		} else {
			sb.WriteString("    │\n")
		}
		sb.WriteString("    ▼\n")
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
		sb.WriteString("\n```\n\n")

		// Example usage section
		sb.WriteString("### Template Usage\n\n")
		sb.WriteString("```\n")

		// Build the example usage string
		usage := variable.Position.Text + " | " + method.Name
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
