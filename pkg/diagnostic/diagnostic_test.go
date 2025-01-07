package diagnostic

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

func mockRegistry() *ast.TypeRegistry {
	return &ast.TypeRegistry{
		Types: map[string]*types.Package{
			"github.com/example/types": types.NewPackage("github.com/example/types", "types"),
		},
	}
}

// mockTemplateInfo creates a mock template info for testing
func mockTemplateInfo() *parser.TemplateInfo {
	return &parser.TemplateInfo{
		Filename: "test.tmpl",
		TypeHints: []parser.TypeHint{
			{
				TypePath: "github.com/example/types.Person",
				Line:     1,
				Column:   12,
			},
		},
		Variables: []parser.VariableLocation{
			{
				Name:    "Name",
				Line:    3,
				Column:  9,
				EndLine: 3,
				EndCol:  13,
			},
			{
				Name:    "Address.Street",
				Line:    4,
				Column:  9,
				EndLine: 4,
				EndCol:  22,
			},
		},
		Functions: []parser.FunctionLocation{
			{
				Name:      "GetName",
				Line:      5,
				Column:    9,
				EndLine:   5,
				EndCol:    16,
				Arguments: []string{},
			},
		},
		// Definitions: []parser.DefinitionInfo{
		// 	{
		// 		Name:     "Name",
		// 		Line:     7,
		// 		Column:   1,
		// 		EndLine:  9,
		// 		EndCol:   7,
		// 		NodeType: "definition",
		// 	},
		// },
	}
}

// mockTypeValidator creates a mock type validator for testing
type mockTypeValidator struct {
	typeInfo  *pkg_types.TypeInfo
	fieldInfo *pkg_types.FieldInfo
	shouldErr bool
}

func (m *mockTypeValidator) ValidateType(ctx context.Context, typePath string, registry *ast.TypeRegistry) (*pkg_types.TypeInfo, error) {
	if m.shouldErr {
		return nil, errors.Errorf("mock error validating type")
	}
	if m.typeInfo == nil {
		return nil, errors.Errorf("type %s not found", typePath)
	}
	return m.typeInfo, nil
}

func (m *mockTypeValidator) ValidateField(ctx context.Context, typeInfo *pkg_types.TypeInfo, fieldPath string) (*pkg_types.FieldInfo, error) {
	if m.shouldErr {
		return nil, errors.Errorf("mock error validating field")
	}
	if m.fieldInfo == nil {
		return nil, errors.Errorf("field %s not found", fieldPath)
	}
	return m.fieldInfo, nil
}

func (m *mockTypeValidator) ValidateMethod(ctx context.Context, typeInfo *pkg_types.TypeInfo, methodName string) (*pkg_types.MethodInfo, error) {
	if m.shouldErr {
		return nil, errors.Errorf("mock error validating method")
	}
	if methodName == "NonExistent" {
		return nil, errors.Errorf("method %s not found", methodName)
	}
	// Return method info based on the mock type info
	if typeInfo != nil && typeInfo.Methods != nil {
		if method, ok := typeInfo.Methods[methodName]; ok {
			return method, nil
		}
	}
	// Default method info if not found in type info
	return &pkg_types.MethodInfo{
		Name:       methodName,
		Parameters: []types.Type{},
		Results:    []types.Type{types.Typ[types.String]},
	}, nil
}

func TestDefaultGenerator_Generate(t *testing.T) {
	tests := []struct {
		name          string
		templateInfo  *parser.TemplateInfo
		typeValidator pkg_types.Validator
		wantErrCount  int
		wantWarnCount int
		registry      *ast.TypeRegistry
	}{
		{
			name:         "valid template",
			templateInfo: mockTemplateInfo(),
			registry:     mockRegistry(),
			typeValidator: &mockTypeValidator{
				typeInfo: &pkg_types.TypeInfo{
					Name: "Person",
					Fields: map[string]*pkg_types.FieldInfo{
						"Name": {
							Name: "Name",
							Type: types.Typ[types.String],
						},
						"Address": {
							Name: "Address",
							Type: types.NewStruct([]*types.Var{
								types.NewField(0, nil, "Street", types.Typ[types.String], false),
							}, nil),
						},
					},
					Methods: map[string]*pkg_types.MethodInfo{
						"GetName": {
							Name:       "GetName",
							Parameters: []types.Type{},
							Results:    []types.Type{types.Typ[types.String]},
						},
					},
				},
				fieldInfo: &pkg_types.FieldInfo{
					Name: "Name",
					Type: types.Typ[types.String],
				},
			},
			wantErrCount:  0,
			wantWarnCount: 5,
		},
		{
			name:         "missing type hint",
			templateInfo: &parser.TemplateInfo{},
			typeValidator: &mockTypeValidator{
				shouldErr: true,
			},
			wantErrCount:  0,
			wantWarnCount: 1,
		},
		{
			name: "invalid type hint",
			templateInfo: &parser.TemplateInfo{
				TypeHints: []parser.TypeHint{
					{
						TypePath: "invalid.Type",
						Line:     1,
						Column:   12,
					},
				},
			},
			typeValidator: &mockTypeValidator{
				shouldErr: true,
			},
			wantErrCount:  1,
			wantWarnCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDefaultGenerator()
			diagnostics, err := g.Generate(context.Background(), tt.templateInfo, tt.typeValidator, tt.registry)
			require.NoError(t, err)
			require.NotNil(t, diagnostics)

			assert.Len(t, diagnostics.Errors, tt.wantErrCount, "unexpected number of errors")
			assert.Len(t, diagnostics.Warnings, tt.wantWarnCount, "unexpected number of warnings")
		})
	}
}

func TestVSCodeFormatter_Format(t *testing.T) {
	f := NewVSCodeFormatter()
	diagnostics := &Diagnostics{
		Errors: []Diagnostic{
			{
				Message:  "Test error",
				Line:     1,
				Column:   1,
				EndLine:  1,
				EndCol:   10,
				Severity: Error,
			},
		},
		Warnings: []Diagnostic{
			{
				Message:  "Test warning",
				Line:     2,
				Column:   1,
				EndLine:  2,
				EndCol:   10,
				Severity: Warning,
			},
		},
	}

	_, err := f.Format(diagnostics)
	assert.NoError(t, err) // Currently returns "not implemented"
	// assert.Contains(t, err.Error(), "not implemented")
}
