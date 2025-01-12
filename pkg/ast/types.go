package ast

import (
	"context"
	"go/types"
	"reflect"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// TypeHintDefinition represents information about a Go type
type TypeHintDefinition struct {
	Name string
	Type types.Type
	// Reflect reflect.Type
	Fields map[string]*FieldInfo
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name     string
	Type     types.Type
	Reflect  reflect.Type
	FullName string
}

// createFieldInfo creates a new FieldInfo from a types.Object (can be Var or Func)
func createFieldInfo(obj types.Object) (*FieldInfo, error) {
	// reflectType, err := astreflect.AST2Reflect(obj.Type())
	// if err != nil {
	// 	return nil, errors.Errorf("failed to reflect field %s: %w", obj.Name(), err)
	// }

	return &FieldInfo{
		Name: obj.Name(),
		Type: obj.Type(),
		// Reflect:  reflectType,
		FullName: obj.String(),
	}, nil
}

// createTypeInfoFromStruct creates a TypeInfo from a types.Struct
func createTypeInfoFromStruct(name string, obj types.Type, strict bool) (*TypeHintDefinition, error) {
	typeInfo := &TypeHintDefinition{
		Name:   name,
		Fields: make(map[string]*FieldInfo),
	}

	var namedType *types.Named
	var structType *types.Struct

	switch t := obj.(type) {
	case *types.Named:
		namedType = t
		if s, ok := t.Underlying().(*types.Struct); ok {
			structType = s
		}
	case *types.Struct:
		structType = t
	}

	if structType == nil && strict {
		return nil, errors.Errorf("type %s is not a struct type", name)
	}

	if structType != nil {
		for i := 0; i < structType.NumFields(); i++ {
			field := structType.Field(i)
			fieldInfo, err := createFieldInfo(field)
			if err != nil {
				return nil, errors.Errorf("failed to create field info for %s: %w", field.Name(), err)
			}
			if _, ok := typeInfo.Fields[field.Name()]; ok {
				return nil, errors.Errorf("name conflict: %s already exists in type %s", field.Name(), name)
			}
			typeInfo.Fields[field.Name()] = fieldInfo
		}
	}

	if namedType != nil {
		// Add methods
		for i := 0; i < namedType.NumMethods(); i++ {
			method := namedType.Method(i)
			methodInfo, err := createFieldInfo(method)
			if err != nil {
				return nil, errors.Errorf("failed to create method info for %s: %w", method.Name(), err)
			}
			if _, ok := typeInfo.Fields[method.Name()]; ok {
				return nil, errors.Errorf("name conflict: %s already exists in type %s", method.Name(), name)
			}
			typeInfo.Fields[method.Name()] = methodInfo
		}
	}

	return typeInfo, nil
}

// ValidateField validates a field access on a type
func GenerateFieldInfoFromPosition(ctx context.Context, typeInfo *TypeHintDefinition, pos position.RawPosition) (*FieldInfo, error) {
	parts := strings.Split(pos.Text, ".")
	currentType := typeInfo
	var currentField *FieldInfo

	for _, part := range parts {
		if part == "" {
			continue
		}

		field, ok := currentType.Fields[part]
		if !ok {
			return nil, errors.Errorf("field %s not found in type %s", part, currentType.Name)
		}

		currentField = field

		if part != parts[len(parts)-1] {
			var err error
			currentType, err = createTypeInfoFromStruct(part, field.Type, false)
			if err != nil {
				return nil, errors.Errorf("failed to create type info for %s: %w", part, err)
			}
		}
	}

	return currentField, nil
}

type FunctionCallInfo struct {
	Name    string
	Args    []*types.Var
	Results []*types.Var
}

func GenerateFunctionCallInfoFromPosition(ctx context.Context, pos position.RawPosition) (*TemplateMethodInfo, error) {

	method := BuiltinTemplateMethods[pos.Text]
	if method == nil {
		return nil, errors.Errorf("method %s not found", pos.Text)
	}

	// methodType, ok := method.Type.(*types.Func)
	// if !ok {
	// 	return nil, errors.Errorf("expected method %s to be a function, got %s", pos.Text, methodType.Type().String())
	// }

	// signature, ok := method..(*types.Signature)
	// if !ok {
	// 	return nil, errors.Errorf("expected method %s to have a signature, got %s", pos.Text, method.Type.String())
	// }
	return method, nil
	// input := []*types.Var{}
	// output := []*types.Var{}

	// for i := 0; i < signature.Params().Len(); i++ {
	// 	input = append(input, signature.Params().At(i))
	// }

	// for i := 0; i < signature.Results().Len(); i++ {
	// 	output = append(output, signature.Results().At(i))
	// }

	// return &FunctionCallInfo{
	// 	Name:    pos.Text,
	// 	Args:    input,
	// 	Results: output,
	// }, nil

}

func BuildTypeHintDefinitionFromRegistry(ctx context.Context, typePath string, r *Registry) (*TypeHintDefinition, error) {
	lastDot := strings.LastIndex(typePath, ".")
	if lastDot == -1 {
		return nil, errors.Errorf("invalid type path: %s", typePath)
	}

	pkgName, typeName := typePath[:lastDot], typePath[lastDot+1:]

	pkg, err := r.GetPackage(ctx, pkgName)
	if err != nil {
		return nil, errors.Errorf("package not found in registry: %w", err)
	}

	obj := pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil, errors.Errorf("type %s not found in package %s", typeName, pkgName)
	}

	typeInfo, err := createTypeInfoFromStruct(typeName, obj.Type(), true)
	if err != nil {
		return nil, errors.Errorf("failed to create type info: %w", err)
	}

	return typeInfo, nil
}
