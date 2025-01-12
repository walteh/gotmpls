package astreflect

import (
	"fmt"
	"go/types"
	"reflect"
)

func Reflect2AST(t reflect.Type) types.Type {
	if t == nil {
		return types.NewInterfaceType(nil, nil)
	}

	switch t.Kind() {
	case reflect.Bool:
		return types.Typ[types.Bool]
	case reflect.Int:
		return types.Typ[types.Int]
	case reflect.Int8:
		return types.Typ[types.Int8]
	case reflect.Int16:
		return types.Typ[types.Int16]
	case reflect.Int32:
		return types.Typ[types.Int32]
	case reflect.Int64:
		return types.Typ[types.Int64]
	case reflect.Uint:
		return types.Typ[types.Uint]
	case reflect.Uint8:
		return types.Typ[types.Uint8]
	case reflect.Uint16:
		return types.Typ[types.Uint16]
	case reflect.Uint32:
		return types.Typ[types.Uint32]
	case reflect.Uint64:
		return types.Typ[types.Uint64]
	case reflect.Float32:
		return types.Typ[types.Float32]
	case reflect.Float64:
		return types.Typ[types.Float64]
	case reflect.Complex64:
		return types.Typ[types.Complex64]
	case reflect.Complex128:
		return types.Typ[types.Complex128]
	case reflect.String:
		return types.Typ[types.String]
	case reflect.Array:
		return types.NewArray(Reflect2AST(t.Elem()), int64(t.Len()))
	case reflect.Slice:
		return types.NewSlice(Reflect2AST(t.Elem()))
	case reflect.Map:
		return types.NewMap(Reflect2AST(t.Key()), Reflect2AST(t.Elem()))
	case reflect.Ptr:
		return types.NewPointer(Reflect2AST(t.Elem()))
	case reflect.Struct:
		var fields []*types.Var
		var tags []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fields = append(fields, types.NewVar(0, nil, field.Name, Reflect2AST(field.Type)))
			if tag := field.Tag; tag != "" {
				tags = append(tags, fmt.Sprintf("`%s`", tag))
			} else {
				tags = append(tags, "")
			}
		}
		return types.NewStruct(fields, tags)
	case reflect.Interface:
		// For now, return empty interface. In future we could potentially
		// extract methods and create a proper interface type
		return types.NewInterfaceType(nil, nil)
	default:
		return types.NewInterfaceType(nil, nil)
	}
}
