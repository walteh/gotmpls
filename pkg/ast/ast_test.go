package ast_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
)

func TestTemplateNode_Basic(t *testing.T) {
	tests := []struct {
		name      string
		template  *ast.TemplateNode
		wantStart ast.Position
		wantEnd   ast.Position
	}{
		{
			name: "simple template with type hint",
			template: func() *ast.TemplateNode {
				tmpl := ast.NewTemplateNode(ast.Position{Line: 1, Column: 1})
				tmpl.TypeHint = ast.NewTypeHintNode("github.com/example/types.Config", ast.Position{Line: 1, Column: 1})

				def := ast.NewDefinitionNode("main", ast.Position{Line: 2, Column: 1}, ast.Position{Line: 4, Column: 1})
				action := ast.NewActionNode(ast.Position{Line: 3, Column: 1}, ast.Position{Line: 3, Column: 10})
				action.Pipeline = []ast.Node{
					ast.NewVariableNode("Name", ast.Position{Line: 3, Column: 3}),
				}
				def.Body = []ast.Node{action}
				tmpl.Definitions = []*ast.DefinitionNode{def}
				return tmpl
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 4, Column: 1},
		},
		{
			name: "empty template",
			template: func() *ast.TemplateNode {
				return ast.NewTemplateNode(ast.Position{Line: 1, Column: 1})
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 1, Column: 1},
		},
		{
			name: "multiple definitions",
			template: func() *ast.TemplateNode {
				tmpl := ast.NewTemplateNode(ast.Position{Line: 1, Column: 1})
				def1 := ast.NewDefinitionNode("header", ast.Position{Line: 2, Column: 1}, ast.Position{Line: 4, Column: 1})
				def2 := ast.NewDefinitionNode("footer", ast.Position{Line: 5, Column: 1}, ast.Position{Line: 7, Column: 1})
				tmpl.Definitions = []*ast.DefinitionNode{def1, def2}
				return tmpl
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 7, Column: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := tt.template.Position()
			assert.Equal(t, tt.wantStart, start, "template start position should match")
			assert.Equal(t, tt.wantEnd, end, "template end position should match")
		})
	}
}

func TestActionNode_Basic(t *testing.T) {
	tests := []struct {
		name      string
		action    *ast.ActionNode
		wantStart ast.Position
		wantEnd   ast.Position
	}{
		{
			name: "simple variable reference",
			action: func() *ast.ActionNode {
				action := ast.NewActionNode(ast.Position{Line: 1, Column: 1}, ast.Position{Line: 1, Column: 10})
				action.Pipeline = []ast.Node{
					ast.NewVariableNode("Name", ast.Position{Line: 1, Column: 3}),
				}
				return action
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 1, Column: 10},
		},
		{
			name: "function call",
			action: func() *ast.ActionNode {
				action := ast.NewActionNode(ast.Position{Line: 1, Column: 1}, ast.Position{Line: 1, Column: 20})
				fn := ast.NewFunctionNode("printf", ast.Position{Line: 1, Column: 3})
				fn.Arguments = []ast.Node{
					ast.NewVariableNode("Name", ast.Position{Line: 1, Column: 10}),
				}
				action.Pipeline = []ast.Node{fn}
				return action
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 1, Column: 20},
		},
		{
			name: "empty action",
			action: func() *ast.ActionNode {
				return ast.NewActionNode(ast.Position{Line: 1, Column: 1}, ast.Position{Line: 1, Column: 5})
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 1, Column: 5},
		},
		{
			name: "multiple pipeline nodes",
			action: func() *ast.ActionNode {
				action := ast.NewActionNode(ast.Position{Line: 1, Column: 1}, ast.Position{Line: 1, Column: 30})
				fn1 := ast.NewFunctionNode("upper", ast.Position{Line: 1, Column: 3})
				fn2 := ast.NewFunctionNode("trim", ast.Position{Line: 1, Column: 15})
				action.Pipeline = []ast.Node{fn1, fn2}
				return action
			}(),
			wantStart: ast.Position{Line: 1, Column: 1},
			wantEnd:   ast.Position{Line: 1, Column: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := tt.action.Position()
			assert.Equal(t, tt.wantStart, start, "action start position should match")
			assert.Equal(t, tt.wantEnd, end, "action end position should match")
		})
	}
}

