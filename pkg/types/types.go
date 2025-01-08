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
	Name   string
	Fields map[string]*FieldInfo
	// RootMethods map[string]*MethodInfo
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name       string
	Type       types.Type
	MethodInfo *MethodInfo
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
	ValidateType(ctx context.Context, typePath string, registry ast.PackageAnalyzer) (*TypeInfo, error)
	// ValidateField validates a field access on a type
	ValidateField(ctx context.Context, typeInfo *TypeInfo, fieldPath string) (*FieldInfo, error)
	// ValidateMethod validates a method call on a type
	ValidateMethod(ctx context.Context, methodName string) (*MethodInfo, error)
	// GetRootMethods returns the map of root methods
	GetRootMethods() map[string]*MethodInfo
}

// DefaultValidator is the default implementation of Validator
type DefaultValidator struct {
	RootMethods map[string]*MethodInfo
}

var knownMethods = map[string]*MethodInfo{
	// "and":      and,
	// "call":     emptyCall,
	// "html":     HTMLEscaper,
	// "index":    index,
	// "slice":    slice,
	// "js":       JSEscaper,
	// "len":      length,
	// "not":      not,
	// "or":       or,
	// "print":    fmt.Sprint,
	// "printf":   fmt.Sprintf,
	// "println":  fmt.Sprintln,
	// "urlquery": URLQueryEscaper,

	// // Comparisons
	// "eq": eq, // ==
	// "ge": ge, // >=
	// "gt": gt, // >
	// "le": le, // <=
	// "lt": lt, // <
	// "ne": ne, // !=
	"upper": {
		Name:       "upper",
		Parameters: []types.Type{types.Typ[types.String]},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"and": {
		Name:       "and",
		Parameters: []types.Type{types.Typ[types.Bool], types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"call": {
		Name:       "call",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"html": {
		Name:       "html",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"index": {
		Name:       "index",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"slice": {
		Name:       "slice",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.Typ[types.Int], types.Typ[types.Int]},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"js": {
		Name:       "js",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"len": {
		Name:       "len",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Int]},
	},
	"not": {
		Name:       "not",
		Parameters: []types.Type{types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"or": {
		Name:       "or",
		Parameters: []types.Type{types.Typ[types.Bool], types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"print": {
		Name:       "print",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"printf": {
		Name:       "printf",
		Parameters: []types.Type{types.Typ[types.String], types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"println": {
		Name:       "println",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"urlquery": {
		Name:       "urlquery",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"eq": {
		Name:       "eq",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"ge": {
		Name:       "ge",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"gt": {
		Name:       "gt",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"le": {
		Name:       "le",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"lt": {
		Name:       "lt",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"ne": {
		Name:       "ne",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
}

// NewDefaultValidator creates a new DefaultValidator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		RootMethods: knownMethods,
	}
}

// ValidateType implements Validator
func (v *DefaultValidator) ValidateType(ctx context.Context, typePath string, registry ast.PackageAnalyzer) (*TypeInfo, error) {
	// Split the type path into package and type name
	lastDot := strings.LastIndex(typePath, ".")
	if lastDot == -1 {
		return nil, errors.Errorf("invalid type path: %s", typePath)
	}

	pkgName, typeName := typePath[:lastDot], typePath[lastDot+1:]

	// Get the package from the registry
	pkg, err := registry.GetPackage(ctx, pkgName)
	if err != nil {
		return nil, errors.Errorf("package not found in registry: %w", err)
	}

	// Find the type in the package scope
	obj := pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil, errors.Errorf("type %s not found in package %s", typeName, pkgName)
	}

	// Get the type information
	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("type %s is not a named type", typeName)
	}

	// Get the underlying struct type
	structType, ok := namedType.Underlying().(*types.Struct)
	if !ok {
		return nil, errors.Errorf("type %s is not a struct type", typeName)
	}

	// Create TypeInfo with fields
	typeInfo := &TypeInfo{
		Name:   typeName,
		Fields: make(map[string]*FieldInfo),
	}

	// Add fields to the type info
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		typeInfo.Fields[field.Name()] = &FieldInfo{
			Name: field.Name(),
			Type: field.Type(),
		}
	}

	// Add methods to the type info
	for i := 0; i < namedType.NumMethods(); i++ {
		method := namedType.Method(i)
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

		typeInfo.Fields[method.Name()] = &FieldInfo{
			Name:       method.Name(),
			Type:       method.Type(),
			MethodInfo: methodInfo,
		}
	}

	return typeInfo, nil
}

// ValidateField implements Validator
func (v *DefaultValidator) ValidateField(ctx context.Context, typeInfo *TypeInfo, fieldPath string) (*FieldInfo, error) {
	parts := strings.Split(fieldPath, ".")
	currentType := typeInfo

	// Check if we're in a range context
	if len(parts) > 0 {
		if field, ok := currentType.Fields[parts[0]]; ok {
			// If it's a range variable, we need to get the element type of the slice/array
			if sliceType, ok := field.Type.(*types.Slice); ok {
				// Create a new type info for the element type
				elemType := sliceType.Elem()
				if named, ok := elemType.(*types.Named); ok {
					if structType, ok := named.Underlying().(*types.Struct); ok {
						currentType = &TypeInfo{
							Name:   named.Obj().Name(),
							Fields: make(map[string]*FieldInfo),
						}
						// Add fields from the struct
						for i := 0; i < structType.NumFields(); i++ {
							f := structType.Field(i)
							currentType.Fields[f.Name()] = &FieldInfo{
								Name: f.Name(),
								Type: f.Type(),
							}
						}

						// Add methods from the named type
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

							currentType.Fields[method.Name()] = &FieldInfo{
								Name:       method.Name(),
								Type:       method.Type(),
								MethodInfo: methodInfo,
							}
						}

						// If we're accessing a field directly on the range variable, return it
						if len(parts) == 1 {
							if field, ok := currentType.Fields[parts[0]]; ok {
								return field, nil
							}
						}
					}
				}
			}
		}
	}

	// Handle the field path normally
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
			Name:   part,
			Fields: make(map[string]*FieldInfo),
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
func (v *DefaultValidator) ValidateMethod(ctx context.Context, methodName string) (*MethodInfo, error) {
	method, ok := v.RootMethods[methodName]
	if !ok {
		return nil, errors.Errorf("method %s not known", methodName)
	}
	return method, nil
}

func (v *DefaultValidator) GetRootMethods() map[string]*MethodInfo {
	return v.RootMethods
}
