package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func TestTypeNamer(t *testing.T) {
	testCases := []struct {
		name        string
		input       interface{}
		wantInfo    TypeInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "string_literal",
			input: &vscodemetamodel.StringLiteralType{
				Kind:  "stringLiteral",
				Value: "test",
			},
			wantInfo: TypeInfo{
				Name:          "String",
				GoType:        "string",
				IsPointer:     true,
				IsBuiltin:     true,
				IsNullable:    true,
				Documentation: `String literal with value "test"`,
				Dependencies:  nil,
			},
		},
		{
			name: "array_of_strings",
			input: &vscodemetamodel.ArrayType{
				Kind: "array",
				Element: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesString,
				},
			},
			wantInfo: TypeInfo{
				Name:          "StringArray",
				GoType:        "[]string",
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: "Array of string",
				Dependencies:  []string{"String"},
			},
		},
		{
			name: "map_string_to_int",
			input: &vscodemetamodel.MapType{
				Kind: "map",
				Key: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesString,
				},
				Value: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesInteger,
				},
			},
			wantInfo: TypeInfo{
				Name:          "StringToIntMap",
				GoType:        "map[string]int",
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: "Map from string to int",
				Dependencies:  []string{"String", "Int"},
			},
		},
		{
			name: "nested_union",
			input: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.StringLiteralType{
						Kind:  "stringLiteral",
						Value: "test",
					},
					&vscodemetamodel.OrType{
						Kind: "or",
						Items: []vscodemetamodel.OrTypeItemsElem{
							&vscodemetamodel.IntegerLiteralType{
								Kind:  "integerLiteral",
								Value: 42,
							},
							&vscodemetamodel.BooleanLiteralType{
								Kind:  "booleanLiteral",
								Value: true,
							},
						},
					},
				},
			},
			wantInfo: TypeInfo{
				Name:          "StringOrIntOrBool",
				GoType:        "StringOrIntOrBool",
				IsPointer:     true,
				IsBuiltin:     false,
				IsNullable:    true,
				Documentation: "Union type of: string, int, bool",
				Dependencies:  []string{"String", "Int", "Bool"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namer := NewTypeNamer()
			info, err := namer.GetTypeInfo(tc.input)

			if tc.wantErr {
				require.Error(t, err, "expected error")
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains,
						"error should contain expected message")
				}
				return
			}

			require.NoError(t, err, "getting type info should not fail")
			assert.Equal(t, tc.wantInfo, info, "type info should match")
		})
	}
}

func TestGenerateUnionTypeName(t *testing.T) {
	testCases := []struct {
		name        string
		union       *vscodemetamodel.OrType
		wantName    string
		wantErr     bool
		errContains string
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
			wantName: "StringOrInt",
		},
		{
			name: "array_or_map",
			union: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.ArrayType{
						Kind: "array",
						Element: &vscodemetamodel.BaseType{
							Kind: "base",
							Name: vscodemetamodel.BaseTypesString,
						},
					},
					&vscodemetamodel.MapType{
						Kind: "map",
						Key: &vscodemetamodel.BaseType{
							Kind: "base",
							Name: vscodemetamodel.BaseTypesString,
						},
						Value: &vscodemetamodel.BaseType{
							Kind: "base",
							Name: vscodemetamodel.BaseTypesInteger,
						},
					},
				},
			},
			wantName: "StringArrayOrStringToIntMap",
		},
		{
			name: "nested_union",
			union: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.StringLiteralType{
						Kind:  "stringLiteral",
						Value: "test",
					},
					&vscodemetamodel.OrType{
						Kind: "or",
						Items: []vscodemetamodel.OrTypeItemsElem{
							&vscodemetamodel.IntegerLiteralType{
								Kind:  "integerLiteral",
								Value: 42,
							},
							&vscodemetamodel.BooleanLiteralType{
								Kind:  "booleanLiteral",
								Value: true,
							},
						},
					},
				},
			},
			wantName: "StringOrIntOrBool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namer := NewTypeNamer()
			name, err := namer.GenerateUnionTypeName(tc.union)

			if tc.wantErr {
				require.Error(t, err, "expected error")
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains,
						"error should contain expected message")
				}
				return
			}

			require.NoError(t, err, "generating union type name should not fail")
			assert.Equal(t, tc.wantName, name, "type name should match")
		})
	}
}

