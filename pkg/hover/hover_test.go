package hover_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/hover"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func TestFormatHoverResponse(t *testing.T) {
	tests := []struct {
		name     string
		variable *parser.VariableLocation
		method   *ast.TemplateMethodInfo
		want     []string
		wantErr  bool
	}{
		{
			name: "simple function call",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   ".GetJob",
					Offset: 0,
				},
				MethodArguments: nil,
				Scope:           "",
			},
			method: &ast.TemplateMethodInfo{
				Name: "upper",
				Parameters: []types.Type{
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
			want: []string{
				`### Template Function

.GetJob
    │
    ▼
upper

### Signature

` + "```go\n" + `func upper(arg1 string) string
` + "```" + `

### Template Usage

` + "```\n" + `.GetJob | upper
` + "```",
			},
			wantErr: false,
		},
		{
			name: "multiple function chain",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   ".GetJob",
					Offset: 0,
				},
				MethodArguments: []types.Type{
					&parser.VariableLocation{
						Position: position.RawPosition{
							Text:   "lower",
							Offset: 10,
						},
					},
				},
				Scope: "",
			},
			method: &ast.TemplateMethodInfo{
				Name: "upper",
				Parameters: []types.Type{
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
			want: []string{
				`### Template Function

.GetJob
    │
lower
    │
    ▼
upper

### Signature

` + "```go\n" + `func upper(arg1 string) string
` + "```" + `

### Template Usage

` + "```\n" + `.GetJob | upper
` + "```",
			},
			wantErr: false,
		},
		{
			name: "function with multiple arguments",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   ".Name",
					Offset: 0,
				},
				MethodArguments: []types.Type{
					types.Typ[types.String],
					types.Typ[types.String],
				},
				Scope: "",
			},
			method: &ast.TemplateMethodInfo{
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
			want: []string{
				`### Template Function

.Name
    │
    ▼
replace

### Signature

` + "```go\n" + `func replace(arg1 string, arg2 string, arg3 string) string
` + "```" + `

### Template Usage

` + "```\n" + `.Name | replace
` + "```",
			},
			wantErr: false,
		},
		{
			name: "simple variable",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   ".Name",
					Offset: 0,
				},
				Scope: "",
			},
			method: nil,
			want: []string{
				`### Variable

.Name`,
			},
			wantErr: false,
		},
		{
			name:     "nil variable",
			variable: nil,
			method:   nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hover.FormatHoverResponse(context.Background(), tt.variable, tt.method)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Content)
		})
	}
}

func TestFormatHoverResponseFunction(t *testing.T) {
	t.Skip() // TODO: support this, we probably need better parsing support for funcitons first
	tests := []struct {
		name     string
		method   *ast.TemplateMethodInfo
		variable parser.VariableLocation
		block    *parser.BlockInfo
		want     string
	}{
		{
			name: "simple function call",
			method: &ast.TemplateMethodInfo{
				Name: "upper",
				Parameters: []types.Type{
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
			variable: parser.VariableLocation{
				Position: position.RawPosition{
					Text: ".GetJob",
				},
				MethodArguments: []types.Type{},
			},
			block: &parser.BlockInfo{
				TypeHint: &parser.TypeHint{
					Position: position.RawPosition{
						Text: "github.com/example/pkg.Person",
					},
				},
			},
			want: `### Template Function

#### Signature
` + "```" + `go
func upper(arg1 string) string
` + "```" + `

#### Chain Preview
` + "```" + `go
func (me *Person) chainPreview() (string) {
    var out1 string = me.GetJob()
    return upper(out1)
}
` + "```" + `

#### Template Usage
` + "```" + `go-template
.GetJob | upper
` + "```" + ``,
		},
		{
			name: "multiple function chain",
			method: &ast.TemplateMethodInfo{
				Name: "upper",
				Parameters: []types.Type{
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.String],
				},
			},
			variable: parser.VariableLocation{
				Position: position.RawPosition{
					Text: ".GetJob",
				},
				MethodArguments: []types.Type{
					&parser.VariableLocation{
						Position: position.RawPosition{
							Text: "lower",
						},
					},
				},
			},
			block: &parser.BlockInfo{
				TypeHint: &parser.TypeHint{
					Position: position.RawPosition{
						Text: "github.com/example/pkg.Person",
					},
				},
			},
			want: `### Template Function

` + "```" + `
.GetJob
    │
    lower
    │
    ▼
upper
` + "```" + `

#### Signature
` + "```" + `go
func upper(arg1 string) string
` + "```" + `

#### Chain Preview
` + "```" + `go
func (me *Person) chainPreview() (string) {
    var out1 string = me.GetJob()
    var out2 string = lower(out1)
    return upper(out2)
}
` + "```" + `

#### Template Usage
` + "```" + `go-template
.GetJob | lower | upper
` + "```" + ``,
		},
		{
			name: "function with multiple arguments",
			method: &ast.TemplateMethodInfo{
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
			variable: parser.VariableLocation{
				Position: position.RawPosition{
					Text: ".Name",
				},
				MethodArguments: []types.Type{
					&parser.VariableLocation{
						Position: position.RawPosition{
							Text: `"old"`,
						},
					},
					&parser.VariableLocation{
						Position: position.RawPosition{
							Text: `"new"`,
						},
					},
				},
			},
			block: &parser.BlockInfo{
				TypeHint: &parser.TypeHint{
					Position: position.RawPosition{
						Text: "github.com/example/pkg.Person",
					},
				},
			},
			want: `### Template Function

` + "```" + `
.Name
    │
    ▼
replace "old" "new"
` + "```" + `

#### Signature
` + "```" + `go
func replace(arg1 string, arg2 string, arg3 string) string
` + "```" + `

#### Chain Preview
` + "```" + `go
func (me *Person) chainPreview() (string) {
    var out1 string = me.Name()
    return replace(out1, "old", "new")
}
` + "```" + `

#### Template Usage
` + "```" + `go-template
.Name | replace "old" "new"
` + "```" + ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := hover.FormatHoverResponse(ctx, &tt.variable, tt.method)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Content)
		})
	}
}
