package lsp

import (
	"context"
	"encoding/json"
	"testing"

	"go/types"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
)

func TestLSPHoverProtocol(t *testing.T) {
	mockValidator := mockery.NewMockValidator_types(t)
	mockParser := mockery.NewMockTemplateParser_parser(t)
	ctx := context.Background()

	t.Run("converts parser positions to LSP for simple variable", func(t *testing.T) {
		// Given a variable location from the parser (1-based positions)
		varLoc := parser.VariableLocation{
			Name:   "Name",
			Line:   2, // 1-based
			Column: 3, // 1-based
			EndCol: 7, // 1-based
		}

		// And a type from the validator
		mockValidator.EXPECT().ValidateField(mock.Anything, mock.Anything, "Name").Return(&pkg_types.FieldInfo{
			Type: types.Typ[types.String],
		}, nil).Once()

		// When we handle the hover
		hover, err := HandleSimpleVariableHover(ctx, mockValidator, varLoc, &pkg_types.TypeInfo{}, "Person")

		// Then we get LSP hover response with 0-based positions
		require.NoError(t, err)
		require.NotNil(t, hover)
		assert.Equal(t, "markdown", hover.Contents.Kind)
		assert.Contains(t, hover.Contents.Value, "**Variable**: Person.Name")
		assert.Contains(t, hover.Contents.Value, "**Type**: string")
		assert.Equal(t, Position{Line: 1, Character: 2}, hover.Range.Start) // 0-based
		assert.Equal(t, Position{Line: 1, Character: 6}, hover.Range.End)   // 0-based
	})

	t.Run("converts parser positions to LSP for nested field", func(t *testing.T) {
		// Given a variable location from the parser (1-based positions)
		varLoc := parser.VariableLocation{
			Name:   "Address.Street",
			Line:   2,  // 1-based
			Column: 3,  // 1-based
			EndCol: 16, // 1-based
		}

		// And type information from the validator
		mockValidator.EXPECT().ValidateField(mock.Anything, mock.Anything, "Address.Street").Return(&pkg_types.FieldInfo{
			Type: types.Typ[types.String],
		}, nil).Once()

		// When we handle the hover with LSP position (0-based)
		hover, err := HandleFieldAccessHover(ctx, mockValidator, varLoc, &pkg_types.TypeInfo{}, "Person", Position{Line: 1, Character: 5})

		// Then we get LSP hover response with 0-based positions
		require.NoError(t, err)
		require.NotNil(t, hover)
		assert.Equal(t, "markdown", hover.Contents.Kind)
		assert.Contains(t, hover.Contents.Value, "**Variable**: Person.Address.Street")
		assert.Contains(t, hover.Contents.Value, "**Type**: string")
		assert.Equal(t, Position{Line: 1, Character: 2}, hover.Range.Start) // 0-based
		assert.Equal(t, Position{Line: 1, Character: 15}, hover.Range.End)  // 0-based
	})

	t.Run("server delegates to parser and validator", func(t *testing.T) {
		// Given a server with mocked dependencies
		server := NewServer(mockParser, mockValidator, nil, nil, true)

		// And a document in the server's store
		uri := "test.tmpl"
		server.documents.Store(uri, "{{.Name}}")

		// And the parser returns a variable location (1-based positions)
		tmpl := &parser.TemplateInfo{
			TypeHints: []parser.TypeHint{{TypePath: "example.com/test/types.Person"}},
			Variables: []parser.VariableLocation{{
				Name:   "Name",
				Line:   1, // 1-based
				Column: 3, // 1-based
				EndCol: 7, // 1-based
			}},
		}

		// And the parser and validator are set up with expectations
		mockParser.EXPECT().Parse(mock.Anything, mock.Anything, uri).Return(tmpl, nil).Once()
		mockValidator.EXPECT().ValidateType(mock.Anything, "example.com/test/types.Person", mock.Anything).Return(&pkg_types.TypeInfo{}, nil).Once()
		mockValidator.EXPECT().ValidateField(mock.Anything, mock.Anything, "Name").Return(&pkg_types.FieldInfo{
			Type: types.Typ[types.String],
		}, nil).Once()

		// When we send an LSP hover request (0-based positions)
		params := HoverParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: 0, Character: 4}, // 0-based
		}
		paramsBytes, _ := json.Marshal(params)
		rawParams := json.RawMessage(paramsBytes)
		req := &jsonrpc2.Request{
			Method: "textDocument/hover",
			Params: &rawParams,
		}

		// Then the hover response uses LSP positions (0-based)
		hover, err := server.handleTextDocumentHover(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, hover)

		hoverResp, ok := hover.(*Hover)
		require.True(t, ok)
		assert.Equal(t, "markdown", hoverResp.Contents.Kind)
		assert.Contains(t, hoverResp.Contents.Value, "**Variable**: Person.Name")
		assert.Contains(t, hoverResp.Contents.Value, "**Type**: string")
		assert.Equal(t, Position{Line: 0, Character: 2}, hoverResp.Range.Start) // 0-based
		assert.Equal(t, Position{Line: 0, Character: 6}, hoverResp.Range.End)   // 0-based
	})

	mockValidator.AssertExpectations(t)
	mockParser.AssertExpectations(t)
}
