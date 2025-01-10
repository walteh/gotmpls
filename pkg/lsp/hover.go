package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// HandleSimpleVariableHover handles hover for simple variables like .Name
func HandleSimpleVariableHover(ctx context.Context, validator pkg_types.Validator, v parser.VariableLocation, typeInfo *pkg_types.TypeInfo, typeName string) (*Hover, error) {
	field, err := validator.ValidateField(ctx, typeInfo, v.Name)
	if err != nil || field == nil {
		debugf(ctx, "failed to validate field %s: %v", v.Name, err)
		return nil, err
	}

	// Convert parser's 1-based positions to LSP's 0-based positions
	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: fmt.Sprintf("**Variable**: %s.%s\n**Type**: %s", typeName, v.Name, field.Type.String()),
		},
		Range: &Range{
			Start: Position{Line: v.Line - 1, Character: v.Column - 1},
			End:   Position{Line: v.Line - 1, Character: v.EndCol - 1},
		},
	}, nil
}

// HandleFieldAccessHover handles hover for field access like .Address.Street
func HandleFieldAccessHover(ctx context.Context, validator pkg_types.Validator, v parser.VariableLocation, typeInfo *pkg_types.TypeInfo, typeName string, position Position) (*Hover, error) {
	debugf(ctx, "handling field access hover for %s at position line:%d char:%d", v.Name, position.Line, position.Character)

	// Convert LSP's 0-based line to parser's 1-based line for comparison
	if position.Line+1 != v.Line {
		return nil, nil
	}

	field, err := validator.ValidateField(ctx, typeInfo, v.Name)
	if err != nil {
		debugf(ctx, "failed to validate field %s: %v", v.Name, err)
		return nil, err
	}

	// Convert parser's 1-based positions to LSP's 0-based positions
	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: fmt.Sprintf("**Variable**: %s.%s\n**Type**: %s", typeName, v.Name, field.Type.String()),
		},
		Range: &Range{
			Start: Position{Line: v.Line - 1, Character: v.Column - 1},
			End:   Position{Line: v.Line - 1, Character: v.EndCol - 1},
		},
	}, nil
}

func (s *Server) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if s.debug {
		s.debugf(ctx, "handling textDocument/hover")
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.debugf(ctx, "failed to unmarshal hover params: %v", err)
		return nil, errors.Errorf("failed to unmarshal hover params: %w", err)
	}

	text, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		s.debugf(ctx, "document not found: %s", params.TextDocument.URI)
		return nil, errors.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Let the parser handle all template parsing
	tmpl, err := s.parser.Parse(ctx, []byte(text), params.TextDocument.URI)
	if err != nil {
		s.debugf(ctx, "failed to parse template: %v", err)
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	if len(tmpl.TypeHints) == 0 {
		s.debugf(ctx, "no type hints found in template")
		return nil, errors.Errorf("no type hints found in template")
	}

	typeHint := tmpl.TypeHints[0]
	typeName := strings.Split(typeHint.TypePath, ".")[len(strings.Split(typeHint.TypePath, "."))-1]

	// Let the validator handle all type validation
	typeInfo, err := s.validator.ValidateType(ctx, typeHint.TypePath, s.analyzer)
	if err != nil {
		s.debugf(ctx, "failed to validate type: %v", err)
		return nil, errors.Errorf("failed to validate type: %w", err)
	}

	// Find the variable or function at the hover position
	for _, v := range tmpl.Variables {
		// Convert LSP's 0-based line to parser's 1-based line for comparison
		if v.Line != params.Position.Line+1 {
			continue
		}

		// Let the parser determine if this is a field access
		if strings.Contains(v.Name, ".") {
			return HandleFieldAccessHover(ctx, s.validator, v, typeInfo, typeName, params.Position)
		}

		// Convert LSP's 0-based column to parser's 1-based column for comparison
		if v.Column <= params.Position.Character+1 && v.EndCol >= params.Position.Character+1 {
			return HandleSimpleVariableHover(ctx, s.validator, v, typeInfo, typeName)
		}
	}

	// Handle function hovers similarly
	for _, f := range tmpl.Functions {
		// Convert LSP's 0-based positions to parser's 1-based positions for comparison
		if f.Line == params.Position.Line+1 &&
			f.Column <= params.Position.Character+1 &&
			f.EndLine == params.Position.Line+1 &&
			f.EndCol >= params.Position.Character+1 {

			var args []string
			for _, arg := range f.MethodArguments {
				args = append(args, arg.String())
			}
			signature := fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ", "))

			// Convert parser's 1-based positions to LSP's 0-based positions for response
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("**Function**: %s\n**Signature**: %s\n**Scope**: %s", f.Name, signature, f.Scope),
				},
				Range: &Range{
					Start: Position{Line: f.Line - 1, Character: f.Column - 1},
					End:   Position{Line: f.EndLine - 1, Character: f.EndCol - 1},
				},
			}, nil
		}
	}

	return nil, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func createHoverResponse(s *Server, ctx context.Context, v parser.VariableLocation) (interface{}, error) {
	var typeInfo string
	if v.MethodArguments != nil && len(v.MethodArguments) > 0 {
		typeInfo = v.MethodArguments[0].String()
		s.debugf(ctx, "variable type info from MethodArguments: %s", typeInfo)
	} else {
		typeInfo = "unknown"
		s.debugf(ctx, "no type info available for variable")
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: fmt.Sprintf("**Variable**: %s\n**Type**: %s\n**Scope**: %s", v.Name, typeInfo, v.Scope),
		},
		Range: &Range{
			Start: Position{Line: v.Line - 1, Character: v.Column - 1},
			End:   Position{Line: v.Line - 1, Character: v.EndCol - 1},
		},
	}, nil
}
