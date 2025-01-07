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
	}{
		{
			name:     "valid type",
			typePath: "github.com/example/types.Person",
			wantErr:  false,
		},
		{
			name:     "invalid type",
			typePath: "github.com/example/types.NonExistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, err := validator.ValidateType(context.Background(), tt.typePath, registry)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, typeInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, typeInfo)
			}
		})
	}
}

func TestDefaultValidator_ValidateField(t *testing.T) {
	validator := &DefaultValidator{}
	registry := mockTypeRegistry(t)
	typeInfo, err := validator.ValidateType(context.Background(), "github.com/example/types.Person", registry)
	require.NoError(t, err)

	tests := []struct {
		name      string
		fieldPath string
		wantErr   bool
	}{
		{
			name:      "valid field",
			fieldPath: "Name",
			wantErr:   false,
		},
		{
			name:      "nested field",
			fieldPath: "Address.Street",
			wantErr:   false,
		},
		{
			name:      "invalid field",
			fieldPath: "NonExistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldInfo, err := validator.ValidateField(context.Background(), typeInfo, tt.fieldPath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, fieldInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, fieldInfo)
			}
		})
	}
}

func TestDefaultValidator_ValidateMethod(t *testing.T) {
	validator := &DefaultValidator{}
	registry := mockTypeRegistry(t)
	typeInfo, err := validator.ValidateType(context.Background(), "github.com/example/types.Person", registry)
	require.NoError(t, err)

	tests := []struct {
		name       string
		methodName string
		wantErr    bool
	}{
		{
			name:       "valid method",
			methodName: "GetName",
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
			methodInfo, err := validator.ValidateMethod(context.Background(), typeInfo, tt.methodName)
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
