package parser_test

import (
	"context"
	"go/types"
	"strings"
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
		want     *parser.FileInfo
		wantErr  bool
	}{
		{
			name: "basic template with type hint",
			template: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
Hello {{.Name}}! You are {{.Age}} years old.
{{end}}`,
			want: &parser.FileInfo{
				Filename: "test.tmpl",
				SourceContent: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
Hello {{.Name}}! You are {{.Age}} years old.
{{end}}`,
				Blocks: []parser.BlockInfo{
					{
						Name:          "test.tmpl",
						StartPosition: position.RawPosition{Text: "<<SOF>>", Offset: 0},
						TypeHint: &parser.TypeHint{
							TypePath: "github.com/example/types.Config",
							Position: position.RawPosition{
								Text:   "github.com/example/types.Config",
								Offset: 14,
							},
							StartPosition: position.RawPosition{
								Text:   "{{- /*gotype: github.com/example/types.Config */ -}}",
								Offset: 0,
							},
							EndPosition: position.RawPosition{
								Text:   "{{- /*gotype: github.com/example/types.Config */ -}}",
								Offset: 0,
							},
							Scope: "",
						},
						Variables:   []parser.VariableLocation{},
						Functions:   []parser.VariableLocation{},
						EndPosition: position.RawPosition{Text: "<<EOF>>", Offset: 107},
					},
					{
						Name:          "main",
						StartPosition: position.RawPosition{Text: "{{define \"main\"}}", Offset: 52},
						TypeHint:      nil,
						Variables: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: ".Name", Offset: 65},
								Scope:    "main",
							},
							{
								Position: position.RawPosition{Text: ".Age", Offset: 85},
								Scope:    "main",
							},
						},
						Functions:   []parser.VariableLocation{},
						EndPosition: position.RawPosition{Text: "}}", Offset: 104},
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
			want: &parser.FileInfo{
				Filename: "test.tmpl",
				SourceContent: `{{- /*gotype: github.com/example/types.Config */ -}}
{{define "main"}}
{{printf "Hello %s" .Name | upper}}
{{end}}`,
				Blocks: []parser.BlockInfo{
					{
						Name:          "test.tmpl",
						StartPosition: position.RawPosition{Text: "<<SOF>>", Offset: 0},
						TypeHint: &parser.TypeHint{
							TypePath: "github.com/example/types.Config",
							Position: position.RawPosition{
								Text:   "github.com/example/types.Config",
								Offset: 14,
							},
							StartPosition: position.RawPosition{
								Text:   "{{- /*gotype: github.com/example/types.Config */ -}}",
								Offset: 0,
							},
							EndPosition: position.RawPosition{
								Text:   "{{- /*gotype: github.com/example/types.Config */ -}}",
								Offset: 0,
							},
							Scope: "",
						},
						Variables:   []parser.VariableLocation{},
						Functions:   []parser.VariableLocation{},
						EndPosition: position.RawPosition{Text: "<<EOF>>", Offset: 108},
					},
					{
						Name:          "main",
						StartPosition: position.RawPosition{Text: "{{define \"main\"}}", Offset: 52},
						TypeHint:      nil,
						Variables: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: ".Name", Offset: 85},
								Scope:    "main",
							},
						},
						Functions: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: "printf", Offset: 63},
								MethodArguments: []types.Type{
									types.Typ[types.String],
								},
								Scope: "main",
							},
							{
								Position: position.RawPosition{Text: "upper", Offset: 92},
								MethodArguments: []types.Type{
									&parser.VariableLocation{
										Position: position.RawPosition{Text: ".Name", Offset: 85},
										Scope:    "main",
									},
								},
								Scope: "main",
							},
						},
						EndPosition: position.RawPosition{Text: "}}", Offset: 105},
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
			want: &parser.FileInfo{
				Filename:      "test.tmpl",
				SourceContent: `JobZ: {{printf "%s" .GetJob | upper}}`,
				Blocks: []parser.BlockInfo{
					{
						Name:          "test.tmpl",
						StartPosition: position.RawPosition{Text: "<<SOF>>", Offset: 0},
						TypeHint:      nil,
						Variables: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: ".GetJob", Offset: 20},
								Scope:    "",
							},
						},
						Functions: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: "printf", Offset: 8},
								MethodArguments: []types.Type{
									types.Typ[types.String],
								},
								Scope: "",
							},
							{
								Position: position.RawPosition{Text: "upper", Offset: 29},
								MethodArguments: []types.Type{
									&parser.VariableLocation{
										Position: position.RawPosition{Text: ".GetJob", Offset: 20},
										Scope:    "",
									},
								},
								Scope: "",
							},
						},
						EndPosition: position.RawPosition{Text: "<<EOF>>", Offset: 35},
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
			want: &parser.FileInfo{
				Filename: "test.tmpl",
				SourceContent: `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
				Blocks: []parser.BlockInfo{
					{
						Name:          "test.tmpl",
						StartPosition: position.RawPosition{Text: "<<SOF>>", Offset: 0},
						TypeHint: &parser.TypeHint{
							TypePath: "test.Person",
							Position: position.RawPosition{
								Text:   "test.Person",
								Offset: 14,
							},
							StartPosition: position.RawPosition{
								Text:   "{{- /*gotype: test.Person*/ -}}",
								Offset: 0,
							},
							EndPosition: position.RawPosition{
								Text:   "{{- /*gotype: test.Person*/ -}}",
								Offset: 0,
							},
							Scope: "",
						},
						Variables: []parser.VariableLocation{
							{
								Position: position.RawPosition{Text: ".Address.Street", Offset: 61},
								Scope:    "",
							},
						},
						Functions:   []parser.VariableLocation{},
						EndPosition: position.RawPosition{Text: "<<EOF>>", Offset: 77},
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
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestSample1(t *testing.T) {
	data := `{{- define "header" -}}
# Person Information
{{- end -}}

{{define "person"}}
{{- /*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Person*/ -}}

Name: {{.Names}}
Age: {{.Age}}
Address:
  Street: {{.Address.Street}}
  City: {{.Address.City}}

{{if .HasJob}}
Job: {{.GetJob | upper}}
{{end}} 
{{end}}

{{define "animal"}}
{{- /*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Animal*/ -}}

Name: {{.Name}}
{{end}}
`

	want := &parser.FileInfo{
		Filename:      "test.tmpl",
		SourceContent: data,
		Blocks: []parser.BlockInfo{
			{
				Name:          "header",
				StartPosition: position.RawPosition{Text: "{{- define \"header\" -}}", Offset: 0},
				TypeHint:      nil,
				Variables:     []parser.VariableLocation{},
				Functions:     []parser.VariableLocation{},
				EndPosition:   position.RawPosition{Text: "}}", Offset: 54},
			},
			{
				Name:          "person",
				StartPosition: position.RawPosition{Text: "{{define \"person\"}}", Offset: 58},
				TypeHint: &parser.TypeHint{
					TypePath: "github.com/walteh/go-tmpl-types-vscode/examples/types.Person",
					Position: position.RawPosition{
						Text:   "github.com/walteh/go-tmpl-types-vscode/examples/types.Person",
						Offset: 82,
					},
					StartPosition: position.RawPosition{
						Text:   "{{/*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Person*/}}",
						Offset: 82,
					},
					EndPosition: position.RawPosition{
						Text:   "{{/*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Person*/}}Name: {{.Names}}\nAge: {{.Age}}\nAddress:\n  Street: {{.Address.Street}}\n  City: {{.Address.City}}\n\n{{if .HasJob}}\nJob: {{.GetJob | upper}}\n{{end}} \n",
						Offset: 304,
					},
					Scope: "person",
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.RawPosition{Text: ".Names", Offset: 168},
						Scope:    "person",
					},
					{
						Position: position.RawPosition{Text: ".Age", Offset: 184},
						Scope:    "person",
					},
					{
						Position: position.RawPosition{Text: ".Address.Street", Offset: 220},
						Scope:    "person",
					},
					{
						Position: position.RawPosition{Text: ".Address.City", Offset: 248},
						Scope:    "person",
					},
					{
						Position: position.RawPosition{Text: ".HasJob", Offset: 262},
						Scope:    "person",
					},
					{
						Position: position.RawPosition{Text: ".GetJob", Offset: 279},
						Scope:    "person",
					},
				},
				Functions: []parser.VariableLocation{
					{
						Position: position.RawPosition{Text: "upper", Offset: 289},
						MethodArguments: []types.Type{
							&parser.VariableLocation{
								Position: position.RawPosition{Text: ".GetJob", Offset: 279},
								Scope:    "person",
							},
						},
						Scope: "person",
					},
				},
				EndPosition: position.RawPosition{Text: "}}", Offset: 311},
			},
			{
				Name:          "animal",
				StartPosition: position.RawPosition{Text: "{{define \"animal\"}}", Offset: 315},
				TypeHint: &parser.TypeHint{
					TypePath: "github.com/walteh/go-tmpl-types-vscode/examples/types.Animal",
					Position: position.RawPosition{
						Text:   "github.com/walteh/go-tmpl-types-vscode/examples/types.Animal",
						Offset: 339,
					},
					StartPosition: position.RawPosition{
						Text:   "{{/*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Animal*/}}",
						Offset: 339,
					},
					EndPosition: position.RawPosition{
						Text:   "{{/*gotype: github.com/walteh/go-tmpl-types-vscode/examples/types.Animal*/}}Name: {{.Name}}\n",
						Offset: 432,
					},
					Scope: "animal",
				},
				Variables: []parser.VariableLocation{
					{
						Position: position.RawPosition{Text: ".Name", Offset: 425},
						Scope:    "animal",
					},
				},
				Functions:   []parser.VariableLocation{},
				EndPosition: position.RawPosition{Text: "}}", Offset: 438},
			},
			{
				Name:          "test.tmpl",
				StartPosition: position.RawPosition{Text: "<<SOF>>", Offset: 0},
				TypeHint:      nil,
				Variables:     []parser.VariableLocation{},
				Functions:     []parser.VariableLocation{},
				EndPosition:   position.RawPosition{Text: "<<EOF>>", Offset: 441},
			},
		},
	}

	ctx := context.Background()

	got, err := parser.Parse(ctx, []byte(data), "test.tmpl")
	require.NoError(t, err)

	assert.EqualExportedValues(t, want, got)
}

func TestUseRegexToFindStartOfBlock(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		blockName   string
		wantText    string
		wantOffset  int
		wantErr     bool
		errContains string
	}{
		{
			name: "simple define block",
			content: `{{define "header"}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 0,
		},
		{
			name: "define block with dashes",
			content: `{{- define "header" -}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{- define "header" -}}`,
			wantOffset: 0,
		},
		{
			name: "block with dot argument",
			content: `{{define "header" .}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header" .}}`,
			wantOffset: 0,
		},
		{
			name: "block with complex dot argument",
			content: `{{define "header" .User.Name}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header" .User.Name}}`,
			wantOffset: 0,
		},
		{
			name: "multiple definitions",
			content: `{{define "header"}}
	First definition
{{end}}

{{define "other"}}
	Other block
{{end}}

{{define "header"}}
	Second definition
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: "block \"header\" is defined multiple times: found at line 1, line 9",
		},
		{
			name: "block with special characters in name",
			content: `{{define "header.sub-section_1"}}
	Some content
{{end}}`,
			blockName:  "header.sub-section_1",
			wantText:   `{{define "header.sub-section_1"}}`,
			wantOffset: 0,
		},
		{
			name: "block with mixed whitespace and dashes",
			content: `{{-  define  "header"  -}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{-  define  "header"  -}}`,
			wantOffset: 0,
		},
		{
			name: "block definition with comment",
			content: `{{/* comment */}}{{define "header"}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 17,
		},
		{
			name: "non-existent block",
			content: `{{define "other"}}
	Some content
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: `block "header" not found in template`,
		},
		{
			name: "malformed block name",
			content: `{{define "header}}
	Some content
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: `block "header" not found in template`,
		},
		{
			name: "nested block definitions",
			content: `{{define "outer"}}
	{{define "inner"}}
		Some content
	{{end}}
{{end}}`,
			blockName:  "inner",
			wantText:   `{{define "inner"}}`,
			wantOffset: 20,
		},
		{
			name: "block with escaped quotes",
			content: `{{define "header\"quote"}}
	Some content
{{end}}`,
			blockName:  `header"quote`,
			wantText:   `{{define "header\"quote"}}`,
			wantOffset: 0,
		},
		{
			name: "block with unicode name",
			content: `{{define "header_🚀_test"}}
	Some content
{{end}}`,
			blockName:  "header_🚀_test",
			wantText:   `{{define "header_🚀_test"}}`,
			wantOffset: 0,
		},
		{
			name: "block with very long name",
			content: `{{define "this.is.a.very.long.block.name.with.lots.of.dots.and.more.dots.to.make.it.really.long"}}
	Some content
{{end}}`,
			blockName:  "this.is.a.very.long.block.name.with.lots.of.dots.and.more.dots.to.make.it.really.long",
			wantText:   `{{define "this.is.a.very.long.block.name.with.lots.of.dots.and.more.dots.to.make.it.really.long"}}`,
			wantOffset: 0,
		},
		{
			name: "block with regex special chars in name",
			content: `{{define "header[]*+?{}"}}
	Some content
{{end}}`,
			blockName:  "header[]*+?{}",
			wantText:   `{{define "header[]*+?{}"}}`,
			wantOffset: 0,
		},
		{
			name: "multiple blocks with similar names",
			content: `{{define "header"}}
	First
{{end}}
{{define "header2"}}
	Second
{{end}}
{{define "header_"}}
	Third
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 0,
		},
		{
			name: "block with newlines in definition",
			content: `{{define 
"header"
}}
	Some content
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: `block "header" not found in template`,
		},
		{
			name: "block with HTML-like content",
			content: `{{define "header"}}<div>
	{{ if .Condition }}
		<span>{{ .Value }}</span>
	{{ end }}
</div>{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 0,
		},
		{
			name: "block with comment-like name",
			content: `{{define "/*header*/"}}
	Some content
{{end}}`,
			blockName:  "/*header*/",
			wantText:   `{{define "/*header*/"}}`,
			wantOffset: 0,
		},
		{
			name: "empty block name",
			content: `{{define ""}}
	Some content
{{end}}`,
			blockName:  "",
			wantText:   `{{define ""}}`,
			wantOffset: 0,
		},
		{
			name: "block with only whitespace",
			content: `{{define "   "}}
	Some content
{{end}}`,
			blockName:  "   ",
			wantText:   `{{define "   "}}`,
			wantOffset: 0,
		},
		{
			name: "multiple blocks with comments between",
			content: `{{/* first comment */}}
{{define "header"}}
	Content
{{end}}
{{/* second comment */}}
{{define "header"}}
	More content
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: "block \"header\" is defined multiple times",
		},
		{
			name: "block with mismatched quotes",
			content: `{{define "header'}}
	Some content
{{end}}`,
			blockName:   "header",
			wantErr:     true,
			errContains: `block "header" not found in template`,
		},
		{
			name: "block with extra closing braces",
			content: `{{define "header"}}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 0,
		},
		{
			name: "block with template syntax in name",
			content: `{{define "{{header}}"}}
	Some content
{{end}}`,
			blockName:  "{{header}}",
			wantText:   `{{define "{{header}}"}}`,
			wantOffset: 0,
		},
		{
			name: "define vs block keyword",
			content: `{{block "header" .}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{block "header" .}}`,
			wantOffset: 0,
		},
		{
			name: "extremely long content before block",
			content: strings.Repeat("x", 10000) + `{{define "header"}}
	Some content
{{end}}`,
			blockName:  "header",
			wantText:   `{{define "header"}}`,
			wantOffset: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := parser.UseRegexToFindStartOfBlock(ctx, tt.content, tt.blockName)

			if tt.wantErr {
				require.Error(t, err, "expected error but got none")
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains, "error message mismatch")
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantText, got.Text, "text mismatch")
			assert.Equal(t, tt.wantOffset, got.Offset, "offset mismatch")
		})
	}
}
