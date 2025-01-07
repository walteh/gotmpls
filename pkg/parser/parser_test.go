package parser_test

import (
	"context"
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

func TestTemplateParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filename string
		want     *parser.TemplateInfo
		wantErr  bool
	}{
		{
			name: "basic template with type hint",
			content: `{{- /*gotype: github.com/example/types.Config */ -}}
package templates

{{define "main"}}
Hello {{.Name}}! You are {{.Age}} years old. zzz32
{{end}}`,
			filename: "test.tmpl",
			want: &parser.TemplateInfo{
				Functions: []parser.FunctionLocation{},
				Variables: []parser.VariableLocation{
					{
						Name:    "Name",
						Line:    5,
						Column:  9,
						EndLine: 5,
						EndCol:  13,
					},
					{
						Name:    "Age",
						Line:    5,
						Column:  28,
						EndLine: 5,
						EndCol:  31,
					},
				},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Line:     1,
						Column:   12,
					},
				},
				Filename: "test.tmpl",
			},
			wantErr: false,
		},
		{
			name: "template with function calls",
			content: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`,
			filename: "test.tmpl",
			want: &parser.TemplateInfo{
				Variables: []parser.VariableLocation{
					{
						Name:    "Name",
						Line:    3,
						Column:  21,
						EndLine: 3,
						EndCol:  25,
					},
				},
				Functions: []parser.FunctionLocation{
					{
						Name:    "printf",
						Line:    3,
						Column:  3,
						EndLine: 3,
						EndCol:  9,
					},
					{
						Name:    "upper",
						Line:    3,
						Column:  28,
						EndLine: 3,
						EndCol:  33,
					},
				},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Line:     1,
						Column:   12,
					},
				},
				Filename: "test.tmpl",
			},
			wantErr: false,
		},
		{
			name: "invalid template",
			content: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{.Name} // Missing closing brace
{{end}}`,
			filename: "test.tmpl",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewDefaultTemplateParser()
			got, err := p.Parse(context.Background(), []byte(tt.content), tt.filename)
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
