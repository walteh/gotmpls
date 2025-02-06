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

// GenerateRequestType generates Go code for a request type
func (g *Generator) GenerateRequestType(_ context.Context, req *vscodemetamodel.Request) (string, error) {
	if req == nil {
		return "", errors.New("request is nil")
	}

	// Get type info for params and result
	var paramsInfo, resultInfo TypeInfo
	var err error

	if req.Params != nil {
		paramsInfo, err = g.namer.GetTypeInfo(req.Params)
		if err != nil {
			return "", errors.Errorf("getting params type info: %w", err)
		}
	}

	if req.Result != nil {
		resultInfo, err = g.namer.GetTypeInfo(req.Result)
		if err != nil {
			return "", errors.Errorf("getting result type info: %w", err)
		}
	}

	// Build the type definition
	var code string

	// Add documentation
	if req.Documentation != nil {
		code += fmt.Sprintf("// %s\n", *req.Documentation)
	}

	// Add request type
	code += fmt.Sprintf("type %s struct {\n", *req.TypeName)

	// Add params field if present
	if req.Params != nil {
		code += fmt.Sprintf("\tParams %s `json:\"params\"`\n", paramsInfo.GoType)
	}

	// Add result field if present
	if req.Result != nil {
		code += fmt.Sprintf("\tResult %s `json:\"result,omitempty\"`\n", resultInfo.GoType)
	}

	code += "}\n\n"

	// Add method constant
	code += fmt.Sprintf("const %sMethod = %q\n", *req.TypeName, req.Method)

	return code, nil
}

// GenerateUnionType generates a Go type for a union type
func (g *Generator) GenerateUnionType(ctx context.Context, union *vscodemetamodel.OrType) (string, error) {
	// Get type info for the union
	info, err := g.namer.GetTypeInfo(union)
	if err != nil {
		return "", errors.Errorf("getting union type info: %w", err)
	}

	// Start with type definition
	code := fmt.Sprintf("// %s represents a union type\n", info.Name)
	code += fmt.Sprintf("type %s struct {\n", info.Name)

	// Add fields for each type in the union
	for i, item := range union.Items {
		itemInfo, err := g.namer.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info for field %d: %w", i, err)
		}

		// Add field with comment
		code += fmt.Sprintf("\t%sValue *%s `json:\"%s,omitempty\"` // Option %c\n",
			itemInfo.Name, itemInfo.GoType, itemInfo.Name, rune('A'+i))
	}

	// Close type definition
	code += "}\n\n"

	// Add validation method
	code += fmt.Sprintf(`// Validate ensures exactly one field is set
func (t *%s) Validate() error {
	count := 0
`, info.Name)

	for _, item := range union.Items {
		itemInfo, err := g.namer.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info for validation: %w", err)
		}
		code += fmt.Sprintf("\tif t.%sValue != nil { count++ }\n", itemInfo.Name)
	}

	code += `	if count != 1 {
		return errors.New("exactly one field must be set")
	}
	return nil
}

`

	// Add marshal method
	code += fmt.Sprintf(`// MarshalJSON implements json.Marshaler
func (t %s) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

`, info.Name)

	for _, item := range union.Items {
		itemInfo, err := g.namer.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info for marshaling: %w", err)
		}
		code += fmt.Sprintf("\tif t.%sValue != nil { return json.Marshal(*t.%sValue) }\n",
			itemInfo.Name, itemInfo.Name)
	}

	code += `	return nil, errors.New("no field set")
}

`

	// Add unmarshal method
	code += fmt.Sprintf(`// UnmarshalJSON implements json.Unmarshaler
func (t *%s) UnmarshalJSON(data []byte) error {
`, info.Name)

	for _, item := range union.Items {
		itemInfo, err := g.namer.GetTypeInfo(item)
		if err != nil {
			return "", errors.Errorf("getting type info for unmarshaling: %w", err)
		}

		code += fmt.Sprintf(`	// Try %s
	var v %s
	if err := json.Unmarshal(data, &v); err == nil {
		t.%sValue = &v
		return nil
	}

`, itemInfo.Name, itemInfo.GoType, itemInfo.Name)
	}

	code += `	return errors.New("data matches no expected type")
}
`

	return code, nil
}
