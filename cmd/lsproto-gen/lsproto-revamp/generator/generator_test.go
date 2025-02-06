package generator

import (
	"context"
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"github.com/walteh/gotmpls/pkg/diff"
)

// 🧪 Test Plan:
// 1. Test basic request type generation
//    - Input: Implementation request from metaModel.json
//    - Expected: Properly structured Go types with embedding
//
// 2. Test union type generation
//    - Input: Definition|[]DefinitionLink|null union
//    - Expected: Clean struct with proper JSON marshaling
//
// 3. Test full file generation
//    - Input: Complete model with multiple types
//    - Expected: Properly organized file with all types

func strPtr(s string) *string { return &s }

func TestGenerateImplementationRequest(t *testing.T) {
	// 📝 Test case based on the textDocument/implementation request
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

		Structures: []vscodemetamodel.Structure{{
			Name: "ImplementationParams",
			// TODO(dr.methodical): add fields
		}},
	}

	// Create generator
	gen := NewGenerator(*model, "lsproto")

	// Generate code
	files, err := gen.GenerateFiles(context.Background(), "lsproto")
	require.NoError(t, err, "generating files should not fail")
	require.NotEmpty(t, files, "should generate at least one file")

	// Find the types file
	var typesFile *File
	for _, file := range files {
		if strings.HasSuffix(file.Path, "types.go") {
			typesFile = &file
			break
		}
	}
	require.NotNil(t, typesFile, "should generate types.go file")

	expected := `
package lsproto

type ImplementationRequest struct {
	ImplementationParams
}

type ImplementationResultOrs struct {
	Definition      *Definition
	DefinitionLinks []DefinitionLink
	IsNull         bool
}

func (r ImplementationResultOrs) MarshalJSON() ([]byte, error) {
	if r.IsNull {
		return json.Marshal(nil)
	}

	if r.Definition != nil {
		return json.Marshal(r.Definition)
	}

	if r.DefinitionLinks != nil {
		return json.Marshal(r.DefinitionLinks)
	}

	return nil, errors.New("invalid implementation result")
}

func (r *ImplementationResultOrs) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		r.IsNull = true
		return nil
	}

	if err := json.Unmarshal(data, &r.Definition); err == nil {
		return nil
	}

	if err := json.Unmarshal(data, &r.DefinitionLinks); err == nil {
		return nil
	}

	return errors.New("invalid implementation result")
}`

	want, err := format.Source([]byte(expected))
	require.NoError(t, err, "formatting expected code should not fail")

	got, err := format.Source([]byte(typesFile.Contents))
	require.NoError(t, err, "formatting generated code should not fail")

	diff.RequireKnownValueEqual(t, want, got)

}
