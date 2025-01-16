package ast

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/position"
)

func TestGenerateFunctionCallInfo(t *testing.T) {
	tests := []struct {
		name    string
		pos     position.RawPosition
		want    *TemplateMethodInfo
		wantErr bool
	}{
		{
			name: "simple function",
			pos: position.RawPosition{
				Text: "upper",
			},
			want: &TemplateMethodInfo{
				Name: "upper",
				Parameters: []types.Type{
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
		},
		{
			name: "function with multiple args",
			pos: position.RawPosition{
				Text: "replace",
			},
			want: &TemplateMethodInfo{
				Name: "replace",
				Parameters: []types.Type{
					types.Typ[types.String],
					types.Typ[types.String],
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
		},
		{
			name: "invalid function",
			pos: position.RawPosition{
				Text: "nonexistent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateFunctionCallInfoFromPosition(context.Background(), tt.pos)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.ElementsMatch(t, tt.want.Parameters, got.Parameters)
			assert.ElementsMatch(t, tt.want.Results, got.Results)
		})
	}
}
