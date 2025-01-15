package ast

import (
	"context"
	"go/types"
	"strings"

	"github.com/rs/zerolog"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// TypeHintDefinition represents information about a Go type
type TypeHintDefinition struct {

	// Reflect reflect.Type
	Fields      map[string]*FieldInfo
	MyFieldInfo FieldInfo
	MyType      *types.Named
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name string
	Type FieldVarOrFunc
	// Reflect  reflect.Type
	FormattedTypeString string
	Parent              *TypeHintDefinition
}

func (f *FieldInfo) TypeName() string {
	if named, ok := f.Type.Type().(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

type FieldVarOrFunc struct {
	Var  *types.Var
	Func *types.Func
}

func (f FieldVarOrFunc) Type() types.Type {
	if f.Var != nil {
		return f.Var.Type()
	}
	if f.Func != nil {
		return f.Func.Type()
	}
	return nil
}

func (f FieldVarOrFunc) String() string {
	if f.Var != nil {
		return f.Var.Type().String()
	}
	if f.Func != nil {
		return f.Func.Type().String()
	}
	return "<unknown>"
}

func (f FieldVarOrFunc) Obj() types.Object {
	if f.Var != nil {
		return f.Var
	}
	if f.Func != nil {
		return f.Func
	}
	return nil
}

func (f FieldVarOrFunc) Underlying() types.Type {
	if f.Var != nil {
		return f.Var.Type().Underlying()
	}
	if f.Func != nil {
		return f.Func.Type().Underlying()
	}
	return nil
}

// createFieldInfo creates a new FieldInfo from a types.Object (can be Var or Func)
func createFieldInfo(ctx context.Context, obj FieldVarOrFunc, parent *TypeHintDefinition) (*FieldInfo, error) {
	if obj.Var != nil {
		zerolog.Ctx(ctx).Debug().
			Str("type", obj.Var.Type().String()).
			Str("name", obj.Var.Name()).
			Msg("creating field info for var")
	} else if obj.Func != nil {
		zerolog.Ctx(ctx).Debug().
			Str("type", obj.Func.Type().String()).
			Str("name", obj.Func.Name()).
			Msg("creating field info for func")
	}
	return &FieldInfo{
		Type:   obj,
		Parent: parent,
	}, nil
}

// createTypeInfoFromStruct creates a TypeInfo from a types.Struct
func createTypeInfoFromStruct(ctx context.Context, name string, obj types.Type, strict bool, parent *TypeHintDefinition) (*TypeHintDefinition, error) {
	zerolog.Ctx(ctx).Debug().
		Str("name", name).
		Str("type", obj.String()).
		Bool("strict", strict).
		Msg("creating type info")

	typeInfo := &TypeHintDefinition{
		Fields: make(map[string]*FieldInfo),
	}

	// Create a FieldInfo for the type itself
	typeInfo.MyFieldInfo = FieldInfo{
		Name:   name,
		Type:   FieldVarOrFunc{}, // Empty for now since this is the root type
		Parent: parent,
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
			fieldInfo, err := createFieldInfo(ctx, FieldVarOrFunc{Var: field}, typeInfo)
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
			methodInfo, err := createFieldInfo(ctx, FieldVarOrFunc{Func: method}, typeInfo)
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
		zerolog.Ctx(ctx).Debug().Str("part", part).Msgf("generating field '%s' in type '%s' using position '%s'", part, currentType.MyFieldInfo.Name, pos.ID())
		field, ok := currentType.Fields[part]
		if !ok {
			// "field not found" is relied on downstream in hover.go
			return nil, errors.Errorf("field not found [ %s ] in type [ %s ]", part, currentType.MyFieldInfo.Name)
		}

		currentField = field

		if part != parts[len(parts)-1] {
			var err error
			// Get the underlying type if it's a named type
			fieldType := field.Type.Type()
			if named, ok := fieldType.(*types.Named); ok {
				fieldType = named.Underlying()
			}

			// Check if it's a struct type
			structType, ok := fieldType.(*types.Struct)
			if !ok {
				return nil, errors.Errorf("field %s is not a struct type", part)
			}

			currentType, err = createTypeInfoFromStruct(ctx, part, structType, false, currentType)
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

func GenerateFunctionCallInfoFromSignature(ctx context.Context, signature *types.Signature) (*TemplateMethodInfo, error) {
	input := []types.Type{}
	output := []types.Type{}

	for i := 0; i < signature.Params().Len(); i++ {
		input = append(input, signature.Params().At(i).Type())
	}

	for i := 0; i < signature.Results().Len(); i++ {
		output = append(output, signature.Results().At(i).Type())
	}

	return &TemplateMethodInfo{
		Name:       signature.String(),
		Parameters: input,
		Results:    output,
	}, nil
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

	typeInfo, err := createTypeInfoFromStruct(ctx, typeName, obj.Type(), true, nil)
	if err != nil {
		return nil, errors.Errorf("failed to create type info: %w", err)
	}

	return typeInfo, nil
}
