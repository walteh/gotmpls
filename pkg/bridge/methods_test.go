package bridge

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
		wantType string
	}{
		{
			name:     "upper method",
			method:   "upper",
			wantName: "upper",
			wantType: "string",
		},
		{
			name:     "and method",
			method:   "and",
			wantName: "and",
			wantType: "bool",
		},
		{
			name:     "non-existent method",
			method:   "nonexistent",
			wantName: "",
			wantType: "",
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

			if tt.wantType != "" {
				switch tt.wantType {
				case "string":
					assert.Equal(t, types.Typ[types.String], method.Parameters[0], "parameter type should be string")
				case "bool":
					assert.Equal(t, types.Typ[types.Bool], method.Parameters[0], "parameter type should be bool")
				}
			}
		})
	}
}
