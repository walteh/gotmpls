package parser_test

import (
	"context"
	"go/types"
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
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
				Filename: "test.tmpl",
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Line:     1,
						Column:   12,
					},
				},
				Variables: []parser.VariableLocation{
					{
						Name:    "Name",
						Line:    3,
						Column:  9,
						EndLine: 3,
						EndCol:  13,
					},
					{
						Name:    "Age",
						Line:    3,
						Column:  28,
						EndLine: 3,
						EndCol:  31,
					},
				},
				Functions: []parser.VariableLocation{},
				// Definitions: []parser.DefinitionInfo{},
			},
			wantErr: false,
		},
		{
			name: "template with function calls",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`,
			want: func() *parser.TemplateInfo {
				variable := parser.VariableLocation{
					Name:    "Name",
					Line:    3,
					Column:  21,
					EndLine: 3,
					EndCol:  25,
				}
				want := &parser.TemplateInfo{
					Filename: "test.tmpl",
					TypeHints: []parser.TypeHint{
						{
							TypePath: "github.com/example/types.Config",
							Line:     1,
							Column:   12,
						},
					},
					Variables: []parser.VariableLocation{
						variable,
					},
					Functions: []parser.VariableLocation{
						{
							Name:    "printf",
							Line:    3,
							Column:  3,
							EndLine: 3,
							EndCol:  9,
							MethodArguments: []types.Type{
								types.Typ[types.String],
							},
						},
						{
							Name:            "upper",
							Line:            3,
							Column:          28,
							EndLine:         3,
							EndCol:          33,
							MethodArguments: []types.Type{&variable},
						},
					},
				}
				return want
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
				variable := parser.VariableLocation{
					Name:            "GetJob",
					Line:            1,
					Column:          24,
					EndLine:         1,
					EndCol:          30,
					MethodArguments: []types.Type{},
				}
				want := &parser.TemplateInfo{
					Filename: "test.tmpl",
					Variables: []parser.VariableLocation{
						variable,
					},
				}
				want.Functions = []parser.VariableLocation{
					{
						Name:            "upper",
						Line:            1,
						Column:          31,
						EndLine:         1,
						EndCol:          35,
						MethodArguments: []types.Type{&variable},
					},
				}
				return want
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewDefaultTemplateParser()
			got, err := p.Parse(context.Background(), []byte(tt.template), "test.tmpl")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetLineAndColumn(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      parse.Pos
		wantLine int
		wantCol  int
	}{
		{
			name:     "empty text",
			text:     "",
			pos:      parse.Pos(0),
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "single line, first position",
			text:     "Hello, World! ",
			pos:      parse.Pos(2),
			wantLine: 1,
			wantCol:  3,
		},
		{
			name:     "single line, middle position",
			text:     "Hello, World!",
			pos:      parse.Pos(7),
			wantLine: 1,
			wantCol:  8,
		},
		{
			name:     "multiple lines, first line",
			text:     "Hello\nWorld\nTest",
			pos:      parse.Pos(3),
			wantLine: 1,
			wantCol:  4,
		},
		{
			name:     "multiple lines, second line",
			text:     "Hello\nWorld\nTest zzz",
			pos:      parse.Pos(8),
			wantLine: 2,
			wantCol:  3,
		},
		{
			name:     "multiple lines with varying lengths",
			text:     "Hello, World!\nThis is a test\nShort\nLonger line here zzz",
			pos:      parse.Pos(16),
			wantLine: 2,
			wantCol:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotCol := parser.GetLineAndColumn(tt.text, tt.pos)
			if gotLine != tt.wantLine || gotCol != tt.wantCol {
				t.Errorf("GetLineAndColumn() = (%v, %v), want (%v, %v)", gotLine, gotCol, tt.wantLine, tt.wantCol)
			}
		})
	}
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

	p := parser.NewDefaultTemplateParser()
	info, err := p.Parse(context.Background(), []byte(data), "test.tmpl")
	require.NoError(t, err)

	// Check type hint
	require.Equal(t, 1, len(info.TypeHints))
	require.Equal(t, "github.com/walteh/go-tmpl-types-vscode/examples/types.Person", info.TypeHints[0].TypePath)

	// Check variables - should include all parts of nested fields
	expectedVars := []string{
		"Names",
		"Age",
		"Address.Street", // From .Address.Street
		"Address.City",   // From .Address.City
		"HasJob",
		"GetJob",
	}
	foundVars := make([]string, len(info.Variables))
	for i, v := range info.Variables {
		foundVars[i] = v.Name
	}
	require.ElementsMatch(t, expectedVars, foundVars)

	// Check functions
	require.Equal(t, 1, len(info.Functions))
	require.Equal(t, "upper", info.Functions[0].Name)
}
