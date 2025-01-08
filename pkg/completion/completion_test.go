package completion_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/completion"
	"github.com/walteh/go-tmpl-typer/pkg/completion/providers"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	typesp "github.com/walteh/go-tmpl-typer/pkg/types"
)

func TestProvider_GetCompletions(t *testing.T) {
	mockValidator := &mockery.MockValidator_types{}
	mockRegistry := &mockery.MockPackageAnalyzer_ast{}

	// Setup mock validator
	mockValidator.EXPECT().ValidateType(mock.Anything, "example.Person", mock.Anything).Return(&typesp.TypeInfo{
		Name: "Person",
		Fields: map[string]*typesp.FieldInfo{
			"Name": {
				Name: "Name",
				Type: types.Typ[types.String],
			},
			"Age": {
				Name: "Age",
				Type: types.Typ[types.Int],
			},
		},
	}, nil)

	mockValidator.EXPECT().GetRootMethods().Return(map[string]*typesp.MethodInfo{
		"len": {
			Name:       "len",
			Parameters: []types.Type{types.NewInterface(nil, nil)},
			Results:    []types.Type{types.Typ[types.Int]},
		},
	})

	// Setup mock registry
	mockRegistry.EXPECT().GetPackage(mock.Anything, mock.Anything).Return(&types.Package{}, nil)
	mockRegistry.EXPECT().GetTypes().Return(map[string]*types.Package{
		"example": types.NewPackage("example", "example"),
	})

	// Create a new provider with the mocks
	provider := completion.NewProvider(mockValidator, mockRegistry)

	tests := []struct {
		name          string
		content       string
		line          int
		character     int
		templateInfo  *parser.TemplateInfo
		expectError   bool
		validateItems func(t *testing.T, items []providers.CompletionItem)
	}{
		{
			name:      "basic field completion",
			content:   "{{ .Name }}",
			line:      1,
			character: 8,
			templateInfo: &parser.TemplateInfo{
				TypeHints: []parser.TypeHint{
					{
						TypePath: "example.Person",
						Line:     1,
						Column:   1,
						Scope:    "",
					},
				},
			},
			validateItems: func(t *testing.T, items []providers.CompletionItem) {
				require.NotEmpty(t, items)
				found := false
				for _, item := range items {
					if item.Label == "Name" {
						found = true
						assert.Equal(t, "field", item.Kind)
						break
					}
				}
				assert.True(t, found, "Expected field 'Name' not found")
			},
		},
		{
			name:      "variable completion",
			content:   "{{ . }}",
			line:      1,
			character: 4,
			templateInfo: &parser.TemplateInfo{
				Variables: []parser.VariableLocation{
					{
						Name:   "user",
						Line:   1,
						Column: 1,
						Scope:  "",
					},
				},
			},
			validateItems: func(t *testing.T, items []providers.CompletionItem) {
				require.NotEmpty(t, items)
				found := false
				for _, item := range items {
					if item.Label == "user" {
						found = true
						assert.Equal(t, "variable", item.Kind)
						break
					}
				}
				assert.True(t, found, "Expected variable 'user' not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := provider.GetCompletions(context.Background(), tt.templateInfo, tt.line, tt.character, tt.content)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.validateItems(t, items)
		})
	}
}
