package ast

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBuiltinMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantName string
		wantType types.Type
	}{
		{
			name:     "upper method",
			method:   "upper",
			wantName: "upper",
			wantType: types.Typ[types.String],
		},
		{
			name:     "and method",
			method:   "canBeNil",
			wantName: "canBeNil",
			wantType: types.Typ[types.Bool],
		},
		{
			name:     "non-existent method",
			method:   "nonexistent",
			wantName: "",
			wantType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := GetBuiltinMethod(tt.method)
			if tt.wantName == "" {
				assert.Nil(t, method, "method should not exist")
				return
			}

			assert.NotNil(t, method, "method should exist")
			assert.Equal(t, tt.wantName, method.Name, "method name should match")

			if tt.wantType != nil {
				assert.Equal(t, tt.wantType, method.Results[0], "return type should match")
			}
		})
	}
}
