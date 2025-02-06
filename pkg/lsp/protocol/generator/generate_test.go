package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
)

func TestGetTypeInfo_ImplementationRequest(t *testing.T) {
	// This test case is based on a real example from the LSP protocol
	// See: textDocument/implementation request in metaModel.json

	tests := []struct {
		name     string
		input    interface{}
		expected *TypeInfo
		wantErr  bool
	}{
		{
			name: "implementation_request_result",
			input: &vscodemetamodel.OrType{
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
			expected: &TypeInfo{
				GoType:    "[]interface{}",
				IsPointer: false,
			},
			wantErr: false,
		},
		{
			name: "implementation_request_partial_result",
			input: &vscodemetamodel.OrType{
				Kind: "or",
				Items: []vscodemetamodel.OrTypeItemsElem{
					&vscodemetamodel.ArrayType{
						Kind: "array",
						Element: &vscodemetamodel.ReferenceType{
							Kind: "reference",
							Name: "Location",
						},
					},
					&vscodemetamodel.ArrayType{
						Kind: "array",
						Element: &vscodemetamodel.ReferenceType{
							Kind: "reference",
							Name: "DefinitionLink",
						},
					},
				},
			},
			expected: &TypeInfo{
				GoType:    "[]interface{}",
				IsPointer: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTypeInfo(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
