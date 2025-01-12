package astreflect

import (
	"go/types"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomString string
type CustomStruct struct {
	Field1 int    `json:"field1"`
	Field2 string `json:"field2"`
}

func TestReflect2AST(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		validate func(*testing.T, types.Type)
	}{
		{
			name:  "nil input",
			input: nil,
			validate: func(t *testing.T, typ types.Type) {
				assert.IsType(t, (*types.Interface)(nil), typ, "should return empty interface for nil input")
			},
		},
		{
			name:  "bool",
			input: bool(true),
			validate: func(t *testing.T, typ types.Type) {
				assert.Equal(t, types.Typ[types.Bool], typ, "should return bool type")
			},
		},
		{
			name:  "int",
			input: int(42),
			validate: func(t *testing.T, typ types.Type) {
				assert.Equal(t, types.Typ[types.Int], typ, "should return int type")
			},
		},
		{
			name:  "string",
			input: string("test"),
			validate: func(t *testing.T, typ types.Type) {
				assert.Equal(t, types.Typ[types.String], typ, "should return string type")
			},
		},
		{
			name:  "float64",
			input: float64(3.14),
			validate: func(t *testing.T, typ types.Type) {
				assert.Equal(t, types.Typ[types.Float64], typ, "should return float64 type")
			},
		},
		{
			name:  "slice",
			input: []int{},
			validate: func(t *testing.T, typ types.Type) {
				slice, ok := typ.(*types.Slice)
				assert.True(t, ok, "should return slice type")
				assert.Equal(t, types.Typ[types.Int], slice.Elem(), "slice element should be int")
			},
		},
		{
			name:  "array",
			input: [3]string{},
			validate: func(t *testing.T, typ types.Type) {
				array, ok := typ.(*types.Array)
				assert.True(t, ok, "should return array type")
				assert.Equal(t, int64(3), array.Len(), "array length should be 3")
				assert.Equal(t, types.Typ[types.String], array.Elem(), "array element should be string")
			},
		},
		{
			name:  "map",
			input: map[string]int{},
			validate: func(t *testing.T, typ types.Type) {
				m, ok := typ.(*types.Map)
				assert.True(t, ok, "should return map type")
				assert.Equal(t, types.Typ[types.String], m.Key(), "map key should be string")
				assert.Equal(t, types.Typ[types.Int], m.Elem(), "map value should be int")
			},
		},
		{
			name:  "pointer",
			input: (*string)(nil),
			validate: func(t *testing.T, typ types.Type) {
				ptr, ok := typ.(*types.Pointer)
				assert.True(t, ok, "should return pointer type")
				assert.Equal(t, types.Typ[types.String], ptr.Elem(), "pointer element should be string")
			},
		},
		{
			name:  "struct",
			input: CustomStruct{},
			validate: func(t *testing.T, typ types.Type) {
				str, ok := typ.(*types.Struct)
				assert.True(t, ok, "should return struct type")
				assert.Equal(t, 2, str.NumFields(), "struct should have 2 fields")
				assert.Equal(t, "Field1", str.Field(0).Name(), "first field should be Field1")
				assert.Equal(t, "Field2", str.Field(1).Name(), "second field should be Field2")
				assert.Equal(t, "`json:\"field1\"`", str.Tag(0), "first field should have json tag")
				assert.Equal(t, "`json:\"field2\"`", str.Tag(1), "second field should have json tag")
			},
		},
		{
			name:  "interface",
			input: (interface{})(nil),
			validate: func(t *testing.T, typ types.Type) {
				iface, ok := typ.(*types.Interface)
				assert.True(t, ok, "should return interface type")
				assert.Equal(t, 0, iface.NumMethods(), "interface should have no methods")
			},
		},
		{
			name:  "nested types",
			input: []map[string]*CustomStruct{},
			validate: func(t *testing.T, typ types.Type) {
				slice, ok := typ.(*types.Slice)
				assert.True(t, ok, "should return slice type")

				m, ok := slice.Elem().(*types.Map)
				assert.True(t, ok, "slice element should be map")
				assert.Equal(t, types.Typ[types.String], m.Key(), "map key should be string")

				ptr, ok := m.Elem().(*types.Pointer)
				assert.True(t, ok, "map value should be pointer")

				str, ok := ptr.Elem().(*types.Struct)
				assert.True(t, ok, "pointer element should be struct")
				assert.Equal(t, 2, str.NumFields(), "struct should have 2 fields")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reflectType reflect.Type
			if tt.input != nil {
				reflectType = reflect.TypeOf(tt.input)
			}
			result := Reflect2AST(reflectType)
			tt.validate(t, result)
		})
	}
}
