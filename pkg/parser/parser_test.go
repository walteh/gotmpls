package parser_test

import (
	"context"
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
			want: func() *parser.TemplateInfo {
				p := parser.NewDefaultTemplateParser()
				got, err := p.Parse(context.Background(), []byte(`{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
Hello {{.Name}}! You are {{.Age}} years old.
{{end}}`), "test.tmpl")
				require.NoError(t, err)
				return got
			}(),
			wantErr: false,
		},
		{
			name: "template with function calls",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`,
			want: func() *parser.TemplateInfo {
				p := parser.NewDefaultTemplateParser()
				got, err := p.Parse(context.Background(), []byte(`{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`), "test.tmpl")
				require.NoError(t, err)
				return got
			}(),
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
			want: func() *parser.TemplateInfo {
				p := parser.NewDefaultTemplateParser()
				got, err := p.Parse(context.Background(), []byte(`JobZ: {{printf "%s" .GetJob | upper}}`), "test.tmpl")
				require.NoError(t, err)
				return got
			}(),
			wantErr: false,
		},
		{
			name: "broken example",
			template: `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			want: func() *parser.TemplateInfo {
				p := parser.NewDefaultTemplateParser()
				got, err := p.Parse(context.Background(), []byte(`{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`), "test.tmpl")
				require.NoError(t, err)
				return got
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "broken example" {
				t.Logf("running %s", tt.name)
			}
			p := parser.NewDefaultTemplateParser()
			got, err := p.Parse(ctx, []byte(tt.template), "test.tmpl")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestSample1(t *testing.T) {
	data := `{{- /*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Person */ -}}
{{- define "header" -}}
# Person Information
{{- end -}}

{{template "header"}}

Name: {{.Names}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}

{{if .HasJob}}
Job: {{.GetJob | upper}}
{{end}} `

	doc := position.NewDocument(data)

	expectedPositions := []position.RawPosition{
		doc.NewBasicPosition(".Names", 171),
		doc.NewBasicPosition(".Age", 187),
		doc.NewBasicPosition(".Address.Street", 223),
		doc.NewBasicPosition(".Address.City", 251),
		doc.NewBasicPosition(".HasJob", 265),
		doc.NewBasicPosition(".GetJob", 282),
		doc.NewBasicPosition("upper", 292),
	}

	p := parser.NewDefaultTemplateParser()
	info, err := p.Parse(context.Background(), []byte(data), "test.tmpl")
	require.NoError(t, err)

	// Check type hint
	require.Equal(t, 1, len(info.TypeHints))
	require.Equal(t, "github.com/walteh/go-tmpl-types-vscode/examples/types.Person", info.TypeHints[0].TypePath)
	require.Equal(t, "", info.TypeHints[0].Scope) // Root scope

	// Check variables
	seenVars := position.NewPositionsSeenMap()
	for _, v := range info.Variables {
		seenVars.Add(v.Position)
	}

	for _, f := range info.Functions {
		seenVars.Add(f.Position)
	}

	t.Logf("seenVars: %v", seenVars.PositionsWithText(""))

	for _, pos := range expectedPositions {
		assert.True(t, seenVars.Has(pos), "variable %s should be present - positions with same text: %+v", pos.Text(), seenVars.PositionsWithText(pos.Text()).ToStrings())
	}
}
