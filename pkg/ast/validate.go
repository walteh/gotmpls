package ast

import (
	"context"
	"go/types"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// ValidateField validates a field access on a type
func GenerateFieldInfoFromPosition(ctx context.Context, typeInfo *TypeInfo, pos position.RawPosition) (*FieldInfo, error) {
	// Split the field path into components
	parts := strings.Split(pos.Text, ".")

	// Start with the root type info
	currentType := typeInfo
	var currentField *FieldInfo

	for _, part := range parts {
		// Skip empty parts (can happen with leading dots)
		if part == "" {
			continue
		}

		// Look up the field in the current type
		field, ok := currentType.Fields[part]
		if !ok {
			return nil, errors.Errorf("field %s not found in type %s", part, currentType.Name)
		}

		currentField = field

		// If this isn't the last part, we need to get the type info for the next level
		if part != parts[len(parts)-1] {
			// Get the underlying struct type if this is a struct field
			structType, ok := field.Type.Underlying().(*types.Struct)
			if !ok {
				return nil, errors.Errorf("field %s is not a struct type", part)
			}

			// Create a new TypeInfo for the struct
			currentType = &TypeInfo{
				Name:   part,
				Fields: make(map[string]*FieldInfo),
			}

			// Add all fields from the struct
			for i := 0; i < structType.NumFields(); i++ {
				f := structType.Field(i)
				currentType.Fields[f.Name()] = &FieldInfo{
					Name: f.Name(),
					Type: f.Type(),
				}
			}
		}
	}

	return currentField, nil
}

func GenerateTypeInfoFromRegistry(ctx context.Context, typePath string, r *Registry) (*TypeInfo, error) {
	// Split the type path into package and type name
	lastDot := strings.LastIndex(typePath, ".")
	if lastDot == -1 {
		return nil, errors.Errorf("invalid type path: %s", typePath)
	}

	pkgName, typeName := typePath[:lastDot], typePath[lastDot+1:]

	// Get the package from the registry
	pkg, err := r.GetPackage(ctx, pkgName)
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
		typeInfo.Fields[method.Name()] = &FieldInfo{
			Name: method.Name(),
			Type: method.Type(),
		}
	}

	return typeInfo, nil
}
