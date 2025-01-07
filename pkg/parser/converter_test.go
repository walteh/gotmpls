package parser

import (
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/assert"
)

func TestConverter_SimpleVariableReference(t *testing.T) {
	// Test template with a simple variable reference
	source := "{{.Name}}"
	tree, err := parse.Parse("test.tmpl", source, "{{", "}}")
	assert.NoError(t, err)

	conv := &Converter{}
	got, err := conv.ConvertTree(tree["test.tmpl"].Root, source)
	assert.NoError(t, err)

	// Verify the variable location
	assert.Len(t, got.Variables, 1)
	assert.Equal(t, "Name", got.Variables[0].Name)
	assert.Equal(t, 1, got.Variables[0].Line)
	assert.Equal(t, 3, got.Variables[0].Column)
	assert.Equal(t, 1, got.Variables[0].EndLine)
	assert.Equal(t, 7, got.Variables[0].EndCol)
}

func TestConverter_ConvertTree(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantVars []VariableLocation
		wantFns  []FunctionLocation
	}{
		{
			name:   "simple variable reference",
			source: "{{.Name}}",
			wantVars: []VariableLocation{
				{
					Name:    "Name",
					Line:    1,
					Column:  3,
					EndLine: 1,
					EndCol:  7,
				},
			},
		},
		{
			name:   "function call",
			source: "{{printf \"Hello %s\" .Name | upper}}",
			wantVars: []VariableLocation{
				{
					Name:    "Name",
					Line:    1,
					Column:  21,
					EndLine: 1,
					EndCol:  25,
				},
			},
			wantFns: []FunctionLocation{
				{
					Name:    "printf",
					Line:    1,
					Column:  3,
					EndLine: 1,
					EndCol:  9,
				},
				{
					Name:    "upper",
					Line:    1,
					Column:  28,
					EndLine: 1,
					EndCol:  33,
				},
			},
		},
		{
			name:     "empty template",
			source:   "",
			wantVars: nil,
			wantFns:  nil,
		},
		{
			name:     "invalid template",
			source:   "{{.Name",
			wantVars: nil,
			wantFns:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := parse.Parse("test.tmpl", tt.source, "{{", "}}")
			if err != nil {
				// Skip invalid template tests
				return
			}

			conv := &Converter{}
			got, err := conv.ConvertTree(tree["test.tmpl"].Root, tt.source)
			assert.NoError(t, err)

			// Verify variables
			assert.Equal(t, tt.wantVars, got.Variables)

			// Verify functions
			assert.Equal(t, tt.wantFns, got.Functions)
		})
	}
}

func TestConverter_positionFromOffset(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		offset   int
		wantLine int
		wantCol  int
	}{
		{
			name:     "start of template",
			source:   "{{.Name}}",
			offset:   0,
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "after opening braces",
			source:   "{{.Name}}",
			offset:   2,
			wantLine: 1,
			wantCol:  3,
		},
		{
			name:     "at variable name",
			source:   "{{.Name}}",
			offset:   3,
			wantLine: 1,
			wantCol:  4,
		},
		{
			name:     "with newline",
			source:   "line1\n{{.Name}}",
			offset:   8,
			wantLine: 2,
			wantCol:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotCol := positionFromOffset(tt.source, tt.offset)
			assert.Equal(t, tt.wantLine, gotLine)
			assert.Equal(t, tt.wantCol, gotCol)
		})
	}
}
