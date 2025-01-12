package astreflect

import (
	"fmt"
	"go/types"
	"reflect"
	"unsafe"
)

// AST2Reflect converts a go/types.Type to a reflect.Type.
// Note that some conversions may not be possible due to runtime limitations.
// In such cases, it will return reflect.TypeOf(interface{}{}) and an error.
func AST2Reflect(t types.Type) (reflect.Type, error) {
	if t == nil {
		return reflect.TypeOf((*interface{})(nil)).Elem(), nil
	}

	switch t := t.(type) {
	case *types.Basic:
		return basicType2Reflect(t)
	case *types.Array:
		elem, err := AST2Reflect(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("converting array element type: %w", err)
		}
		return reflect.ArrayOf(int(t.Len()), elem), nil
	case *types.Slice:
		elem, err := AST2Reflect(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("converting slice element type: %w", err)
		}
		return reflect.SliceOf(elem), nil
	case *types.Map:
		key, err := AST2Reflect(t.Key())
		if err != nil {
			return nil, fmt.Errorf("converting map key type: %w", err)
		}
		elem, err := AST2Reflect(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("converting map value type: %w", err)
		}
		return reflect.MapOf(key, elem), nil
	case *types.Pointer:
		elem, err := AST2Reflect(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("converting pointer element type: %w", err)
		}
		return reflect.PtrTo(elem), nil
	case *types.Struct:
		return struct2Reflect(t)
	case *types.Interface:
		fmt.Println("interface")
		return interface2Reflect(t)
	case *types.Signature:
		return signature2Reflect(t)
	case *types.Named:
		return AST2Reflect(t.Underlying())
	case *types.Chan:
		elem, err := AST2Reflect(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("converting channel element type: %w", err)
		}
		var dir reflect.ChanDir
		switch t.Dir() {
		case types.SendRecv:
			dir = reflect.BothDir
		case types.SendOnly:
			dir = reflect.SendDir
		case types.RecvOnly:
			dir = reflect.RecvDir
		}
		return reflect.ChanOf(dir, elem), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}
}

func basicType2Reflect(t *types.Basic) (reflect.Type, error) {
	switch t.Kind() {
	case types.Bool:
		return reflect.TypeOf(bool(false)), nil
	case types.Int:
		return reflect.TypeOf(int(0)), nil
	case types.Int8:
		return reflect.TypeOf(int8(0)), nil
	case types.Int16:
		return reflect.TypeOf(int16(0)), nil
	case types.Int32:
		return reflect.TypeOf(int32(0)), nil
	case types.Int64:
		return reflect.TypeOf(int64(0)), nil
	case types.Uint:
		return reflect.TypeOf(uint(0)), nil
	case types.Uint8:
		return reflect.TypeOf(uint8(0)), nil
	case types.Uint16:
		return reflect.TypeOf(uint16(0)), nil
	case types.Uint32:
		return reflect.TypeOf(uint32(0)), nil
	case types.Uint64:
		return reflect.TypeOf(uint64(0)), nil
	case types.Float32:
		return reflect.TypeOf(float32(0)), nil
	case types.Float64:
		return reflect.TypeOf(float64(0)), nil
	case types.Complex64:
		return reflect.TypeOf(complex64(0)), nil
	case types.Complex128:
		return reflect.TypeOf(complex128(0)), nil
	case types.String:
		return reflect.TypeOf(string("")), nil
	case types.UnsafePointer:
		return reflect.TypeOf(unsafe.Pointer(nil)), nil
	case types.Uintptr:
		return reflect.TypeOf(uintptr(0)), nil
	default:
		return nil, fmt.Errorf("unsupported basic type kind: %v", t.Kind())
	}
}

func struct2Reflect(t *types.Struct) (reflect.Type, error) {
	var fields []reflect.StructField
	for i := 0; i < t.NumFields(); i++ {
		field := t.Field(i)
		fieldType, err := AST2Reflect(field.Type())
		if err != nil {
			return nil, fmt.Errorf("converting field %q type: %w", field.Name(), err)
		}

		// Parse the tag string, removing the backticks if present
		tag := t.Tag(i)
		if len(tag) >= 2 && tag[0] == '`' && tag[len(tag)-1] == '`' {
			tag = tag[1 : len(tag)-1]
		}

		fields = append(fields, reflect.StructField{
			Name: field.Name(),
			Type: fieldType,
			Tag:  reflect.StructTag(tag),
		})
	}
	return reflect.StructOf(fields), nil
}

func interface2Reflect(t *types.Interface) (reflect.Type, error) {
	// methodTypes := make(map[string]reflect.Type)
	// for i := 0; i < t.NumMethods(); i++ {
	// 	method := t.Method(i)
	// 	fmt.Println(method.Name())
	// 	if methodTypes[method.Name()] != nil {
	// 		continue
	// 	}
	// 	mtype, err := signature2Reflect(method.Type().(*types.Signature))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("converting method %q: %w", method.Name(), err)
	// 	}
	// 	methodTypes[method.Name()] = mtype
	// }

	// // Create a struct type that will implement our interface
	// structType := reflect.StructOf(nil) // empty struct as base
	// structValue := reflect.New(structType)

	// // Create method set
	// for name, typ := range methodTypes {
	// 	// Create a method implementation that panics (we only need the type)
	// 	fn := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
	// 		panic("method not implemented")
	// 	})
	// 	structValue.MethodByName(name).Set(fn)
	// }

	// Get the interface type from the struct type
	// return structValue.Type().Elem(), nil

	return reflect.TypeOf((*interface{})(nil)).Elem(), nil
}

func signature2Reflect(t *types.Signature) (reflect.Type, error) {
	var in []reflect.Type

	// Don't include receiver in input parameters for method types
	if recv := t.Recv(); recv != nil {
		// Skip receiver for method types
		_, err := AST2Reflect(recv.Type())
		if err != nil {
			return nil, fmt.Errorf("converting receiver type: %w", err)
		}
	}

	params := t.Params()
	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		paramType, err := AST2Reflect(param.Type())
		if err != nil {
			return nil, fmt.Errorf("converting parameter %d type: %w", i, err)
		}
		in = append(in, paramType)
	}

	var out []reflect.Type
	results := t.Results()
	for i := 0; i < results.Len(); i++ {
		result := results.At(i)
		resultType, err := AST2Reflect(result.Type())
		if err != nil {
			return nil, fmt.Errorf("converting result %d type: %w", i, err)
		}
		out = append(out, resultType)
	}

	return reflect.FuncOf(in, out, t.Variadic()), nil
}
