package providers

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	typesp "github.com/walteh/go-tmpl-typer/pkg/types"
)

func TestFieldProvider_GetCompletions(t *testing.T) {
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

	// Create provider
	provider := NewFieldProvider(mockValidator, mockRegistry)

	// Test cases
	tests := []struct {
		name      string
		typePath  string
		wantNames []string
	}{
		{
			name:      "person fields",
			typePath:  "example.Person",
			wantNames: []string{"Name", "Age"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := provider.GetCompletions(context.Background(), tt.typePath)
			require.NoError(t, err)
			require.Len(t, items, len(tt.wantNames))

			// Check that all expected fields are present
			for _, wantName := range tt.wantNames {
				found := false
				for _, item := range items {
					if item.Label == wantName {
						found = true
						assert.Equal(t, "field", item.Kind)
						break
					}
				}
				assert.True(t, found, "Expected field '%s' not found", wantName)
			}
		})
	}
}
