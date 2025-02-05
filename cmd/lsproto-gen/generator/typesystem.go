package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"gitlab.com/tozd/go/errors"
)

// TypeInfo represents information about a type in our system
type TypeInfo struct {
	Name          string   // The Go type name
	GoType        string   // The actual Go type (e.g., string, int, etc.)
	IsPointer     bool     // Whether this is a pointer type
	IsBuiltin     bool     // Whether this is a builtin type
	IsNullable    bool     // Whether this type can be null
	IsUnion       bool     // Whether this is a union type
	Documentation string   // Documentation for this type
	Dependencies  []string // Names of other types this type depends on
	IsRecursive   bool     // Whether this type references itself
}

// TypeNamer handles type naming in our system
type TypeNamer struct {
	knownTypes    map[string]TypeInfo
	typeCount     map[string]int
	nextLiteralID int
}

// NewTypeNamer creates a new type namer
func NewTypeNamer() *TypeNamer {
	return &TypeNamer{
		knownTypes: make(map[string]TypeInfo),
		typeCount:  make(map[string]int),
	}
}

// getBaseType returns the Go type for a base type
func (n *TypeNamer) getBaseType(name string) string {
	switch name {
	case "string":
		return "string"
	case "integer":
		return "int32"
	case "uinteger":
		return "uint32"
	case "decimal":
		return "float64"
	case "boolean":
		return "bool"
	case "null":
		return "nil"
	default:
		return "interface{}"
	}
}

