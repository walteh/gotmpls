package diagnostic_test

import (
	"context"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
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

func setupMockValidator(t *testing.T, shouldErr bool, typeInfo *pkg_types.TypeInfo, fieldInfo *pkg_types.FieldInfo, methodInfo *pkg_types.MethodInfo) *mockery.MockValidator_types {
	mockVal := mockery.NewMockValidator_types(t)

	// GetRootMethods is called in multiple places, so we'll set it up first
	mockVal.EXPECT().GetRootMethods().Return(map[string]*pkg_types.MethodInfo{}).Maybe()

	if shouldErr {
		mockVal.EXPECT().ValidateType(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.Errorf("mock error validating type")).Once()
		mockVal.EXPECT().ValidateField(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.Errorf("mock error validating field")).Maybe()
		mockVal.EXPECT().ValidateMethod(mock.Anything, mock.Anything).Return(nil, errors.Errorf("mock error validating method")).Maybe()
	} else if typeInfo != nil && len(typeInfo.Fields) > 0 && typeInfo.Fields["GetJob"] != nil {
		// This is the pipe operations test case
		mockVal.EXPECT().ValidateType(mock.Anything, mock.Anything, mock.Anything).Return(typeInfo, nil).Once()
		mockVal.EXPECT().ValidateField(mock.Anything, mock.Anything, "GetJob").Return(fieldInfo, nil).Once()

		// Method validations for pipe operations
		// First upper call with GetJob argument
		mockVal.EXPECT().ValidateMethod(mock.Anything, "upper").Return(&pkg_types.MethodInfo{
			Name:       "upper",
			Parameters: []types.Type{types.NewInterface(nil, nil)},
			Results:    []types.Type{types.Typ[types.String]},
		}, nil).Once()

		// printf call with GetJob argument
		mockVal.EXPECT().ValidateMethod(mock.Anything, "printf").Return(&pkg_types.MethodInfo{
			Name:       "printf",
			Parameters: []types.Type{types.Typ[types.String], types.NewInterface(nil, nil)},
			Results:    []types.Type{types.Typ[types.String]},
		}, nil).Once()

		// Second upper call with printf result argument
		mockVal.EXPECT().ValidateMethod(mock.Anything, "upper").Return(&pkg_types.MethodInfo{
			Name:       "upper",
			Parameters: []types.Type{types.NewInterface(nil, nil)},
			Results:    []types.Type{types.Typ[types.String]},
		}, nil).Once()

		// Additional ValidateMethod calls for argument validation
		mockVal.EXPECT().ValidateMethod(mock.Anything, "GetJob").Return(&pkg_types.MethodInfo{
			Name:       "GetJob",
			Parameters: []types.Type{},
			Results:    []types.Type{types.Typ[types.String]},
		}, nil).Maybe()

		mockVal.EXPECT().ValidateMethod(mock.Anything, "printf").Return(&pkg_types.MethodInfo{
			Name:       "printf",
			Parameters: []types.Type{types.Typ[types.String], types.NewInterface(nil, nil)},
			Results:    []types.Type{types.Typ[types.String]},
		}, nil).Maybe()
	} else if typeInfo != nil {
		// This is the regular test case
		mockVal.EXPECT().ValidateType(mock.Anything, mock.Anything, mock.Anything).Return(typeInfo, nil).Once()
		mockVal.EXPECT().ValidateField(mock.Anything, mock.Anything, "Name").Return(fieldInfo, nil).Once()
		mockVal.EXPECT().ValidateField(mock.Anything, mock.Anything, "Address.Street").Return(fieldInfo, nil).Once()
		mockVal.EXPECT().ValidateMethod(mock.Anything, "GetName").Return(methodInfo, nil).Once()
	}

	return mockVal
}

func TestDefaultGenerator_Generate(t *testing.T) {
	tests := []struct {
		name          string
		info          *parser.TemplateInfo
		typeValidator pkg_types.Validator
		registry      *ast.TypeRegistry
		want          *diagnostic.Diagnostics
		wantErr       bool
	}{
		{
			name: "valid template info",
			info: mockTemplateInfo(),
			typeValidator: setupMockValidator(t, false, &pkg_types.TypeInfo{
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
			}, &pkg_types.FieldInfo{
				Name: "Name",
				Type: types.Typ[types.String],
			}, &pkg_types.MethodInfo{
				Name:       "GetName",
				Parameters: []types.Type{},
				Results:    []types.Type{types.Typ[types.String]},
			}),
			registry: mockRegistry(),
			want: &diagnostic.Diagnostics{
				Errors:   []diagnostic.Diagnostic{},
				Warnings: []diagnostic.Diagnostic{},
				Hints: []diagnostic.Diagnostic{
					{
						Message:  "Type: string",
						Line:     3,
						Column:   9,
						EndLine:  3,
						EndCol:   13,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: string",
						Line:     4,
						Column:   9,
						EndLine:  4,
						EndCol:   22,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     5,
						Column:   9,
						EndLine:  5,
						EndCol:   16,
						Severity: diagnostic.Hint,
					},
				},
			},
			wantErr: false,
		},
		{
			name:          "invalid type hint",
			info:          mockTemplateInfo(),
			typeValidator: setupMockValidator(t, true, nil, nil, nil),
			registry:      mockRegistry(),
			want: &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{
					{
						Message:  "Invalid type hint: mock error validating type",
						Line:     1,
						Column:   12,
						EndLine:  1,
						EndCol:   43,
						Severity: diagnostic.Error,
					},
				},
				Warnings: []diagnostic.Diagnostic{},
			},
			wantErr: false,
		},
		{
			name: "no type hint",
			info: &parser.TemplateInfo{
				Filename:  "test.tmpl",
				TypeHints: []parser.TypeHint{},
			},
			typeValidator: setupMockValidator(t, false, nil, nil, nil),
			registry:      mockRegistry(),
			want: &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{},
				Warnings: []diagnostic.Diagnostic{
					{
						Message:  "No type hint found in template",
						Line:     1,
						Column:   1,
						EndLine:  1,
						EndCol:   1,
						Severity: diagnostic.Warning,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "pipe operations with function arguments",
			info: func() *parser.TemplateInfo {
				getJobVar := &parser.VariableLocation{
					Name:    "GetJob",
					Line:    15,
					Column:  8,
					EndLine: 15,
					EndCol:  14,
				}
				printfFunc := &parser.VariableLocation{
					Name:    "printf",
					Line:    16,
					Column:  9,
					EndLine: 16,
					EndCol:  15,
					MethodArguments: []types.Type{
						types.Typ[types.String],
						getJobVar,
					},
				}
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
						*getJobVar,
					},
					Functions: []parser.VariableLocation{
						{
							Name:    "upper",
							Line:    15,
							Column:  17,
							EndLine: 15,
							EndCol:  22,
							MethodArguments: []types.Type{
								getJobVar,
							},
						},
						*printfFunc,
						{
							Name:    "upper",
							Line:    16,
							Column:  30,
							EndLine: 16,
							EndCol:  35,
							MethodArguments: []types.Type{
								printfFunc,
							},
						},
					},
				}
			}(),
			typeValidator: setupMockValidator(t, false, &pkg_types.TypeInfo{
				Name: "Person",
				Fields: map[string]*pkg_types.FieldInfo{
					"GetJob": {
						Name: "GetJob",
						Type: types.Typ[types.String],
					},
				},
			}, &pkg_types.FieldInfo{
				Name: "GetJob",
				Type: types.Typ[types.String],
			}, &pkg_types.MethodInfo{
				Name:       "upper",
				Parameters: []types.Type{types.NewInterface(nil, nil)},
				Results:    []types.Type{types.Typ[types.String]},
			}),
			registry: mockRegistry(),
			want: &diagnostic.Diagnostics{
				Errors:   []diagnostic.Diagnostic{},
				Warnings: []diagnostic.Diagnostic{},
				Hints: []diagnostic.Diagnostic{
					{
						Message:  "Type: string",
						Line:     15,
						Column:   8,
						EndLine:  15,
						EndCol:   14,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     15,
						Column:   17,
						EndLine:  15,
						EndCol:   22,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     16,
						Column:   9,
						EndLine:  16,
						EndCol:   15,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     16,
						Column:   30,
						EndLine:  16,
						EndCol:   35,
						Severity: diagnostic.Hint,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := diagnostic.NewDefaultGenerator()
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
	f := diagnostic.NewVSCodeFormatter()
	diagnostics := &diagnostic.Diagnostics{
		Errors: []diagnostic.Diagnostic{
			{
				Message:  "Test error",
				Line:     1,
				Column:   1,
				EndLine:  1,
				EndCol:   10,
				Severity: diagnostic.Error,
			},
		},
		Warnings: []diagnostic.Diagnostic{
			{
				Message:  "Test warning",
				Line:     2,
				Column:   1,
				EndLine:  2,
				EndCol:   10,
				Severity: diagnostic.Warning,
			},
		},
	}

	formatted, err := f.Format(diagnostics)
	require.NoError(t, err)
	require.NotEmpty(t, formatted)
}
