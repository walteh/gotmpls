package generator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func strPtr(s string) *string {
	return &s
}

func TestGenerateImplementationRequest(t *testing.T) {
	// Create a simple model with just the implementation request
	model := &vscodemetamodel.MetaModel{
		Requests: []vscodemetamodel.Request{{
			Method:   "textDocument/implementation",
			TypeName: strPtr("ImplementationRequest"),
			Result: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.ReferenceType{
						Kind: "reference",
						Name: "Definition",
					},
					&vscodemetamodel.ArrayType{
						Kind: "array",
						Element: &vscodemetamodel.ReferenceType{
							Kind: "reference",
							Name: "DefinitionLink",
						},
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesNull,
					},
				},
			},
			Params: &vscodemetamodel.ReferenceType{
				Kind: "reference",
				Name: "ImplementationParams",
			},
			Documentation: strPtr("A request to resolve the implementation locations of a symbol at a given text document position."),
		}},
	}

	// Create generator
	gen := NewGenerator(model)

	// Generate code for the request
	code, err := gen.GenerateRequestType(context.Background(), &model.Requests[0])
	require.NoError(t, err, "generating request type should not fail")

	// Verify the generated code
	assert.Contains(t, code, "type ImplementationRequest struct",
		"should generate request type")
	assert.Contains(t, code, "Params ImplementationParams",
		"should include params field")
	assert.Contains(t, code, "Result *OrDefinitionDefinitionLinkNull",
		"should include result field with union type")
	assert.Contains(t, code, "const ImplementationRequestMethod =",
		"should generate method constant")
	assert.Contains(t, code, "A request to resolve the implementation",
		"should include documentation")
}