func TestDependencyGraph(t *testing.T) {
	namer := NewTypeNamer()

	// Create some types with dependencies
	union := &vscodemetamodel.OrType{
		Kind: "or",
		Items: []vscodemetamodel.OrTypeItemsElem{
			&vscodemetamodel.ArrayType{
				Kind: "array",
				Element: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesString,
				},
			},
			&vscodemetamodel.MapType{
				Kind: "map",
				Key: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesString,
				},
				Value: &vscodemetamodel.BaseType{
					Kind: "base",
					Name: vscodemetamodel.BaseTypesInteger,
				},
			},
		},
	}

	// Get type info to populate known types
	info, err := namer.GetTypeInfo(union)
	require.NoError(t, err, "getting type info should not fail")

	// Get dependency graph
	graph := namer.GetDependencyGraph()

	// Check dependencies
	assert.Contains(t, graph, info.Name, "graph should contain union type")
	assert.ElementsMatch(t, []string{"StringArray", "StringToIntMap"}, graph[info.Name],
		"union type should depend on array and map types")
}

func TestDocumentation(t *testing.T) {
	namer := NewTypeNamer()

	// Create a type with documentation
	union := &vscodemetamodel.OrType{
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
	}

	// Get type info to populate known types
	info, err := namer.GetTypeInfo(union)
	require.NoError(t, err, "getting type info should not fail")

	// Get documentation
	doc := namer.GetDocumentation(info.Name)
	assert.Equal(t, "Union type of: string, int", doc,
		"documentation should match expected format")
}

func TestGetTypeInfo_TupleType(t *testing.T) {
	// Create test cases
	testCases := []struct {
		name        string
		input       interface{}
		want        TypeInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "simple_tuple",
			input: &vscodemetamodel.TupleType{
				Kind: "tuple",
				Items: []vscodemetamodel.TupleTypeItemsElem{
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesString,
					},
					&vscodemetamodel.BaseType{
						Kind: "base",
						Name: vscodemetamodel.BaseTypesInteger,
					},
				},
			},
			want: TypeInfo{
				Name:          "TupleStringInt",
				GoType:        "TupleStringInt",
				IsBuiltin:     false,
				Dependencies:  []string{"String", "Int"},
				Documentation: "Tuple of string and integer",
			},
		},
		{
			name: "empty_tuple",
			input: &vscodemetamodel.TupleType{
				Kind:  "tuple",
				Items: []vscodemetamodel.TupleTypeItemsElem{},
			},
			want: TypeInfo{
				Name:          "TupleEmpty",
				GoType:        "TupleEmpty",
				IsBuiltin:     false,
				Dependencies:  []string{},
				Documentation: "Empty tuple",
			},
		},
		{
			name:        "nil_tuple",
			input:       (*vscodemetamodel.TupleType)(nil),
			wantErr:     true,
			errContains: "type is nil",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new type namer for each test
			namer := NewTypeNamer()

			// Get type info
			got, err := namer.GetTypeInfo(tc.input)

			// Check error cases
			if tc.wantErr {
				require.Error(t, err, "expected error for case %s", tc.name)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains,
						"error should contain expected message for case %s", tc.name)
				}
				return
			}

			// Check success cases
			require.NoError(t, err, "unexpected error for case %s", tc.name)
			assert.Equal(t, tc.want.Name, got.Name,
				"name should match for case %s", tc.name)
			assert.Equal(t, tc.want.GoType, got.GoType,
				"go type should match for case %s", tc.name)
			assert.Equal(t, tc.want.IsBuiltin, got.IsBuiltin,
				"is builtin should match for case %s", tc.name)
			assert.ElementsMatch(t, tc.want.Dependencies, got.Dependencies,
				"dependencies should match for case %s", tc.name)
		})
	}
}
