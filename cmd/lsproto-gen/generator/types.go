package generator

import (
	"context"
	"fmt"

	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"gitlab.com/tozd/go/errors"
)

// Generator handles the generation of LSP protocol code
type Generator struct {
	model *vscodemetamodel.MetaModel
	namer *TypeNamer
}

// NewGenerator creates a new LSP protocol code generator
func NewGenerator(model *vscodemetamodel.MetaModel) *Generator {
	return &Generator{
		model: model,
		namer: NewTypeNamer(),
	}
}

// GenerateUnionType generates a Go type for a union type
func (g *Generator) GenerateUnionType(ctx context.Context, union *vscodemetamodel.OrType) (string, error) {
	// Generate type name
	typeName, err := g.namer.GenerateUnionTypeName(union)
	if err != nil {
		return "", errors.Errorf("generating type name: %w", err)
	}

	// Start with type definition
	code := fmt.Sprintf("// %s represents a union of multiple types\n", typeName)
	code += fmt.Sprintf("type %s struct {\n", typeName)

	// Add fields for each type in the union
	for i, item := range union.Items {
		info, err := g.namer.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info for field %d: %w", i, err)
		}

		// Add field with comment
		code += fmt.Sprintf("\t%sValue *%s // Option %c\n",
			info.Name, info.GoType, rune('A'+i))
	}

	// Close type definition
	code += "}\n\n"

	// Add validation method
	code += g.generateValidationMethod(typeName, union)

	// Add marshal method
	code += g.generateMarshalMethod(typeName, union)

	// Add unmarshal method
	code += g.generateUnmarshalMethod(typeName, union)

	return code, nil
}

// generateValidationMethod generates the Validate method for the union type
func (g *Generator) generateValidationMethod(typeName string, union *vscodemetamodel.OrType) string {
	code := fmt.Sprintf("// Validate ensures exactly one field is set\n")
	code += fmt.Sprintf("func (t *%s) Validate() error {\n", typeName)
	code += "\tcount := 0\n"

	// Add count checks for each field
	for _, item := range union.Items {
		info, _ := g.namer.GetTypeInfo(item) // Error already checked in parent
		code += fmt.Sprintf("\tif t.%sValue != nil { count++ }\n", info.Name)
	}

	code += "\tif count != 1 {\n"
	code += "\t\treturn errors.New(\"exactly one field must be set\")\n"
	code += "\t}\n"
	code += "\treturn nil\n"
	code += "}\n\n"

	return code
}

// generateMarshalMethod generates the MarshalJSON method for the union type
func (g *Generator) generateMarshalMethod(typeName string, union *vscodemetamodel.OrType) string {
	code := fmt.Sprintf("// MarshalJSON implements json.Marshaler\n")
	code += fmt.Sprintf("func (t %s) MarshalJSON() ([]byte, error) {\n", typeName)
	code += "\tif err := t.Validate(); err != nil {\n"
	code += "\t\treturn nil, err\n"
	code += "\t}\n\n"

	// Add marshal logic for each field
	for _, item := range union.Items {
		info, _ := g.namer.GetTypeInfo(item) // Error already checked in parent
		code += fmt.Sprintf("\tif t.%sValue != nil { return json.Marshal(*t.%sValue) }\n",
			info.Name, info.Name)
	}

	code += "\treturn nil, errors.New(\"no field set\")\n"
	code += "}\n\n"

	return code
}

// generateUnmarshalMethod generates the UnmarshalJSON method for the union type
func (g *Generator) generateUnmarshalMethod(typeName string, union *vscodemetamodel.OrType) string {
	code := fmt.Sprintf("// UnmarshalJSON implements json.Unmarshaler\n")
	code += fmt.Sprintf("func (t *%s) UnmarshalJSON(data []byte) error {\n", typeName)

	// Add unmarshal logic for each field
	for _, item := range union.Items {
		info, _ := g.namer.GetTypeInfo(item) // Error already checked in parent

		// Special case for null
		if info.Name == "Null" {
			code += "\t// Try null\n"
			code += "\tif string(data) == \"null\" {\n"
			code += "\t\tt.NullValue = new(bool)\n"
			code += "\t\t*t.NullValue = true\n"
			code += "\t\treturn nil\n"
			code += "\t}\n\n"
			continue
		}

		code += fmt.Sprintf("\t// Try %s\n", info.Name)
		code += fmt.Sprintf("\tvar %c %s\n", 'v', info.GoType)
		code += fmt.Sprintf("\tif err := json.Unmarshal(data, &%c); err == nil {\n", 'v')
		code += fmt.Sprintf("\t\tt.%sValue = &%c\n", info.Name, 'v')
		code += "\t\treturn nil\n"
		code += "\t}\n\n"
	}

	code += "\treturn errors.New(\"data matches no expected type\")\n"
	code += "}\n\n"

	return code
}
