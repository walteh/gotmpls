package ast_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
)

func createMockRegistry(t *testing.T) *ast.Registry {
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
	registry := ast.NewRegistry()
	registry.Types[pkg.Path()] = pkg
	return registry
}

func TestRegistry_GetFieldType(t *testing.T) {
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

func TestRegistry_GetPackage(t *testing.T) {
	registry := createMockRegistry(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		packageName string
		wantErr     bool
	}{
		{
			name:        "exact match",
			packageName: "github.com/example/types",
			wantErr:     false,
		},
		{
			name:        "match by name",
			packageName: "types",
			wantErr:     false,
		},
		{
			name:        "match by suffix",
			packageName: "example/types",
			wantErr:     false,
		},
		{
			name:        "non-existent package",
			packageName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := registry.GetPackage(ctx, tt.packageName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, pkg)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, pkg)
		})
	}
}

func TestRegistry_TypeExists(t *testing.T) {
	registry := createMockRegistry(t)

	tests := []struct {
		name     string
		typePath string
		want     bool
	}{
		{
			name:     "existing package",
			typePath: "github.com/example/types",
			want:     true,
		},
		{
			name:     "non-existent package",
			typePath: "nonexistent",
			want:     false,
		},
		{
			name:     "partial match",
			typePath: "example/types",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.TypeExists(tt.typePath)
			assert.Equal(t, tt.want, got)
		})
	}
}
