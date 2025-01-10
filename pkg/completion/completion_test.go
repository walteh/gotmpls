package completion_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/completion"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func TestGetCompletions(t *testing.T) {
	// Create test registry
	registry := ast.NewTypeRegistry()
	pkg := types.NewPackage("test", "test")
	registry.AddPackage(pkg)

	// Add test type
	scope := pkg.Scope()
	scope.Insert(types.NewTypeName(0, pkg, "Name", types.Typ[types.String]))

	doc := position.NewDocument("dummy")

	tests := []struct {
		name          string
		content       string
		line          int
		character     int
		templateInfo  *parser.TemplateInfo
		expectError   bool
		validateItems func(t *testing.T, items []completion.CompletionItem)
	}{
		{
			name:      "field completion",
			content:   "{{ .Name }}",
			line:      1,
			character: 8,
			templateInfo: &parser.TemplateInfo{
				TypeHints: []parser.TypeHint{
					{
						TypePath: "Name",
						Position: doc.NewBasicPosition("Name", 0),
					},
				},
			},
			validateItems: func(t *testing.T, items []completion.CompletionItem) {
				require.Len(t, items, 1, "should have one completion item")
				item := items[0]
				assert.Equal(t, "Name", item.Label, "completion label should match")
				assert.Equal(t, string(completion.CompletionKindField), item.Kind, "should be a field completion")
				assert.Equal(t, "string", item.Detail, "type detail should match")
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
						Position: doc.NewBasicPosition("user", 0),
						Scope:    "",
					},
				},
			},
			validateItems: func(t *testing.T, items []completion.CompletionItem) {
				require.Len(t, items, 1, "should have one completion item")
				item := items[0]
				assert.Equal(t, "user", item.Label, "completion label should match")
				assert.Equal(t, string(completion.CompletionKindVariable), item.Kind, "should be a variable completion")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := completion.GetCompletions(context.Background(), registry, tt.templateInfo, tt.line, tt.character, tt.content)
			if tt.expectError {
				require.Error(t, err, "should return an error")
				return
			}
			require.NoError(t, err, "should not return an error")
			tt.validateItems(t, items)
		})
	}
}
