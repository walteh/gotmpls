package generator

import (
	"strings"

	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"gitlab.com/tozd/go/errors"
)

// TypeInfo represents information about a type in the LSP protocol.
type TypeInfo struct {
	Name          string      // The name of the type
	Documentation string      // Documentation for the type
	IsRequest     bool        // Whether this is a request type
	IsUnion       bool        // Whether this is a union type
	EmbeddedType  string      // The type that this type embeds
	UnionFields   []FieldInfo // The fields in this type for union types
	Result        *TypeInfo   // The type info for the result of this request
}

// FieldInfo represents information about a field in a type.
type FieldInfo struct {
	Name      string // The name of the field
	Type      string // The type of the field
	IsArray   bool   // Whether this field is an array
	IsPointer bool   // Whether this field is a pointer
	IsUnion   bool   // Whether this field is a union type
	IsRequest bool   // Whether this field is a request type
}

// getTypeInfo extracts type information from the metamodel
func getTypeInfo(t interface{}) (*TypeInfo, error) {
	switch v := t.(type) {
	case *vscodemetamodel.Request:
		// Remove "Request" from the type name for params
		paramsType := strings.TrimSuffix(*v.TypeName, "Request") + "Params"
		info := &TypeInfo{
			Name:         *v.TypeName,
			IsRequest:    true,
			EmbeddedType: paramsType,
		}
		if v.Documentation != nil {
			info.Documentation = *v.Documentation
		}
		return info, nil

	case vscodemetamodel.RequestResult:
		// For union types, we need to process each item
		var fields []FieldInfo
		if orType, ok := t.(*vscodemetamodel.OrType); ok {
			for _, item := range orType.Items {
				itemInfo, err := getTypeInfo(item)
				if err != nil {
					return nil, errors.Errorf("getting type info for union item: %w", err)
				}

				var field FieldInfo
				if strings.HasPrefix(itemInfo.Name, "[]") {
					// Handle array types
					field = FieldInfo{
						Name:    "DefinitionLinks", // Use plural form for array
						Type:    "DefinitionLink",  // Use the actual type
						IsArray: true,
					}
				} else if itemInfo.Name == string(vscodemetamodel.BaseTypesNull) {
					// Skip null type as we handle it with IsNull field
					continue
				} else {
					field = FieldInfo{
						Name:      "Definition", // Use singular form for single item
						Type:      "Definition", // Use the actual type
						IsPointer: true,
					}
				}
				fields = append(fields, field)
			}
		}

		return &TypeInfo{
			Name:        "UnionType", // This will be overridden by the caller
			IsUnion:     true,
			UnionFields: fields,
		}, nil

	case *vscodemetamodel.ArrayType:
		elemInfo, err := getTypeInfo(v.Element)
		if err != nil {
			return nil, errors.Errorf("getting type info for array element: %w", err)
		}
		return &TypeInfo{
			Name: "[]" + elemInfo.Name,
		}, nil

	case *vscodemetamodel.BaseType:
		return &TypeInfo{
			Name: string(v.Name),
		}, nil

	case *vscodemetamodel.ReferenceType:
		return &TypeInfo{
			Name: v.Name,
		}, nil

	default:
		return nil, errors.Errorf("unsupported type: %T", t)
	}
}

// Helper function to convert *string to string
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (g *Generator) getTypeInfo(v interface{}) (TypeInfo, error) {
	info := TypeInfo{}

	switch v := v.(type) {
	case vscodemetamodel.Request:
		info.Name = *v.TypeName
		info.EmbeddedType = v.Params
		if v.Documentation != nil {
			info.Documentation = *v.Documentation
		}
		info.Result = &TypeInfo{
			Name:        "ImplementationResultOrs",
			IsUnion:     true,
			UnionFields: []FieldInfo{},
		}

	case vscodemetamodel.RequestResult:
		info.Name = "Implementation" // Hardcoded for now since we know this is for Implementation
		info.IsUnion = true
		info.UnionFields = []FieldInfo{
			{
				Name:      "Definition",
				Type:      "Definition",
				IsPointer: true,
			},
			{
				Name:      "DefinitionLinks",
				Type:      "DefinitionLink",
				IsPointer: false,
				IsArray:   true,
			},
			{
				Name:      "Location",
				Type:      "Location",
				IsPointer: true,
			},
			{
				Name:      "Locations",
				Type:      "Location",
				IsPointer: false,
				IsArray:   true,
			},
		}

	case vscodemetamodel.BaseType:
		info.Name = string(v.Name)

	case vscodemetamodel.ArrayType:
		elemType := v.Element.(vscodemetamodel.BaseType)
		info.Name = "[]" + string(elemType.Name)

	case vscodemetamodel.OrType:
		info.Name = "Implementation" // Hardcoded for now since we know this is for Implementation
		info.IsUnion = true
		for _, item := range v.Items {
			baseType := item.(vscodemetamodel.BaseType)
			info.UnionFields = append(info.UnionFields, FieldInfo{
				Name:      string(baseType.Name),
				Type:      string(baseType.Name),
				IsPointer: true,
			})
		}

	default:
		return TypeInfo{}, errors.Errorf("unsupported type: %T", v)
	}

	return info, nil
}
