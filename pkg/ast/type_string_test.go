package ast

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper functions for creating test cases
func createMethodSignature(name string, receiverTypeName string, params, results []types.Type) *types.Signature {
	var paramVars []*types.Var
	var resultVars []*types.Var

	// Create a receiver parameter for the specified type
	pkg := types.NewPackage("test", "test")
	recvType := types.NewNamed(types.NewTypeName(0, pkg, receiverTypeName, nil), types.NewStruct(nil, nil), nil)
	recv := types.NewVar(0, pkg, "p", types.NewPointer(recvType))

	for _, p := range params {
		paramVars = append(paramVars, types.NewVar(0, nil, "", p))
	}

	for _, r := range results {
		resultVars = append(resultVars, types.NewVar(0, nil, "", r))
	}

	return types.NewSignature(recv, types.NewTuple(paramVars...), types.NewTuple(resultVars...), false)
}

func createTestFieldInfo(name string, varOrFunc FieldVarOrFunc, parent *TypeHintDefinition) *FieldInfo {
	return &FieldInfo{
		Name:   name,
		Type:   varOrFunc,
		Parent: parent,
	}
}

func createTestFieldVar(name string, typ types.Type) FieldVarOrFunc {
	return FieldVarOrFunc{
		Var: types.NewVar(0, nil, name, typ),
	}
}

func createTestFieldFunc(name string, sig *types.Signature) FieldVarOrFunc {
	return FieldVarOrFunc{
		Func: types.NewFunc(0, nil, name, sig),
	}
}

func createTestParent(name string, parent *TypeHintDefinition) *TypeHintDefinition {
	return &TypeHintDefinition{
		MyFieldInfo: FieldInfo{
			Name:   name,
			Parent: parent,
		},
	}
}

func TestNestedMultiLineTypeString(t *testing.T) {
	tests := []struct {
		name     string
		field    *FieldInfo
		expected string
	}{
		{
			name:     "nil_field",
			field:    nil,
			expected: "",
		},
		{
			name: "simple_method",
			field: createTestFieldInfo("GetJob",
				createTestFieldFunc("GetJob", createMethodSignature("GetJob", "Person", nil, []types.Type{types.Typ[types.Bool]})),
				nil),
			expected: "```go\nfunc (*Person) GetJob() (bool)\n```",
		},
		{
			name: "method_with_receiver",
			field: createTestFieldInfo("GetFullAddress",
				createTestFieldFunc("GetFullAddress", createMethodSignature("GetFullAddress", "Person", nil, []types.Type{types.Typ[types.String]})),
				createTestParent("Address", createTestParent("Person", nil))),
			expected: "```go\nfunc (*Person) GetFullAddress() (string)\n```",
		},
		{
			name: "method_with_params",
			field: createTestFieldInfo("SetName",
				createTestFieldFunc("SetName", createMethodSignature("SetName", "Person", []types.Type{types.Typ[types.String]}, nil)),
				nil),
			expected: "```go\nfunc (*Person) SetName(string)\n```",
		},
		{
			name: "method_with_multiple_params_and_returns",
			field: createTestFieldInfo("ProcessEmployee",
				createTestFieldFunc("ProcessEmployee", createMethodSignature("ProcessEmployee", "Person",
					[]types.Type{types.Typ[types.String], types.Typ[types.Int]},
					[]types.Type{types.Typ[types.Bool], types.Typ[types.String]})),
				nil),
			expected: "```go\nfunc (*Person) ProcessEmployee(string, int) (bool, string)\n```",
		},
		{
			name: "simple_field",
			field: createTestFieldInfo("Name",
				createTestFieldVar("Name", types.Typ[types.String]),
				nil),
			expected: "```go\nName string\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.NestedMultiLineTypeString()
			assert.Equal(t, tt.expected, result, "unexpected type string format")
		})
	}
}