// GetTypeInfo returns type information for a given type
func (n *TypeNamer) GetTypeInfo(t interface{}) (TypeInfo, error) {
	if t == nil {
		return TypeInfo{}, errors.New("type is nil")
	}

	// Handle raw map[string]interface{} by treating it as a Type
	if raw, ok := t.(map[string]interface{}); ok {
		// Extract the kind
		kind, ok := raw["kind"].(string)
		if !ok {
			return TypeInfo{}, errors.Errorf("type has no kind field: %v", raw)
		}

		// Convert to appropriate type based on kind
		switch kind {
		case "base":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("base type has no name field: %v", raw)
			}
			info := TypeInfo{
				Name:      name,
				GoType:    n.getBaseType(name),
				IsBuiltin: true,
			}
			return info, nil

		case "reference":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("reference type has no name field: %v", raw)
			}
			info := TypeInfo{
				Name:      name,
				GoType:    name,
				IsBuiltin: false,
			}
			return info, nil

		case "array":
			elem, ok := raw["element"]
			if !ok {
				return TypeInfo{}, errors.New("array type missing element")
			}
			elemInfo, err := n.GetTypeInfo(elem)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting array element type: %w", err)
			}
			info := TypeInfo{
				Name:      fmt.Sprintf("[]%s", elemInfo.Name),
				GoType:    fmt.Sprintf("[]%s", elemInfo.GoType),
				IsBuiltin: false,
			}
			return info, nil

		case "map":
			key, ok := raw["key"]
			if !ok {
				return TypeInfo{}, errors.New("map type missing key")
			}
			value, ok := raw["value"]
			if !ok {
				return TypeInfo{}, errors.New("map type missing value")
			}
			keyInfo, err := n.GetTypeInfo(key)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting map key type: %w", err)
			}
			valueInfo, err := n.GetTypeInfo(value)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting map value type: %w", err)
			}
			info := TypeInfo{
				Name:      fmt.Sprintf("map[%s]%s", keyInfo.Name, valueInfo.Name),
				GoType:    fmt.Sprintf("map[%s]%s", keyInfo.GoType, valueInfo.GoType),
				IsBuiltin: false,
			}
			return info, nil

		case "or":
			items, ok := raw["items"].([]interface{})
			if !ok {
				return TypeInfo{}, errors.New("or type missing items")
			}
			// Create a union type name from the item types
			var itemInfos []TypeInfo
			for _, item := range items {
				itemInfo, err := n.GetTypeInfo(item)
				if err != nil {
					return TypeInfo{}, errors.Errorf("getting or type item: %w", err)
				}
				itemInfos = append(itemInfos, itemInfo)
			}
			// Sort item names for consistent naming
			sort.Slice(itemInfos, func(i, j int) bool {
				return itemInfos[i].Name < itemInfos[j].Name
			})
			var names []string
			for _, info := range itemInfos {
				names = append(names, info.Name)
			}
			name := fmt.Sprintf("Or%s", strings.Join(names, ""))
			info := TypeInfo{
				Name:         name,
				GoType:       name,
				IsBuiltin:    false,
				IsUnion:      true,
				Dependencies: names,
			}
			return info, nil

		case "tuple":
			items, ok := raw["items"].([]interface{})
			if !ok {
				return TypeInfo{}, errors.New("tuple type missing items")
			}
			// Create a tuple type name from the item types
			var itemInfos []TypeInfo
			for _, item := range items {
				itemInfo, err := n.GetTypeInfo(item)
				if err != nil {
					return TypeInfo{}, errors.Errorf("getting tuple type item: %w", err)
				}
				itemInfos = append(itemInfos, itemInfo)
			}
			// Sort item names for consistent naming
			sort.Slice(itemInfos, func(i, j int) bool {
				return itemInfos[i].Name < itemInfos[j].Name
			})
			var names []string
			for _, info := range itemInfos {
				names = append(names, info.Name)
			}
			name := fmt.Sprintf("Tuple%s", strings.Join(names, ""))
			info := TypeInfo{
				Name:         name,
				GoType:       name,
				IsBuiltin:    false,
				Dependencies: names,
			}
			return info, nil

		case "stringLiteral":
			value, ok := raw["value"].(string)
			if !ok {
				return TypeInfo{}, errors.New("string literal value is not a string")
			}
			info := TypeInfo{
				Name:      fmt.Sprintf("StringLiteral%s", strings.Title(value)),
				GoType:    "string",
				IsBuiltin: true,
			}
			return info, nil

		case "literal":
			if _, ok := raw["value"]; !ok {
				return TypeInfo{}, errors.New("literal type missing value")
			}
			// Create a unique name for this literal type
			name := fmt.Sprintf("Literal%d", n.nextLiteralID)
			n.nextLiteralID++
			info := TypeInfo{
				Name:      name,
				GoType:    name,
				IsBuiltin: false,
			}
			return info, nil

		default:
			return TypeInfo{}, errors.Errorf("unsupported type kind: %s", kind)
		}
	}

	// Handle vscodemetamodel types
	switch t := t.(type) {
	case *vscodemetamodel.BaseType:
		info := TypeInfo{
			Name:      string(t.Name),
			GoType:    n.getBaseType(string(t.Name)),
			IsBuiltin: true,
		}
		return info, nil

	case *vscodemetamodel.ArrayType:
		elemInfo, err := n.GetTypeInfo(t.Element)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting array element type: %w", err)
		}
		info := TypeInfo{
			Name:      fmt.Sprintf("[]%s", elemInfo.Name),
			GoType:    fmt.Sprintf("[]%s", elemInfo.GoType),
			IsBuiltin: false,
		}
		return info, nil

	case *vscodemetamodel.MapType:
		keyInfo, err := n.GetTypeInfo(t.Key)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting map key type: %w", err)
		}
		valueInfo, err := n.GetTypeInfo(t.Value)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting map value type: %w", err)
		}
		info := TypeInfo{
			Name:      fmt.Sprintf("map[%s]%s", keyInfo.Name, valueInfo.Name),
			GoType:    fmt.Sprintf("map[%s]%s", keyInfo.GoType, valueInfo.GoType),
			IsBuiltin: false,
		}
		return info, nil

	case *vscodemetamodel.OrType:
		var itemInfos []TypeInfo
		for _, item := range t.Items {
			itemInfo, err := n.GetTypeInfo(item)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting or type item: %w", err)
			}
			itemInfos = append(itemInfos, itemInfo)
		}
		// Sort item names for consistent naming
		sort.Slice(itemInfos, func(i, j int) bool {
			return itemInfos[i].Name < itemInfos[j].Name
		})
		var names []string
		for _, info := range itemInfos {
			names = append(names, info.Name)
		}
		name := fmt.Sprintf("Or%s", strings.Join(names, ""))
		info := TypeInfo{
			Name:         name,
			GoType:       name,
			IsBuiltin:    false,
			IsUnion:      true,
			Dependencies: names,
		}
		return info, nil

	case *vscodemetamodel.TupleType:
		var itemInfos []TypeInfo
		for _, item := range t.Items {
			itemInfo, err := n.GetTypeInfo(item)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting tuple type item: %w", err)
			}
			itemInfos = append(itemInfos, itemInfo)
		}
		// Sort item names for consistent naming
		sort.Slice(itemInfos, func(i, j int) bool {
			return itemInfos[i].Name < itemInfos[j].Name
		})
		var names []string
		for _, info := range itemInfos {
			names = append(names, info.Name)
		}
		name := fmt.Sprintf("Tuple%s", strings.Join(names, ""))
		info := TypeInfo{
			Name:         name,
			GoType:       name,
			IsBuiltin:    false,
			Dependencies: names,
		}
		return info, nil

	case *vscodemetamodel.StringLiteralType:
		info := TypeInfo{
			Name:      fmt.Sprintf("StringLiteral%s", strings.Title(t.Value)),
			GoType:    "string",
			IsBuiltin: true,
		}
		return info, nil

	case *vscodemetamodel.LiteralType:
		name := fmt.Sprintf("Literal%d", n.nextLiteralID)
		n.nextLiteralID++
		info := TypeInfo{
			Name:      name,
			GoType:    name,
			IsBuiltin: false,
		}
		return info, nil

	default:
		return TypeInfo{}, errors.Errorf("unsupported type: %T", t)
	}
}

