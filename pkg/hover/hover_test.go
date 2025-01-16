package hover_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/ast"
	"github.com/walteh/gotmpls/pkg/hover"
	"github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
)

func createTestContext(t *testing.T) context.Context {
	logger := zerolog.New(zerolog.TestWriter{T: t}).With().Timestamp().Logger()
	return logger.WithContext(context.Background())
}

// 	if arg.Variable != nil && arg.Variable.Name() == "GetJob" {
// 		return []types.Type{types.Typ[types.String]}
// 	}
// 	if arg.Type != nil && arg.Type.String() == "string" {
// 		return []types.Type{types.Typ[types.String]}
// 	}
// 	if arg.Variable != nil && arg.Variable.Name() == "Name" {
// 		return []types.Type{types.Typ[types.String]}
// 	}
// 	if arg.Variable != nil && arg.Variable.Name() == "upper" {
// 		return []types.Type{types.Typ[types.String]}
// 	}
// 	if arg.Variable != nil && arg.Variable.Name() == "replace" {
// 		return []types.Type{types.Typ[types.String]}
// 	}
// 	return []types.Type{}
// }

func TestFormatHoverResponse(t *testing.T) {
	tests := []struct {
		name     string
		variable *parser.VariableLocation
		method   *ast.TemplateMethodInfo
		field    *ast.FieldInfo
		want     []string
		wantErr  bool
	}{
		{
			name: "simple function call",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   "GetJob",
					Offset: 0,
				},
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
				"### Template Function\n",
				"```go\nfunc upper(arg1 string) string\n```",
				"### Template Usage\n```go-template\nupper GetJob\n```",
			},
		},
		{
			name: "piped function call",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   "GetJob",
					Offset: 0,
				},
				PipeArguments: []parser.VariableLocationOrType{
					{Type: types.Typ[types.String]},
				},
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
				"### Template Function\n",
				"```go\nfunc upper(arg1 string) string\n```",
				"### Template Usage\n```go-template\nGetJob | upper \"string\"\n```",
			},
		},
		{
			name: "struct field",
			variable: &parser.VariableLocation{
				Position: position.RawPosition{
					Text:   ".Name",
					Offset: 0,
				},
			},
			field: &ast.FieldInfo{
				Name: "Name",
				Type: ast.FieldVarOrFunc{
					Var: types.NewVar(0, nil, "Name", types.Typ[types.String]),
				},
				Parent: &ast.TypeHintDefinition{
					MyFieldInfo: ast.FieldInfo{
						Name: "Person",
						Type: ast.FieldVarOrFunc{
							Var: types.NewVar(0, nil, "Person", types.NewStruct([]*types.Var{}, nil)),
						},
					},
				},
			},
			want: []string{
				"### Type Information\n",
				"```go\ntype Person struct {\n\tName string\n}\n```",
				"\n### Template Access\n```go-template\n.Name\n```",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(t)
			got, err := hover.FormatHoverResponse(ctx, tt.variable, tt.method, tt.field)
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
				PipeArguments: []parser.VariableLocationOrType{},
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
				PipeArguments: []parser.VariableLocationOrType{
					{
						Variable: &parser.VariableLocation{
							Position: position.RawPosition{
								Text: "lower",
							},
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
				PipeArguments: []parser.VariableLocationOrType{
					{
						Type: types.Typ[types.String],
					},
					{
						Type: types.Typ[types.String],
					},
					{
						Variable: &parser.VariableLocation{
							Position: position.RawPosition{
								Text: `"new"`,
							},
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
			got, err := hover.FormatHoverResponse(ctx, &tt.variable, tt.method, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Content)
		})
	}
}
