package types

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
)

// mockTypeRegistry creates a mock type registry for testing
func mockTypeRegistry(t *testing.T) *ast.TypeRegistry {
	// Create a new types.Package
	pkg := types.NewPackage("github.com/example/types", "types")

	// Create a mock struct type
	fields := []*types.Var{
		types.NewField(0, pkg, "Name", types.Typ[types.String], false),
		types.NewField(0, pkg, "Age", types.Typ[types.Int], false),
		types.NewField(0, pkg, "Address", createAddressType(t, pkg), false),
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

// createAddressType creates a nested address type for testing
func createAddressType(t *testing.T, pkg *types.Package) types.Type {
	fields := []*types.Var{
		types.NewField(0, pkg, "Street", types.Typ[types.String], false),
		types.NewField(0, pkg, "City", types.Typ[types.String], false),
		types.NewField(0, pkg, "Location", createLocationType(t, pkg), false),
	}
	return types.NewStruct(fields, nil)
}

// createLocationType creates a deeply nested type for testing
func createLocationType(t *testing.T, pkg *types.Package) types.Type {
	fields := []*types.Var{
		types.NewField(0, pkg, "Latitude", types.Typ[types.Float64], false),
		types.NewField(0, pkg, "Longitude", types.Typ[types.Float64], false),
	}
	return types.NewStruct(fields, nil)
}

func TestDefaultValidator_ValidateType(t *testing.T) {
	validator := &DefaultValidator{}
	registry := mockTypeRegistry(t)

	tests := []struct {
		name     string
		typePath string
		wantErr  bool
		check    func(*testing.T, *TypeInfo)
	}{
		{
			name:     "valid type",
			typePath: "github.com/example/types.Person",
			wantErr:  false,
			check: func(t *testing.T, info *TypeInfo) {
				require.NotNil(t, info.Fields["Name"])
				assert.Equal(t, "string", info.Fields["Name"].Type.String())

				require.NotNil(t, info.Fields["Address"])
				addressType, ok := info.Fields["Address"].Type.(*types.Struct)
				require.True(t, ok)
				assert.Equal(t, 3, addressType.NumFields()) // Street, City, Location
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			typeInfo, err := validator.ValidateType(ctx, tt.typePath, registry)
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

func TestDefaultValidator_ValidateField(t *testing.T) {
	validator := &DefaultValidator{}
	registry := mockTypeRegistry(t)
	ctx := context.Background()
	typeInfo, err := validator.ValidateType(ctx, "github.com/example/types.Person", registry)
	require.NoError(t, err)

	tests := []struct {
		name      string
		fieldPath string
		wantErr   bool
		check     func(*testing.T, *FieldInfo)
	}{
		{
			name:      "simple string field",
			fieldPath: "SimpleString",
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				assert.Equal(t, "string", info.Type.String())
			},
		},
		{
			name:      "nested field first level",
			fieldPath: "Address",
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				structType, ok := info.Type.(*types.Struct)
				require.True(t, ok)
				assert.Equal(t, 3, structType.NumFields())
			},
		},
		{
			name:      "nested field second level",
			fieldPath: "Address.Street",
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				assert.Equal(t, "string", info.Type.String())
			},
		},
		{
			name:      "nested field third level",
			fieldPath: "Address.Location.Latitude",
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				assert.Equal(t, "float64", info.Type.String())
			},
		},
		{
			name:      "invalid root field",
			fieldPath: "NonExistent",
			wantErr:   true,
		},
		{
			name:      "invalid nested field",
			fieldPath: "Address.NonExistent",
			wantErr:   true,
		},
		{
			name:      "invalid deep nested field",
			fieldPath: "Address.Location.NonExistent",
			wantErr:   true,
		},
		{
			name:      "attempt to nest on simple type",
			fieldPath: "SimpleString.Something",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldInfo, err := validator.ValidateField(ctx, typeInfo, tt.fieldPath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, fieldInfo)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, fieldInfo)
			if tt.check != nil {
				tt.check(t, fieldInfo)
			}
		})
	}
}

func TestDefaultValidator_ValidateMethod(t *testing.T) {
	validator := &DefaultValidator{
		RootMethods: map[string]*MethodInfo{
			"eq": {
				Name: "eq",
				Parameters: []types.Type{
					types.Typ[types.String],
					types.Typ[types.String],
				},
				Results: []types.Type{
					types.Typ[types.Bool],
				},
			},
		},
	}
	// registry := mockTypeRegistry(t)
	// typeInfo, err := validator.ValidateType(context.Background(), "github.com/example/types.Person", registry)
	// require.NoError(t, err)

	tests := []struct {
		name       string
		methodName string
		wantErr    bool
	}{
		{
			name:       "valid method",
			methodName: "eq",
			wantErr:    false,
		},
		{
			name:       "invalid method",
			methodName: "NonExistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			methodInfo, err := validator.ValidateMethod(ctx, tt.methodName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, methodInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, methodInfo)
			}
		})
	}
}
