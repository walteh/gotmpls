package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func TestUnionTypeHandling(t *testing.T) {
	// TODO(lsproto): 🧪 This test will be expanded to test our generated code
	// For now it just validates we can create and marshal a union type

	testCases := []struct {
		name      string
		union     vscodemetamodel.OrType
		wantKinds []string
	}{
		{
			name: "string_or_int",
			union: vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.StringLiteralType{
						Kind:  "stringLiteral",
						Value: "test",
					},
					&vscodemetamodel.IntegerLiteralType{
						Kind:  "integerLiteral",
						Value: 42,
					},
				},
			},
			wantKinds: []string{"stringLiteral", "integerLiteral"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate we can create the type
			assert.Equal(t, "or", tc.union.Kind)
			assert.Len(t, tc.union.Items, len(tc.wantKinds))

			// Check each item's kind
			for i, item := range tc.union.Items {
				switch v := item.(type) {
				case *vscodemetamodel.StringLiteralType:
					assert.Equal(t, tc.wantKinds[i], v.Kind)
				case *vscodemetamodel.IntegerLiteralType:
					assert.Equal(t, tc.wantKinds[i], v.Kind)
				default:
					t.Errorf("unexpected type %T", v)
				}
			}

			// TODO(lsproto): 🎯 Add tests for:
			// 1. Generated type marshaling
			// 2. Generated type unmarshaling
			// 3. Validation logic
			// 4. Error cases
		})
	}
}
