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
		Functions: []parser.VariableLocation{
			{
				Name:            "GetName",
				Line:            5,
				Column:          9,
				EndLine:         5,
				EndCol:          16,
				MethodArguments: []types.Type{},
			},
		},
	}
}

// mockTypeValidator creates a mock type validator for testing
type mockTypeValidator struct {
	typeInfo   *pkg_types.TypeInfo
	fieldInfo  *pkg_types.FieldInfo
	methodInfo *pkg_types.MethodInfo
	shouldErr  bool
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

func (m *mockTypeValidator) ValidateMethod(ctx context.Context, methodName string) (*pkg_types.MethodInfo, error) {
	if m.shouldErr {
		return nil, errors.Errorf("mock error validating method")
	}
	if m.methodInfo == nil {
		return nil, errors.Errorf("method %s not found", methodName)
	}
	return m.methodInfo, nil
}

var _ pkg_types.Validator = &mockTypeValidator{}

func TestDefaultGenerator_Generate(t *testing.T) {
	tests := []struct {
		name          string
		info          *parser.TemplateInfo
		typeValidator pkg_types.Validator
		registry      *ast.TypeRegistry
		want          *Diagnostics
		wantErr       bool
	}{
		{
			name: "valid template info",
			info: mockTemplateInfo(),
			typeValidator: &mockTypeValidator{
				typeInfo: &pkg_types.TypeInfo{
					Name: "Person",
					Fields: map[string]*pkg_types.FieldInfo{
						"Name": {
							Name: "Name",
							Type: types.Typ[types.String],
						},
						"Address.Street": {
							Name: "Address.Street",
							Type: types.Typ[types.String],
						},
					},
				},
				fieldInfo: &pkg_types.FieldInfo{
					Name: "Name",
					Type: types.Typ[types.String],
				},
				methodInfo: &pkg_types.MethodInfo{
					Name:       "GetName",
					Parameters: []types.Type{},
					Results:    []types.Type{types.Typ[types.String]},
				},
			},
			registry: mockRegistry(),
			want: &Diagnostics{
				Errors:   []Diagnostic{},
				Warnings: []Diagnostic{},
				Hints: []Diagnostic{
					{
						Message:  "Type: string",
						Line:     3,
						Column:   9,
						EndLine:  3,
						EndCol:   13,
						Severity: Hint,
					},
					{
						Message:  "Type: string",
						Line:     4,
						Column:   9,
						EndLine:  4,
						EndCol:   22,
						Severity: Hint,
					},
					{
						Message:  "Returns: string",
						Line:     5,
						Column:   9,
						EndLine:  5,
						EndCol:   16,
						Severity: Hint,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type hint",
			info: mockTemplateInfo(),
			typeValidator: &mockTypeValidator{
				shouldErr: true,
			},
			registry: mockRegistry(),
			want: &Diagnostics{
				Errors: []Diagnostic{
					{
						Message:  "Invalid type hint: mock error validating type",
						Line:     1,
						Column:   12,
						EndLine:  1,
						EndCol:   43,
						Severity: Error,
					},
				},
				Warnings: []Diagnostic{},
			},
			wantErr: false,
		},
		{
			name: "no type hint",
			info: &parser.TemplateInfo{
				Filename:  "test.tmpl",
				TypeHints: []parser.TypeHint{},
			},
			typeValidator: &mockTypeValidator{},
			registry:      mockRegistry(),
			want: &Diagnostics{
				Errors: []Diagnostic{},
				Warnings: []Diagnostic{
					{
						Message:  "No type hint found in template",
						Line:     1,
						Column:   1,
						EndLine:  1,
						EndCol:   1,
						Severity: Warning,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDefaultGenerator()
			got, err := g.Generate(context.Background(), tt.info, tt.typeValidator, tt.registry)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
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
