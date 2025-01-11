package parser_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func TestTemplateParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     *parser.TemplateInfo
		wantErr  bool
	}{
		{
			name: "basic template with type hint",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
Hello {{.Name}}! You are {{.Age}} years old.
{{end}}`,
			want: &parser.TemplateInfo{
				Filename:  "test.tmpl",
				Functions: []parser.VariableLocation{},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Position: position.NewBasicPosition("github.com/example/types.Config", 0),
					},
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition(".Name", 0),
					},
					{
						Position: position.NewBasicPosition(".Age", 0),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "template with function calls",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`,
			want: &parser.TemplateInfo{
				Filename: "test.tmpl",
				Functions: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition("printf", 0),
						MethodArguments: []types.Type{
							types.Typ[types.String],
						},
					},
					{
						Position: position.NewBasicPosition("upper", 0),
					},
				},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Position: position.NewBasicPosition("github.com/example/types.Config", 0),
					},
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition(".Name", 0),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid template",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{.Name} // Missing closing brace
{{end}}`,
			want:    nil,
			wantErr: true,
		},
		{
			name:     "method call with pipe to upper",
			template: `JobZ: {{printf "%s" .GetJob | upper}}`,
			want: &parser.TemplateInfo{
				Filename: "test.tmpl",
				Functions: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition("printf", 0),
						MethodArguments: []types.Type{
							types.Typ[types.String],
						},
					},
					{
						Position: position.NewBasicPosition("upper", 0),
					},
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition(".GetJob", 0),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "broken example",
			template: `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			want: &parser.TemplateInfo{
				Filename:  "test.tmpl",
				Functions: []parser.VariableLocation{},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "test.Person",
						Position: position.NewBasicPosition("test.Person", 14),
					},
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.NewBasicPosition(".Address.Street", 34),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := parser.Parse(ctx, []byte(tt.template), "test.tmpl")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSample1(t *testing.T) {
	data := `{{- define "header" -}}
# Person Information
{{- end -}}

{{template "headerzzz"}}
{{- /*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Person*/ -}}

Name: {{.Names}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}

{{if .HasJob}}
Job: {{.GetJob | upper}}
{{end}} `

	want := &parser.TemplateInfo{
		Filename: "test.tmpl",
		TypeHints: []parser.TypeHint{
			{
				TypePath:      "github.com/walteh/go-tmpl-types-vscode/examples/types.Person",
				Position:      position.NewBasicPosition("github.com/walteh/go-tmpl-types-vscode/examples/types.Person", 0),
				BlockPosition: position.NewBasicPosition("test.tmpl", 0),
			},
		},
		Variables: []parser.VariableLocation{
			{
				Position: position.NewBasicPosition(".Names", 171),
			},
			{
				Position: position.NewBasicPosition(".Age", 187),
			},
			{
				Position: position.NewBasicPosition(".Address.Street", 223),
			},
			{
				Position: position.NewBasicPosition(".Address.City", 251),
			},
			{
				Position: position.NewBasicPosition(".HasJob", 265),
			},
			{
				Position: position.NewBasicPosition(".GetJob", 282),
			},
		},
		Functions: []parser.VariableLocation{
			{
				Position: position.NewBasicPosition("upper", 292),
				MethodArguments: []types.Type{
					&parser.VariableLocation{
						Position: position.NewBasicPosition(".GetJob", 282),
					},
				},
			},
		},
	}

	ctx := context.Background()

	got, err := parser.Parse(ctx, []byte(data), "test.tmpl")
	require.NoError(t, err)

	assert.Equal(t, want, got)
}
