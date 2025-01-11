package diagnostic_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func TestDiagnosticProvider_GetDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []*diagnostic.Diagnostic
		wantErr  bool
	}{
		{
			name:     "valid template",
			template: "{{/*gotype: github.com/example/types.Person*/}}Hello {{.Name}}!",
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "invalid field",
			template: "{{/*gotype: github.com/example/types.Person*/}}Hello {{.NonExistent}}!",
			want: []*diagnostic.Diagnostic{
				{
					Message:  "field NonExistent not found in type Person",
					Location: position.NewBasicPosition(".NonExistent", 0),
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid type path",
			template: "{{/*gotype: invalid.Type*/}}Hello {{.Name}}!",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

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

			got, err := diagnostic.GetDiagnostics(ctx, tt.template, registry)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.want == nil {
				assert.Empty(t, got)
				return
			}

			require.Equal(t, len(tt.want), len(got), "diagnostics count mismatch")
			for i, want := range tt.want {
				assert.Equal(t, want.Message, got[i].Message, "message mismatch")
				assert.Equal(t, want.Location.Text, got[i].Location.Text, "location mismatch")
			}
		})
	}
}
