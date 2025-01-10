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
			want: func() *parser.TemplateInfo {
				nameVar := parser.VariableLocation{
					Name:    "Name",
					Line:    3,
					Column:  9,
					EndLine: 3,
					EndCol:  13,
					Scope:   "main",
				}
				ageVar := parser.VariableLocation{
					Name:    "Age",
					Line:    3,
					Column:  28,
					EndLine: 3,
					EndCol:  31,
					Scope:   "main",
				}
				return &parser.TemplateInfo{
					Filename: "test.tmpl",
					TypeHints: []parser.TypeHint{
						{
							TypePath: "github.com/example/types.Config",
							Line:     1,
							Column:   12,
							Scope:    "",
						},
					},
					Variables: []parser.VariableLocation{
						nameVar,
						ageVar,
					},
					Functions: []parser.VariableLocation{},
				}
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
				variable := parser.VariableLocation{
					Name:    "Name",
					Line:    3,
					Column:  21,
					EndLine: 3,
					EndCol:  25,
					Scope:   "main",
				}
				printfFunc := parser.VariableLocation{
					Name:    "printf",
					Line:    3,
					Column:  3,
					EndLine: 3,
					EndCol:  9,
					Scope:   "main",
					MethodArguments: []types.Type{
						types.Typ[types.String],
						&variable,
					},
				}
				want := &parser.TemplateInfo{
					Filename: "test.tmpl",
					TypeHints: []parser.TypeHint{
						{
							TypePath: "github.com/example/types.Config",
							Line:     1,
							Column:   12,
							Scope:    "",
						},
					},
					Variables: []parser.VariableLocation{
						variable,
					},
					Functions: []parser.VariableLocation{
						printfFunc,
						{
							Name:    "upper",
							Line:    3,
							Column:  28,
							EndLine: 3,
							EndCol:  33,
							Scope:   "main",
							MethodArguments: []types.Type{
								&printfFunc,
							},
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
					Name:    "GetJob",
					Line:    1,
					Column:  21,
					EndLine: 1,
					EndCol:  27,
					Scope:   "",
				}
				printfFunc := parser.VariableLocation{
					Name:    "printf",
					Line:    1,
					Column:  9,
					EndLine: 1,
					EndCol:  15,
					Scope:   "",
					MethodArguments: []types.Type{
						types.Typ[types.String],
						&variable,
					},
				}
				want := &parser.TemplateInfo{
					Filename: "test.tmpl",
					Variables: []parser.VariableLocation{
						variable,
					},
					Functions: []parser.VariableLocation{
						printfFunc,
						{
							Name:    "upper",
							Line:    1,
							Column:  30,
							EndLine: 1,
							EndCol:  35,
							Scope:   "",
							MethodArguments: []types.Type{
								&printfFunc,
							},
						},
					},
				}
				return want
			}(),
			wantErr: false,
		},
		{
			name: "broken example",
			template: `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			want: func() *parser.TemplateInfo {
				streetVar := parser.VariableLocation{
					Name:     "Street",
					LongName: ".Address.Street",
					Line:     3,
					Column:   21,
					EndLine:  3,
					EndCol:   26,
					Scope:    "",
				}
				return &parser.TemplateInfo{
					Filename: "test.tmpl",
					TypeHints: []parser.TypeHint{
						{
							TypePath: "test.Person",
							Line:     1,
							Column:   12,
							Scope:    "",
						},
					},
					Variables: []parser.VariableLocation{streetVar},
					Functions: []parser.VariableLocation{},
				}
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
		{
			name:     "broken example",
			text:     "{{- /*gotype: test.Person*/ -}}\nAddress:\n  Street: {{.Address.Street}}",
			pos:      parse.Pos(61),
			wantLine: 3,
			wantCol:  13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "broken example" {
				t.Logf("running %s", tt.name)
			}
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
	require.Equal(t, "", info.TypeHints[0].Scope) // Root scope

	// Check variables - should include all parts of nested fields
	expectedVars := map[string]string{
		"Names":          "",
		"Age":            "",
		"Address.Street": "",
		"Address.City":   "",
		"HasJob":         "",
		"GetJob":         "",
	}

	foundVars := make(map[string]string)
	for _, v := range info.Variables {
		foundVars[v.Name] = v.Scope
		require.Equal(t, expectedVars[v.Name], v.Scope, "Variable %s has unexpected scope", v.Name)
	}
	require.Equal(t, len(expectedVars), len(foundVars), "Number of variables does not match")

	// Check functions
	require.Equal(t, 1, len(info.Functions))
	require.Equal(t, "upper", info.Functions[0].Name)
	require.Equal(t, "", info.Functions[0].Scope) // Root scope
	require.Equal(t, 1, len(info.Functions[0].MethodArguments))
	varArg, ok := info.Functions[0].MethodArguments[0].(*parser.VariableLocation)
	require.True(t, ok)
	require.Equal(t, "GetJob", varArg.Name)
	require.Equal(t, "", varArg.Scope) // Root scope
}
