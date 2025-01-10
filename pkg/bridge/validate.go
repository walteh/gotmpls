package bridge

import (
	"context"
	"go/types"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/position"
	"gitlab.com/tozd/go/errors"
)

// ValidateField validates a field access on a type
func ValidateField(ctx context.Context, typeInfo *TypeInfo, pos position.RawPosition) (*FieldInfo, error) {
	// Split the field path into components
	parts := strings.Split(pos.Text(), ".")

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