func TestPackageAnalyzer_AnalyzePackage(t *testing.T) {
	tests := []struct {
		name       string
		packageDir string
		want       *ast.TypeRegistry
		wantErr    bool
	}{
		{
			name:       "valid package",
			packageDir: "testdata/valid",
			want: func() *ast.TypeRegistry {
				info := ast.NewTypeRegistry()
				info.Types["github.com/example/types"] = types.NewPackage("github.com/example/types", "types")
				return info
			}(),
			wantErr: false,
		},
		{
			name:       "invalid package",
			packageDir: "testdata/invalid",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "nonexistent package",
			packageDir: "testdata/nonexistent",
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockAnalyzer := mockery.NewMockPackageAnalyzer_ast(t)

			if !tt.wantErr {
				mockAnalyzer.EXPECT().
					AnalyzePackage(ctx, tt.packageDir).
					Return(tt.want, nil)
			} else {
				mockAnalyzer.EXPECT().
					AnalyzePackage(ctx, tt.packageDir).
					Return(nil, assert.AnError)
			}

			got, err := mockAnalyzer.AnalyzePackage(ctx, tt.packageDir)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTypeRegistry_TypeExists(t *testing.T) {
	tests := []struct {
		name     string
		typePath string
		setup    func() *ast.TypeRegistry
		want     bool
	}{
		{
			name:     "existing type",
			typePath: "github.com/example/types.Person",
			setup: func() *ast.TypeRegistry {
				info := ast.NewTypeRegistry()
				info.Types["github.com/example/types.Person"] = types.NewPackage("github.com/example/types", "types")
				return info
			},
			want: true,
		},
		{
			name:     "non-existent type",
			typePath: "github.com/example/types.NonExistent",
			setup: func() *ast.TypeRegistry {
				return ast.NewTypeRegistry()
			},
			want: false,
		},
		{
			name:     "empty registry",
			typePath: "any.Type",
			setup: func() *ast.TypeRegistry {
				return ast.NewTypeRegistry()
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.setup()
			got := registry.TypeExists(tt.typePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVariableNode_Position(t *testing.T) {
	tests := []struct {
		name      string
		variable  *ast.VariableNode
		wantStart ast.Position
		wantEnd   ast.Position
	}{
		{
			name:      "simple variable",
			variable:  ast.NewVariableNode("Name", ast.Position{Line: 1, Column: 3}),
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 7}, // Column + len("Name")
		},
		{
			name:      "empty variable name",
			variable:  ast.NewVariableNode("", ast.Position{Line: 1, Column: 3}),
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 3},
		},
		{
			name:      "nested variable",
			variable:  ast.NewVariableNode("User.Name", ast.Position{Line: 1, Column: 3}),
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 12}, // Column + len("User.Name")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := tt.variable.Position()
			assert.Equal(t, tt.wantStart, start, "variable start position should match")
			assert.Equal(t, tt.wantEnd, end, "variable end position should match")
		})
	}
}

func TestFunctionNode_Position(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *ast.FunctionNode
		wantStart ast.Position
		wantEnd   ast.Position
	}{
		{
			name: "function without arguments",
			setup: func() *ast.FunctionNode {
				return ast.NewFunctionNode("len", ast.Position{Line: 1, Column: 3})
			},
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 6}, // Column + len("len")
		},
		{
			name: "function with single argument",
			setup: func() *ast.FunctionNode {
				fn := ast.NewFunctionNode("printf", ast.Position{Line: 1, Column: 3})
				fn.Arguments = []ast.Node{
					ast.NewVariableNode("Name", ast.Position{Line: 1, Column: 10}),
				}
				return fn
			},
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 14}, // End of "Name" argument
		},
		{
			name: "function with multiple arguments",
			setup: func() *ast.FunctionNode {
				fn := ast.NewFunctionNode("printf", ast.Position{Line: 1, Column: 3})
				fn.Arguments = []ast.Node{
					ast.NewVariableNode("Format", ast.Position{Line: 1, Column: 10}),
					ast.NewVariableNode("Value", ast.Position{Line: 1, Column: 18}),
				}
				return fn
			},
			wantStart: ast.Position{Line: 1, Column: 3},
			wantEnd:   ast.Position{Line: 1, Column: 23}, // End of last argument
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.setup()
			start, end := fn.Position()
			assert.Equal(t, tt.wantStart, start, "function start position should match")
			assert.Equal(t, tt.wantEnd, end, "function end position should match")
		})
	}
}

