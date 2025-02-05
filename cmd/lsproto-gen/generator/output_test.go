package generator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func TestFileGenerator(t *testing.T) {
	testCases := []struct {
		name            string
		model           *vscodemetamodel.MetaModel
		packageName     string
		setupTypes      func(*FileGenerator)
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:        "basic_types",
			model:       nil,
			packageName: "testpkg",
			setupTypes: func(gen *FileGenerator) {
				// Add a simple union type
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
				_, err := gen.namer.GetTypeInfo(union)
				require.NoError(t, err, "getting type info should not fail")
			},
			wantContains: []string{
				"package testpkg",
				"import (",
				"encoding/json",
				"gitlab.com/tozd/go/errors",
				"// Code generated by lsproto-gen. DO NOT EDIT.",
				"┌──────────────────────────────────────────────────────────────┐",
				"StringOrInt",
				"StringValue",
				"IntValue",
				"func (t *StringOrInt) Validate() error",
				"func (t StringOrInt) MarshalJSON() ([]byte, error)",
				"func (t *StringOrInt) UnmarshalJSON(data []byte) error",
			},
			wantNotContains: []string{
				"package main",
				"DO NOT MODIFY",
				"BoolValue",
			},
		},
		{
			name:        "recursive_types",
			model:       nil,
			packageName: "testpkg",
			setupTypes: func(gen *FileGenerator) {
				// Create a recursive type (like a binary tree)
				treeType := &vscodemetamodel.OrType{
					Kind: "or",
					Items: []vscodemetamodel.OrTypeItemsElem{
						&vscodemetamodel.StringLiteralType{
							Kind:  "stringLiteral",
							Value: "leaf",
						},
						&vscodemetamodel.MapType{
							Kind: "map",
							Key: &vscodemetamodel.StringLiteralType{
								Kind:  "stringLiteral",
								Value: "node",
							},
							Value: &vscodemetamodel.ArrayType{
								Kind: "array",
								Element: &vscodemetamodel.ReferenceType{
									Kind: "reference",
									Name: "Tree", // Reference to self
								},
							},
						},
					},
				}
				_, err := gen.namer.GetTypeInfo(treeType)
				require.NoError(t, err, "getting type info should not fail")
			},
			wantContains: []string{
				"// Reference to Tree",
				"type Tree struct {",
				"StringValue *string",                          // Leaf case
				"StringToTreeArrayMapValue *map[string][]Tree", // Node case with children
				"func (t *Tree) Validate() error",
				"func (t Tree) MarshalJSON() ([]byte, error)",
				"func (t *Tree) UnmarshalJSON(data []byte) error",
			},
			wantNotContains: []string{
				"IntValue",
				"BoolValue",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create generator
			gen := NewFileGenerator(tc.model)

			// Setup test types if needed
			if tc.setupTypes != nil {
				tc.setupTypes(gen)
			}

			// Generate files
			files, err := gen.GenerateFiles(context.Background(), tc.packageName)
			require.NoError(t, err, "generating files should not fail")
			require.NotEmpty(t, files, "should generate at least one file")

			// Check each file
			for _, file := range files {
				// Check path
				assert.NotEmpty(t, file.Path, "file path should not be empty")
				assert.True(t, strings.HasSuffix(file.Path, ".go"),
					"file should have .go extension")

				// Check contents
				for _, want := range tc.wantContains {
					assert.Contains(t, file.Contents, want,
						"file should contain %q", want)
				}
				for _, notWant := range tc.wantNotContains {
					assert.NotContains(t, file.Contents, notWant,
						"file should not contain %q", notWant)
				}
			}
		})
	}
}

func TestGenerateTypesFile(t *testing.T) {
	// Create a generator with some test types
	gen := NewFileGenerator(nil)

	// Add some test types
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
	_, err := gen.namer.GetTypeInfo(union)
	require.NoError(t, err, "getting type info should not fail")

	// Generate types file
	file, err := gen.generateTypesFile(context.Background(), "testpkg")
	require.NoError(t, err, "generating types file should not fail")

	// Check file contents
	assert.Equal(t, "types.go", file.Path, "file path should be types.go")
	assert.Contains(t, file.Contents, "package testpkg",
		"file should contain package declaration")
	assert.Contains(t, file.Contents, "// String literal with value",
		"file should contain type documentation")
	assert.Contains(t, file.Contents, "// Integer literal with value",
		"file should contain type documentation")
}

func TestSortedTypes(t *testing.T) {
	// Create a generator
	gen := NewFileGenerator(nil)

	// Create types with dependencies
	arrayType := &vscodemetamodel.ArrayType{
		Kind: "array",
		Element: &vscodemetamodel.BaseType{
			Kind: "base",
			Name: vscodemetamodel.BaseTypesString,
		},
	}

	mapType := &vscodemetamodel.MapType{
		Kind: "map",
		Key: &vscodemetamodel.BaseType{
			Kind: "base",
			Name: vscodemetamodel.BaseTypesString,
		},
		Value: &vscodemetamodel.BaseType{
			Kind: "base",
			Name: vscodemetamodel.BaseTypesInteger,
		},
	}

	union := &vscodemetamodel.OrType{
		Kind: "or",
		Items: []vscodemetamodel.OrTypeItemsElem{
			arrayType,
			mapType,
		},
	}

	// Get type info to populate known types
	_, err := gen.namer.GetTypeInfo(union)
	require.NoError(t, err, "getting type info should not fail")

	// Get sorted types
	types, err := gen.getSortedTypes()
	require.NoError(t, err, "getting sorted types should not fail")

	// Check order - types with fewer dependencies should come first
	for i := 1; i < len(types); i++ {
		assert.GreaterOrEqual(t,
			len(types[i].Dependencies),
			len(types[i-1].Dependencies),
			"types should be sorted by dependency count")
	}
}
