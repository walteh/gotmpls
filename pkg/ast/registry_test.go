package ast

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/bridge"
)

func createMockRegistry(t *testing.T) *Registry {
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
	registry := NewRegistry()
	registry.Types[pkg.Path()] = pkg
	return registry
}

func TestRegistry_ValidateType(t *testing.T) {
	registry := createMockRegistry(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		typePath string
		wantErr  bool
		check    func(*testing.T, *bridge.TypeInfo)
	}{
		{
			name:     "valid type",
			typePath: "github.com/example/types.Person",
			wantErr:  false,
			check: func(t *testing.T, info *bridge.TypeInfo) {
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