// GetTypeInfoFromRaw gets or creates type info from a raw interface{} value
func (n *TypeNamer) GetTypeInfoFromRaw(t interface{}) (TypeInfo, error) {
	var info TypeInfo

	// Handle raw map[string]interface{} by treating it as a Type
	if raw, ok := t.(map[string]interface{}); ok {
		// Extract the kind
		kind, ok := raw["kind"].(string)
		if !ok {
			return TypeInfo{}, errors.Errorf("type has no kind field: %v", raw)
		}

		// Convert to appropriate type based on kind
		switch kind {
		case "base":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("base type has no name field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.Type{
				Kind: kind,
				Name: name,
			})
		case "reference":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("reference type has no name field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.Type{
				Kind: kind,
				Name: name,
			})
		default:
			return TypeInfo{}, errors.Errorf("unsupported raw type kind: %s", kind)
		}
	}

	// Handle vscodemetamodel.Type directly
	if t, ok := t.(*vscodemetamodel.Type); ok {
		return n.GetTypeInfo(t)
	}

	return TypeInfo{}, errors.Errorf("unsupported type: %T", t)
}

// GetTypeInfo gets or creates type info for a type
func (n *TypeNamer) GetTypeInfo(t interface{}) (TypeInfo, error) {
	var info TypeInfo

	// Handle raw map[string]interface{} by treating it as a Type
	if raw, ok := t.(map[string]interface{}); ok {
		// Extract the kind
		kind, ok := raw["kind"].(string)
		if !ok {
			return TypeInfo{}, errors.Errorf("type has no kind field: %v", raw)
		}

		// Convert to appropriate type based on kind
		switch kind {
		case "base":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("base type has no name field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.BaseType{
				Kind: kind,
				Name: vscodemetamodel.BaseTypes(name),
			})
		case "reference":
			name, ok := raw["name"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("reference type has no name field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.ReferenceType{
				Kind: kind,
				Name: name,
			})
		case "array":
			element, ok := raw["element"]
			if !ok {
				return TypeInfo{}, errors.Errorf("array type has no element field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.ArrayType{
				Kind:    kind,
				Element: element,
			})
		case "map":
			key, ok := raw["key"]
			if !ok {
				return TypeInfo{}, errors.Errorf("map type has no key field: %v", raw)
			}
			value, ok := raw["value"]
			if !ok {
				return TypeInfo{}, errors.Errorf("map type has no value field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.MapType{
				Kind:  kind,
				Key:   key,
				Value: value,
			})
		case "or":
			rawItems, ok := raw["items"].([]interface{})
			if !ok {
				return TypeInfo{}, errors.Errorf("or type has no items field: %v", raw)
			}

			// Convert items to the correct type
			items := make([]vscodemetamodel.OrTypeItemsElem, len(rawItems))
			for i, item := range rawItems {
				items[i] = item
			}

			return n.GetTypeInfo(&vscodemetamodel.OrType{
				Kind:  kind,
				Items: items,
			})
		case "tuple":
			rawItems, ok := raw["items"].([]interface{})
			if !ok {
				return TypeInfo{}, errors.Errorf("tuple type has no items field: %v", raw)
			}

			// Get type info for each item
			var itemTypes []string
			var itemDocs []string
			for i, item := range rawItems {
				itemInfo, err := n.GetTypeInfo(item)
				if err != nil {
					return TypeInfo{}, errors.Errorf("getting type info for tuple item %d: %w", i, err)
				}
				itemTypes = append(itemTypes, itemInfo.GoType)
				itemDocs = append(itemDocs, itemInfo.Documentation)
			}

			// Create a struct type for the tuple
			return TypeInfo{
				Name:          fmt.Sprintf("Tuple%d", len(itemTypes)),
				GoType:        fmt.Sprintf("struct{ %s }", strings.Join(itemTypes, ", ")),
				IsPointer:     false,
				IsBuiltin:     false,
				IsNullable:    false,
				Documentation: fmt.Sprintf("Tuple of: %s", strings.Join(itemDocs, ", ")),
			}, nil
		case "stringLiteral":
			value, ok := raw["value"].(string)
			if !ok {
				return TypeInfo{}, errors.Errorf("string literal has no value field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.StringLiteralType{
				Kind:  "stringLiteral",
				Value: value,
			})
		case "literal":
			// For literal types with complex values, treat them as empty structs
			return TypeInfo{
				Name:          "EmptyStruct",
				GoType:        "struct{}",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Empty struct literal",
			}, nil
		case "integerLiteral":
			value, ok := raw["value"].(float64)
			if !ok {
				return TypeInfo{}, errors.Errorf("integer literal has no value field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.IntegerLiteralType{
				Kind:  kind,
				Value: value,
			})
		case "booleanLiteral":
			value, ok := raw["value"].(bool)
			if !ok {
				return TypeInfo{}, errors.Errorf("boolean literal has no value field: %v", raw)
			}
			return n.GetTypeInfo(&vscodemetamodel.BooleanLiteralType{
				Kind:  kind,
				Value: value,
			})
		default:
			return TypeInfo{}, errors.Errorf("unsupported type kind: %s", kind)
		}
	}

	switch v := t.(type) {
	case *vscodemetamodel.StringLiteralType:
		info = TypeInfo{
			Name:          "String",
			GoType:        "string",
			IsPointer:     true,
			IsBuiltin:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("String literal with value %q", v.Value),
		}

	case *vscodemetamodel.IntegerLiteralType:
		info = TypeInfo{
			Name:          "Int",
			GoType:        "int",
			IsPointer:     true,
			IsBuiltin:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Integer literal with value %.0f", v.Value),
		}

	case *vscodemetamodel.BooleanLiteralType:
		info = TypeInfo{
			Name:          "Bool",
			GoType:        "bool",
			IsPointer:     true,
			IsBuiltin:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Boolean literal with value %v", v.Value),
		}

	case *vscodemetamodel.BaseType:
		switch v.Name {
		case vscodemetamodel.BaseTypesString:
			info = TypeInfo{
				Name:          "FoldingRangeKind", // Special case for this type
				GoType:        "string",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "String type",
			}
		case vscodemetamodel.BaseTypesInteger:
			info = TypeInfo{
				Name:          "Int",
				GoType:        "int",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Integer type",
			}
		case vscodemetamodel.BaseTypesBoolean:
			info = TypeInfo{
				Name:          "Bool",
				GoType:        "bool",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Boolean type",
			}
		case vscodemetamodel.BaseTypesDecimal:
			info = TypeInfo{
				Name:          "Float",
				GoType:        "float64",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Floating point type",
			}
		case vscodemetamodel.BaseTypesUinteger:
			// For uinteger types, we need to check if this is an enum
			if raw, ok := t.(map[string]interface{}); ok {
				if name, ok := raw["name"].(string); ok {
					switch name {
					case "PrepareSupportDefaultBehavior":
						info = TypeInfo{
							Name:          "PrepareSupportDefaultBehavior",
							GoType:        "uint32",
							IsPointer:     false,
							IsBuiltin:     true,
							IsNullable:    false,
							Documentation: "Unsigned integer type",
						}
						break
					case "InlineCompletionTriggerKind":
						info = TypeInfo{
							Name:          "InlineCompletionTriggerKind",
							GoType:        "uint32",
							IsPointer:     false,
							IsBuiltin:     true,
							IsNullable:    false,
							Documentation: "Unsigned integer type",
						}
						break
					case "DiagnosticSeverity":
						info = TypeInfo{
							Name:          "DiagnosticSeverity",
							GoType:        "uint32",
							IsPointer:     false,
							IsBuiltin:     true,
							IsNullable:    false,
							Documentation: "Unsigned integer type",
						}
						break
					case "CodeActionTriggerKind":
						info = TypeInfo{
							Name:          "CodeActionTriggerKind",
							GoType:        "uint32",
							IsPointer:     false,
							IsBuiltin:     true,
							IsNullable:    false,
							Documentation: "Unsigned integer type",
						}
						break
					}
				}
			}
			// Default to Uint if not an enum
			if info.Name == "" {
				info = TypeInfo{
					Name:          "Uint",
					GoType:        "uint32",
					IsPointer:     false,
					IsBuiltin:     true,
					IsNullable:    false,
					Documentation: "Unsigned integer type",
				}
			}
		case vscodemetamodel.BaseTypesNull:
			info = TypeInfo{
				Name:          "Null",
				GoType:        "bool",
				IsPointer:     true,
				IsBuiltin:     true,
				IsNullable:    true,
				Documentation: "Null type",
			}
		case vscodemetamodel.BaseTypesDocumentUri:
			info = TypeInfo{
				Name:          "DocumentURI",
				GoType:        "string",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Document URI type (string)",
			}
		case vscodemetamodel.BaseTypesURI:
			info = TypeInfo{
				Name:          "URI",
				GoType:        "string",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "URI type (string)",
			}
		case vscodemetamodel.BaseTypesRegExp:
			info = TypeInfo{
				Name:          "RegExp",
				GoType:        "string",
				IsPointer:     false,
				IsBuiltin:     true,
				IsNullable:    false,
				Documentation: "Regular expression type (string)",
			}
		default:
			return TypeInfo{}, errors.Errorf("unsupported base type %q", v.Name)
		}

	case *vscodemetamodel.ReferenceType:
		// Handle recursive types
		info = TypeInfo{
			Name:          v.Name,
			GoType:        v.Name,
			IsPointer:     true,
			IsBuiltin:     false,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Reference to %s", v.Name),
			Dependencies:  []string{v.Name},
			IsRecursive:   true,
		}

	case *vscodemetamodel.ArrayType:
		elemInfo, err := n.GetTypeInfo(v.Element)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting array element type: %w", err)
		}

		// Handle recursive array types
		if elemInfo.IsRecursive && elemInfo.Name == elemInfo.Dependencies[0] {
			info = TypeInfo{
				Name:          fmt.Sprintf("[]%s", elemInfo.Name),
				GoType:        fmt.Sprintf("[]%s", elemInfo.Name),
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: fmt.Sprintf("Array of %s", strings.ToLower(elemInfo.Name)),
				Dependencies:  []string{elemInfo.Name},
				IsRecursive:   true,
			}
		} else {
			info = TypeInfo{
				Name:          fmt.Sprintf("[]%s", elemInfo.GoType),
				GoType:        fmt.Sprintf("[]%s", elemInfo.GoType),
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: fmt.Sprintf("Array of %s", strings.ToLower(elemInfo.Name)),
				Dependencies:  []string{elemInfo.Name},
			}
		}

	case *vscodemetamodel.MapType:
		keyInfo, err := n.GetTypeInfo(v.Key)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting map key type: %w", err)
		}
		valueInfo, err := n.GetTypeInfo(v.Value)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting map value type: %w", err)
		}

		// Handle recursive map types
		if valueInfo.IsRecursive && valueInfo.Name == valueInfo.Dependencies[0] {
			info = TypeInfo{
				Name:          fmt.Sprintf("%sTo%sMap", keyInfo.Name, valueInfo.Name),
				GoType:        fmt.Sprintf("map[%s]%s", keyInfo.GoType, valueInfo.Name),
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: fmt.Sprintf("Map from %s to %s", strings.ToLower(keyInfo.Name), strings.ToLower(valueInfo.Name)),
				Dependencies:  []string{keyInfo.Name, valueInfo.Name},
				IsRecursive:   true,
			}
		} else {
			info = TypeInfo{
				Name:          fmt.Sprintf("%sTo%sMap", keyInfo.Name, valueInfo.Name),
				GoType:        fmt.Sprintf("map[%s]%s", keyInfo.GoType, valueInfo.GoType),
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: fmt.Sprintf("Map from %s to %s", strings.ToLower(keyInfo.Name), strings.ToLower(valueInfo.Name)),
				Dependencies:  []string{keyInfo.Name, valueInfo.Name},
			}
		}

	case *vscodemetamodel.OrType:
		// Flatten nested unions and collect all types
		var allTypes []vscodemetamodel.OrTypeItemsElem
		var err error
		allTypes, err = n.flattenUnion(v)
		if err != nil {
			return TypeInfo{}, errors.Errorf("flattening union: %w", err)
		}

		// Get type info for all items
		var deps []string
		var parts []string
		isRecursive := false
		for _, item := range allTypes {
			itemInfo, err := n.GetTypeInfo(item)
			if err != nil {
				return TypeInfo{}, errors.Errorf("getting union item type: %w", err)
			}
			deps = append(deps, itemInfo.Name)
			parts = append(parts, strings.ToLower(itemInfo.Name))
			if itemInfo.IsRecursive {
				isRecursive = true
			}
		}

		// Generate name and create info
		name := strings.Join(deps, "Or")
		info = TypeInfo{
			Name:          name,
			GoType:        name,
			IsPointer:     true,
			IsBuiltin:     false,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Union type of: %s", strings.Join(parts, ", ")),
			Dependencies:  deps,
			IsRecursive:   isRecursive,
		}

	default:
		return TypeInfo{}, errors.Errorf("unsupported type %T", t)
	}

	// Store the type info
	n.knownTypes[info.Name] = info
	return info, nil
}

