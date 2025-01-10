package diagnostic

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

func TestDiagnosticProvider_GetDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		template string
		typePath string
		want     []*Diagnostic
		wantErr  bool
	}{
		{
			name:     "valid template",
			template: "Hello {{.Name}}!",
			typePath: "github.com/example/types.Person",
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "invalid field",
			template: "Hello {{.NonExistent}}!",
			typePath: "github.com/example/types.Person",
			want: []*Diagnostic{
				{
					Message: "field NonExistent not found in type Person",
					Location: parser.RawPosition{
						Text: ".NonExistent",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid type path",
			template: "Hello {{.Name}}!",
			typePath: "invalid.Type",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock registry
			registry := ast.NewRegistry()
			pkg := types.NewPackage("github.com/example/types", "types")
			registry.AddPackage(pkg)

			// Create a mock type
			fields := []*types.Var{
				types.NewField(0, pkg, "Name", types.Typ[types.String], false),
				types.NewField(0, pkg, "Age", types.Typ[types.Int], false),
			}
			structType := types.NewStruct(fields, nil)
			named := types.NewNamed(
				types.NewTypeName(0, pkg, "Person", nil),
				structType,
				nil,
			)
			scope := pkg.Scope()
			scope.Insert(named.Obj())

			provider := NewDiagnosticProvider(registry)
			got, err := provider.GetDiagnostics(context.Background(), tt.template, tt.typePath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.want == nil {
				assert.Empty(t, got)
				return
			}

			require.Equal(t, len(tt.want), len(got))
			for i, want := range tt.want {
				assert.Equal(t, want.Message, got[i].Message)
				assert.Equal(t, want.Location.Text, got[i].Location.Text)
			}
		})
	}
}
