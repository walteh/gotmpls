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

	// Handle vscodemetamodel types
	switch t := t.(type) {
	case *vscodemetamodel.BaseType:
		info := TypeInfo{
			Name:          string(t.Name),
			GoType:        n.getBaseType(string(t.Name)),
			IsBuiltin:     true,
			IsPointer:     false,
			IsNullable:    false,
			Documentation: fmt.Sprintf("%s type", t.Name),
		}
		n.knownTypes[info.Name] = info
		return info, nil

	case *vscodemetamodel.ArrayType:
		elemInfo, err := n.GetTypeInfo(t.Element)
		if err != nil {
			return TypeInfo{}, errors.Errorf("getting array element type: %w", err)
		}
		info := TypeInfo{
			Name:          fmt.Sprintf("%sArray", elemInfo.Name),
			GoType:        fmt.Sprintf("[]%s", elemInfo.GoType),
			IsBuiltin:     false,
			IsPointer:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Array of %s", strings.ToLower(elemInfo.Name)),
			Dependencies:  []string{elemInfo.Name},
		}
		n.knownTypes[info.Name] = info
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
			Name:          fmt.Sprintf("%sTo%sMap", keyInfo.Name, valueInfo.Name),
			GoType:        fmt.Sprintf("map[%s]%s", keyInfo.GoType, valueInfo.GoType),
			IsBuiltin:     false,
			IsPointer:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Map from %s to %s", strings.ToLower(keyInfo.Name), strings.ToLower(valueInfo.Name)),
			Dependencies:  []string{keyInfo.Name, valueInfo.Name},
		}
		n.knownTypes[info.Name] = info
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
		var docs []string
		for _, info := range itemInfos {
			names = append(names, info.Name)
			docs = append(docs, strings.ToLower(info.Name))
		}
		name := fmt.Sprintf("Or%s", strings.Join(names, ""))
		info := TypeInfo{
			Name:          name,
			GoType:        name,
			IsBuiltin:     false,
			IsPointer:     true,
			IsNullable:    true,
			IsUnion:       true,
			Documentation: fmt.Sprintf("Union type of: %s", strings.Join(docs, ", ")),
			Dependencies:  names,
		}
		n.knownTypes[info.Name] = info
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
		var docs []string
		for _, info := range itemInfos {
			names = append(names, info.Name)
			docs = append(docs, strings.ToLower(info.Name))
		}
		name := fmt.Sprintf("Tuple%s", strings.Join(names, ""))
		info := TypeInfo{
			Name:          name,
			GoType:        name,
			IsBuiltin:     false,
			IsPointer:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Tuple of: %s", strings.Join(docs, ", ")),
			Dependencies:  names,
		}
		n.knownTypes[info.Name] = info
		return info, nil

	case *vscodemetamodel.StringLiteralType:
		info := TypeInfo{
			Name:          fmt.Sprintf("StringLiteral%s", strings.Title(t.Value)),
			GoType:        "string",
			IsBuiltin:     true,
			IsPointer:     false,
			IsNullable:    false,
			Documentation: fmt.Sprintf("String literal with value %q", t.Value),
		}
		n.knownTypes[info.Name] = info
		return info, nil

	case *vscodemetamodel.IntegerLiteralType:
		name := fmt.Sprintf("IntegerLiteral%d", t.Value)
		info := TypeInfo{
			Name:          name,
			GoType:        "int32",
			IsBuiltin:     true,
			IsPointer:     false,
			IsNullable:    false,
			Documentation: fmt.Sprintf("Integer literal with value %d", t.Value),
		}
		n.knownTypes[info.Name] = info
		return info, nil

	case *vscodemetamodel.ReferenceType:
		info := TypeInfo{
			Name:          t.Name,
			GoType:        t.Name,
			IsBuiltin:     false,
			IsPointer:     true,
			IsNullable:    true,
			Documentation: fmt.Sprintf("Reference to %s", t.Name),
			Dependencies:  []string{t.Name},
			IsRecursive:   true,
		}
		n.knownTypes[info.Name] = info
		return info, nil

	default:
		return TypeInfo{}, errors.Errorf("unsupported type: %T", t)
	}
}

// GetDependencyGraph returns a map of type names to their dependencies
func (n *TypeNamer) GetDependencyGraph() map[string][]string {
	graph := make(map[string][]string)
	for name, info := range n.knownTypes {
		graph[name] = info.Dependencies
	}
	return graph
}

// GetDocumentation returns the documentation for a type
func (n *TypeNamer) GetDocumentation(typeName string) string {
	if info, ok := n.knownTypes[typeName]; ok {
		return info.Documentation
	}
	return ""
}
