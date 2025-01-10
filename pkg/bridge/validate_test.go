package bridge

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func createTestTypeInfo(t *testing.T) *TypeInfo {
	// Create a mock package
	pkg := types.NewPackage("test", "test")

	// Create the main type info
	return &TypeInfo{
		Name: "Person",
		Fields: map[string]*FieldInfo{
			"Name": {
				Name: "Name",
				Type: types.Typ[types.String],
			},
			"Age": {
				Name: "Age",
				Type: types.Typ[types.Int],
			},
			"Address": {
				Name: "Address",
				Type: types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "Street", types.Typ[types.String], false),
					types.NewField(0, pkg, "City", types.Typ[types.String], false),
				}, nil),
			},
			"SimpleString": {
				Name: "SimpleString",
				Type: types.Typ[types.String],
			},
		},
	}
}

func TestValidateField(t *testing.T) {
	typeInfo := createTestTypeInfo(t)
	ctx := context.Background()

	doc := position.NewDocument("dummy")

	tests := []struct {
		name      string
		fieldPath position.RawPosition
		wantErr   bool
		check     func(*testing.T, *FieldInfo)
	}{
		{
			name:      "simple string field",
			fieldPath: doc.NewBasicPosition("SimpleString", 0),
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				assert.Equal(t, "string", info.Type.String())
			},
		},
		{
			name:      "nested field first level",
			fieldPath: doc.NewBasicPosition("Address", 0),
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				structType, ok := info.Type.(*types.Struct)
				require.True(t, ok)
				assert.Equal(t, 2, structType.NumFields())
			},
		},
		{
			name:      "nested field second level",
			fieldPath: doc.NewBasicPosition("Address.Street", 0),
			wantErr:   false,
			check: func(t *testing.T, info *FieldInfo) {
				assert.Equal(t, "string", info.Type.String())
			},
		},
		{
			name:      "invalid root field",
			fieldPath: doc.NewBasicPosition("NonExistent", 0),
			wantErr:   true,
		},
		{
			name:      "invalid nested field",
			fieldPath: doc.NewBasicPosition("Address.NonExistent", 0),
			wantErr:   true,
		},
		{
			name:      "attempt to nest on simple type",
			fieldPath: doc.NewBasicPosition("SimpleString.Something", 0),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldInfo, err := ValidateField(ctx, typeInfo, tt.fieldPath)
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
