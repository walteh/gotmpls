package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

func TestVariableProvider_GetCompletions(t *testing.T) {
	provider := NewVariableProvider()

	tests := []struct {
		name         string
		templateInfo *parser.TemplateInfo
		wantNames    []string
	}{
		{
			name: "basic variables",
			templateInfo: &parser.TemplateInfo{
				Variables: []parser.VariableLocation{
					{
						Name:   "user",
						Line:   1,
						Column: 1,
						Scope:  "",
					},
					{
						Name:   "data",
						Line:   2,
						Column: 1,
						Scope:  "",
					},
				},
			},
			wantNames: []string{"user", "data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := provider.GetCompletions(tt.templateInfo)
			require.Len(t, items, len(tt.wantNames))

			// Check that all expected variables are present
			for _, wantName := range tt.wantNames {
				found := false
				for _, item := range items {
					if item.Label == wantName {
						found = true
						assert.Equal(t, "variable", item.Kind)
						break
					}
				}
				assert.True(t, found, "Expected variable '%s' not found", wantName)
			}
		})
	}
}