func createMockRegistry(t *testing.T) *ast.TypeRegistry {
	// Create a new types.Package
	pkg := types.NewPackage("github.com/example/types", "types")

	// Create a mock struct type
	fields := []*types.Var{
		types.NewField(0, pkg, "Name", types.Typ[types.String], false),
		types.NewField(0, pkg, "Age", types.Typ[types.Int], false),
		types.NewField(0, pkg, "SimpleString", types.Typ[types.String], false),
	}
	structType := types.NewStruct(fields, nil)

	// Create the named type
	named := types.NewNamed(
		types.NewTypeName(0, pkg, "Person", nil),
		structType,
		nil,
	)

	// Add a method
	sig := types.NewSignature(
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.String])),
		false,
	)
	named.AddMethod(types.NewFunc(0, pkg, "GetName", sig))

	// Store in package scope
	scope := pkg.Scope()
	scope.Insert(named.Obj())

	// Create and return the type registry
	registry := ast.NewTypeRegistry()
	registry.Types[pkg.Path()] = pkg
	return registry
}

func TestTypeRegistry_ValidateType(t *testing.T) {
	registry := createMockRegistry(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		typePath string
		wantErr  bool
		check    func(*testing.T, *ast.TypeInfo)
	}{
		{
			name:     "valid type",
			typePath: "github.com/example/types.Person",
			wantErr:  false,
			check: func(t *testing.T, info *ast.TypeInfo) {
				require.NotNil(t, info.Fields["Name"])
				assert.Equal(t, "string", info.Fields["Name"].Type.String())
				assert.Equal(t, "int", info.Fields["Age"].Type.String())
				assert.Equal(t, "string", info.Fields["SimpleString"].Type.String())
				assert.NotNil(t, info.Fields["GetName"], "method should be included in fields")
			},
		},
		{
			name:     "invalid type",
			typePath: "github.com/example/types.NonExistent",
			wantErr:  true,
		},
		{
			name:     "invalid package",
			typePath: "invalid/package.Type",
			wantErr:  true,
		},
		{
			name:     "invalid type path format",
			typePath: "invalidformat",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, err := registry.ValidateType(ctx, tt.typePath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, typeInfo)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, typeInfo)
			if tt.check != nil {
				tt.check(t, typeInfo)
			}
		})
	}
}

func TestTypeRegistry_GetFieldType(t *testing.T) {
	registry := createMockRegistry(t)
	pkg := types.NewPackage("test", "test")

	// Create a struct type for testing
	fields := []*types.Var{
		types.NewField(0, pkg, "Name", types.Typ[types.String], false),
		types.NewField(0, pkg, "Age", types.Typ[types.Int], false),
	}
	structType := types.NewStruct(fields, nil)

	tests := []struct {
		name      string
		fieldName string
		wantType  types.Type
		wantErr   bool
	}{
		{
			name:      "existing string field",
			fieldName: "Name",
			wantType:  types.Typ[types.String],
			wantErr:   false,
		},
		{
			name:      "existing int field",
			fieldName: "Age",
			wantType:  types.Typ[types.Int],
			wantErr:   false,
		},
		{
			name:      "non-existent field",
			fieldName: "NonExistent",
			wantType:  nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldType, err := registry.GetFieldType(structType, tt.fieldName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, fieldType)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantType, fieldType)
		})
	}
}