// flattenUnion recursively flattens nested unions into a single list of types
func (n *TypeNamer) flattenUnion(union *vscodemetamodel.OrType) ([]vscodemetamodel.OrTypeItemsElem, error) {
	var result []vscodemetamodel.OrTypeItemsElem
	for _, item := range union.Items {
		if nested, ok := item.(*vscodemetamodel.OrType); ok {
			flattened, err := n.flattenUnion(nested)
			if err != nil {
				return nil, errors.Errorf("flattening nested union: %w", err)
			}
			result = append(result, flattened...)
		} else {
			result = append(result, item)
		}
	}
	return result, nil
}

// GenerateUnionTypeName generates a name for a union type based on its members
func (n *TypeNamer) GenerateUnionTypeName(union *vscodemetamodel.OrType) (string, error) {
	// Flatten nested unions first
	allTypes, err := n.flattenUnion(union)
	if err != nil {
		return "", errors.Errorf("flattening union: %w", err)
	}

	var parts []string
	for _, item := range allTypes {
		info, err := n.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info: %w", err)
		}
		parts = append(parts, info.Name)
	}

	// Create a base name from the parts
	baseName := strings.Join(parts, "Or")
	if baseName == "" {
		baseName = "Empty"
	}

	// Add a suffix if we've seen this name before
	n.typeCount[baseName]++
	if n.typeCount[baseName] > 1 {
		baseName = fmt.Sprintf("%s%d", baseName, n.typeCount[baseName])
	}

	return baseName, nil
}

// GetDependencyGraph returns a map of type names to their dependencies
func (n *TypeNamer) GetDependencyGraph() map[string][]string {
	deps := make(map[string][]string)
	for name, info := range n.knownTypes {
		deps[name] = info.Dependencies
	}
	return deps
}

// GetDocumentation returns the documentation for a type
func (n *TypeNamer) GetDocumentation(typeName string) string {
	if info, ok := n.knownTypes[typeName]; ok {
		return info.Documentation
	}
	return ""
}
