package types

import (
	"context"
	"go/types"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"gitlab.com/tozd/go/errors"
)

// TypeInfo represents information about a Go type
type TypeInfo struct {
	Name    string
	Fields  map[string]*FieldInfo
	Methods map[string]*MethodInfo
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name string
	Type types.Type
}

// MethodInfo represents information about a method
type MethodInfo struct {
	Name       string
	Parameters []types.Type
	Results    []types.Type
}

// Validator is responsible for validating types in templates
type Validator interface {
	// ValidateType validates a type against package information
	ValidateType(ctx context.Context, typePath string, registry *ast.TypeRegistry) (*TypeInfo, error)
	// ValidateField validates a field access on a type
	ValidateField(ctx context.Context, typeInfo *TypeInfo, fieldPath string) (*FieldInfo, error)
	// ValidateMethod validates a method call on a type
	ValidateMethod(ctx context.Context, typeInfo *TypeInfo, methodName string) (*MethodInfo, error)
}

// DefaultValidator is the default implementation of Validator
type DefaultValidator struct{}

// NewDefaultValidator creates a new DefaultValidator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// ValidateType implements Validator
func (v *DefaultValidator) ValidateType(ctx context.Context, typePath string, registry *ast.TypeRegistry) (*TypeInfo, error) {
	// Split the type path into package path and type name
	lastDot := strings.LastIndex(typePath, ".")
	if lastDot == -1 {
		return nil, errors.Errorf("invalid type path %s: must be in format package.Type", typePath)
	}

	pkgPath := typePath[:lastDot]
	typeName := typePath[lastDot+1:]

	pkg, ok := registry.Types[pkgPath]
	if !ok {
		known := ""
		for k := range registry.Types {
			known += k + " 	"

		}
		return nil, errors.Errorf("package %s not found in registry\n\nKnown packages:\n%s", pkgPath, known)
	}

	scope := pkg.Scope()
	obj := scope.Lookup(typeName)
	if obj == nil {
		return nil, errors.Errorf("type %s not found in package %s", typeName, pkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("type %s is not a named type", typeName)
	}

	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, errors.Errorf("type %s is not a struct type", typeName)
	}

	info := &TypeInfo{
		Name:    typeName,
		Fields:  make(map[string]*FieldInfo),
		Methods: make(map[string]*MethodInfo),
	}

	// Get fields
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		info.Fields[field.Name()] = &FieldInfo{
			Name: field.Name(),
			Type: field.Type(),
		}
	}

	// Get methods
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		sig := method.Type().(*types.Signature)

		methodInfo := &MethodInfo{
			Name:       method.Name(),
			Parameters: make([]types.Type, sig.Params().Len()),
			Results:    make([]types.Type, sig.Results().Len()),
		}

		for j := 0; j < sig.Params().Len(); j++ {
			methodInfo.Parameters[j] = sig.Params().At(j).Type()
		}

		for j := 0; j < sig.Results().Len(); j++ {
			methodInfo.Results[j] = sig.Results().At(j).Type()
		}

		info.Methods[method.Name()] = methodInfo
	}

	return info, nil
}

// ValidateField implements Validator
func (v *DefaultValidator) ValidateField(ctx context.Context, typeInfo *TypeInfo, fieldPath string) (*FieldInfo, error) {
	parts := strings.Split(fieldPath, ".")
	currentType := typeInfo

	for i, part := range parts {
		field, ok := currentType.Fields[part]
		if !ok {
			return nil, errors.Errorf("field %s not found in type %s", part, currentType.Name)
		}

		// If this is the last part, return the field info
		if i == len(parts)-1 {
			return field, nil
		}

		// Get the underlying type for the next iteration
		underlying := field.Type
		// If it's a named type, get its underlying type
		if named, ok := underlying.(*types.Named); ok {
			underlying = named.Underlying()
		}

		// Check if it's a struct type (either directly or after getting underlying)
		structType, ok := underlying.(*types.Struct)
		if !ok {
			return nil, errors.Errorf("field %s is not a struct type", part)
		}

		// Create new type info for the nested type
		nextType := &TypeInfo{
			Name:    part,
			Fields:  make(map[string]*FieldInfo),
			Methods: make(map[string]*MethodInfo),
		}

		// Add fields from the struct
		for i := 0; i < structType.NumFields(); i++ {
			f := structType.Field(i)
			nextType.Fields[f.Name()] = &FieldInfo{
				Name: f.Name(),
				Type: f.Type(),
			}
		}

		currentType = nextType
	}

	return nil, errors.Errorf("unexpected error validating field path %s", fieldPath)
}

// ValidateMethod implements Validator
func (v *DefaultValidator) ValidateMethod(ctx context.Context, typeInfo *TypeInfo, methodName string) (*MethodInfo, error) {
	method, ok := typeInfo.Methods[methodName]
	if !ok {
		return nil, errors.Errorf("method %s not found in type %s", methodName, typeInfo.Name)
	}
	return method, nil
}
