package ast_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/pkg/ast"
	"github.com/walteh/gotmpls/pkg/position"
)

func createTestContext(t *testing.T) context.Context {
	// Create a test logger that writes to the test log
	logger := zerolog.New(zerolog.TestWriter{T: t}).With().Timestamp().Logger()
	return logger.WithContext(context.Background())
}

func createTestTypeInfo(t *testing.T) *ast.TypeHintDefinition {
	// Create a mock package
	pkg := types.NewPackage("test", "test")

	// Create variables for the fields
	nameVar := types.NewField(0, pkg, "Name", types.Typ[types.String], false)
	ageVar := types.NewField(0, pkg, "Age", types.Typ[types.Int], false)
	streetVar := types.NewField(0, pkg, "Street", types.Typ[types.String], false)
	cityVar := types.NewField(0, pkg, "City", types.Typ[types.String], false)
	simpleStringVar := types.NewField(0, pkg, "SimpleString", types.Typ[types.String], false)

	// Create the main type info
	return &ast.TypeHintDefinition{
		MyFieldInfo: ast.FieldInfo{
			Name: "Person",
			Type: ast.FieldVarOrFunc{
				Var: types.NewField(0, pkg, "Person", types.NewStruct([]*types.Var{nameVar, ageVar}, nil), false),
			},
		},
		Fields: map[string]*ast.FieldInfo{
			"Name": {
				Name: "Name",
				Type: ast.FieldVarOrFunc{Var: nameVar},
			},
			"Age": {
				Name: "Age",
				Type: ast.FieldVarOrFunc{Var: ageVar},
			},
			"Address": {
				Name: "Address",
				Type: ast.FieldVarOrFunc{
					Var: types.NewField(0, pkg, "Address", types.NewStruct([]*types.Var{streetVar, cityVar}, nil), false),
				},
			},
			"SimpleString": {
				Name: "SimpleString",
				Type: ast.FieldVarOrFunc{Var: simpleStringVar},
			},
		},
	}
}

func TestGenerateFieldInfoFromPosition(t *testing.T) {
	typeInfo := createTestTypeInfo(t)
	ctx := createTestContext(t)

	tests := []struct {
		name      string
		fieldPath position.RawPosition
		wantErr   bool
		check     func(*testing.T, *ast.FieldInfo)
	}{
		{
			name:      "simple string field",
			fieldPath: position.NewBasicPosition("SimpleString", 0),
			wantErr:   false,
			check: func(t *testing.T, info *ast.FieldInfo) {
				assert.Equal(t, "string", info.Type.Type().String())
			},
		},
		{
			name:      "nested field first level",
			fieldPath: position.NewBasicPosition("Address", 0),
			wantErr:   false,
			check: func(t *testing.T, info *ast.FieldInfo) {
				structType, ok := info.Type.Type().(*types.Struct)
				require.True(t, ok)
				assert.Equal(t, 2, structType.NumFields())
			},
		},
		{
			name:      "nested field second level",
			fieldPath: position.NewBasicPosition("Address.Street", 0),
			wantErr:   false,
			check: func(t *testing.T, info *ast.FieldInfo) {
				assert.Equal(t, "string", info.Type.Type().String())
			},
		},
		{
			name:      "invalid root field",
			fieldPath: position.NewBasicPosition("NonExistent", 0),
			wantErr:   true,
		},
		{
			name:      "invalid nested field",
			fieldPath: position.NewBasicPosition("Address.NonExistent", 0),
			wantErr:   true,
		},
		{
			name:      "attempt to nest on simple type",
			fieldPath: position.NewBasicPosition("SimpleString.Something", 0),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldInfo, err := ast.GenerateFieldInfoFromPosition(ctx, typeInfo, tt.fieldPath)
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

func TestGenerateTypeInfoFromRegistry(t *testing.T) {
	registry := createMockRegistry(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		typePath string
		wantErr  bool
		check    func(*testing.T, *ast.TypeHintDefinition)
	}{
		{
			name:     "valid type",
			typePath: "github.com/example/types.Person",
			wantErr:  false,
			check: func(t *testing.T, info *ast.TypeHintDefinition) {
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
			typeInfo, err := ast.BuildTypeHintDefinitionFromRegistry(ctx, tt.typePath, (*ast.Registry)(registry))
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
