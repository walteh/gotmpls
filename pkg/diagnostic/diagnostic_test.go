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
			want: []*diagnostic.Diagnostic{
				{
					Message:  "type hint successfully loaded: github.com/example/types.Person",
					Location: position.NewBasicPosition("github.com/example/types.Person", 11),
					Severity: diagnostic.SeverityInformation,
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid field",
			template: "{{/*gotype: github.com/example/types.Person*/}}Hello {{.NonExistent}}!",
			want: []*diagnostic.Diagnostic{
				{
					Message:  "field NonExistent not found in type Person",
					Location: position.NewBasicPosition(".NonExistent", 54),
					Severity: diagnostic.SeverityError,
				},
				{
					Message:  "type hint successfully loaded: github.com/example/types.Person",
					Location: position.NewBasicPosition("github.com/example/types.Person", 11),
					Severity: diagnostic.SeverityInformation,
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid type path",
			template: "{{/*gotype: invalid.Type*/}}Hello {{.Name}}!",
			want:     []*diagnostic.Diagnostic{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			// Create a mock registry
			registry := ast.NewEmptyRegistry()

			pkgd := registry.AddInMemoryPackageForTesting(ctx, "github.com/example/types")

			pkgd.AddStruct("Person", map[string]types.Type{
				"Name": types.Typ[types.String],
				"Age":  types.Typ[types.Int],
			})

			pkgd.MustAddAndParseTemplates(ctx, map[string]string{
				"test.tmpl": tt.template,
			})

			got, err := diagnostic.GetDiagnostics(ctx, tt.template, registry)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, len(tt.want), len(got), "diagnostics count mismatch")
			assert.ElementsMatch(t, tt.want, got, "diagnostics mismatch")

		})
	}
}
