package generator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func TestGenerateUnionType(t *testing.T) {
	// Test cases
	testCases := []struct {
		name            string
		union           *vscodemetamodel.OrType
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "string_or_int",
			union: &vscodemetamodel.OrType{
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
			wantContains: []string{
				"type StringOrInt struct",
				"StringValue *string",
				"IntValue *int",
				"func (t *StringOrInt) Validate() error",
				"func (t StringOrInt) MarshalJSON() ([]byte, error)",
				"func (t *StringOrInt) UnmarshalJSON(data []byte) error",
			},
			wantNotContains: []string{
				"BoolValue",  // Should not have bool field
				"FloatValue", // Should not have float field
			},
		},
		{
			name: "all_base_types",
			union: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesString,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesInteger,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesBoolean,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesDecimal,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesUinteger,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesNull,
					},
				},
			},
			wantContains: []string{
				"StringValue *string",
				"IntValue *int",
				"BoolValue *bool",
				"FloatValue *float64",
				"UintValue *uint",
				"NullValue *bool",
				"if string(data) == \"null\"",
			},
			wantNotContains: []string{
				"RegExpValue", // Should not have regexp field
			},
		},
		{
			name: "mixed_types",
			union: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.StringLiteralType{
						Kind:  "stringLiteral",
						Value: "literal",
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesString,
					},
					&vscodemetamodel.BooleanLiteralType{
						Kind:  "booleanLiteral",
						Value: true,
					},
				},
			},
			wantContains: []string{
				"StringValue *string",
				"BoolValue *bool",
				"var v string",
				"var v bool",
			},
			wantNotContains: []string{
				"IntValue",   // Should not have int field
				"FloatValue", // Should not have float field
			},
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create generator
			gen := NewGenerator(nil) // Model not needed for this test

			// Generate code
			code, err := gen.GenerateUnionType(context.Background(), tc.union)
			require.NoError(t, err, "generating union type should not fail")

			// Check contains
			for _, want := range tc.wantContains {
				assert.True(t, strings.Contains(code, want),
					"generated code should contain %q\nCode:\n%s", want, code)
			}

			// Check not contains
			for _, notWant := range tc.wantNotContains {
				assert.False(t, strings.Contains(code, notWant),
					"generated code should not contain %q\nCode:\n%s", notWant, code)
			}

			// TODO(lsproto): 🧪 Add more test cases:
			// 1. Invalid union types
			// 2. Empty unions
			// 3. Complex nested types
			// 4. All primitive types
		})
	}
}
