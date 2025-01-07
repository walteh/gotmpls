package diagnostic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
)

func TestGenerator_Generate(t *testing.T) {
	tests := []struct {
		name    string
		info    *parser.TemplateInfo
		want    *diagnostic.Diagnostics
		wantErr bool
	}{
		{
			name: "valid template info",
			info: &parser.TemplateInfo{
				Variables: []parser.VariableLocation{
					{
						Name:    "Name",
						Line:    4,
						Column:  9,
						EndLine: 4,
						EndCol:  13,
					},
				},
				TypeHints: []parser.TypeHint{
					{
						TypePath: "github.com/example/types.Config",
						Line:     1,
						Column:   12,
					},
				},
				Filename: "test.tmpl",
			},
			want: &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{
					{
						Message:  "Type 'Name' not found in Config",
						Line:     4,
						Column:   9,
						EndLine:  4,
						EndCol:   13,
						Severity: diagnostic.Error,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing type hint",
			info: &parser.TemplateInfo{
				Variables: []parser.VariableLocation{
					{
						Name:    "Name",
						Line:    4,
						Column:  9,
						EndLine: 4,
						EndCol:  13,
					},
				},
				Filename: "test.tmpl",
			},
			want: &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{
					{
						Message:  "No type hint found in template",
						Line:     1,
						Column:   1,
						EndLine:  1,
						EndCol:   1,
						Severity: diagnostic.Error,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGen := mockery.NewMockGenerator_diagnostic(t)

			if !tt.wantErr {
				mockGen.EXPECT().
					Generate(context.Background(), tt.info).
					Return(tt.want, nil)
			} else {
				mockGen.EXPECT().
					Generate(context.Background(), tt.info).
					Return(nil, assert.AnError)
			}

			got, err := mockGen.Generate(context.Background(), tt.info)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatter_Format(t *testing.T) {
	tests := []struct {
		name        string
		diagnostics *diagnostic.Diagnostics
		want        []byte
		wantErr     bool
	}{
		{
			name: "format vscode diagnostics",
			diagnostics: &diagnostic.Diagnostics{
				Errors: []diagnostic.Diagnostic{
					{
						Message:  "Type 'Name' not found in Config",
						Line:     4,
						Column:   9,
						EndLine:  4,
						EndCol:   13,
						Severity: diagnostic.Error,
					},
				},
			},
			want:    []byte(`[{"message":"Type 'Name' not found in Config","range":{"start":{"line":3,"character":8},"end":{"line":3,"character":12}},"severity":1}]`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFmt := mockery.NewMockFormatter_diagnostic(t)

			if !tt.wantErr {
				mockFmt.EXPECT().
					Format(tt.diagnostics).
					Return(tt.want, nil)
			} else {
				mockFmt.EXPECT().
					Format(tt.diagnostics).
					Return(nil, assert.AnError)
			}

			got, err := mockFmt.Format(tt.diagnostics)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, string(tt.want), string(got))
		})
	}
}
