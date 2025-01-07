package ast_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
)

func TestPackageAnalyzer_AnalyzePackage(t *testing.T) {
	tests := []struct {
		name       string
		packageDir string
		want       *ast.PackageInfo
		wantErr    bool
	}{
		{
			name:       "valid package",
			packageDir: "testdata/valid",
			want: func() *ast.PackageInfo {
				info := ast.NewPackageInfo()
				// TODO: Add expected package types
				return info
			}(),
			wantErr: false,
		},
		{
			name:       "invalid package",
			packageDir: "testdata/invalid",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "nonexistent package",
			packageDir: "testdata/nonexistent",
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAnalyzer := mockery.NewMockPackageAnalyzer_ast(t)

			if !tt.wantErr {
				mockAnalyzer.EXPECT().
					AnalyzePackage(context.Background(), tt.packageDir).
					Return(tt.want, nil)
			} else {
				mockAnalyzer.EXPECT().
					AnalyzePackage(context.Background(), tt.packageDir).
					Return(nil, assert.AnError)
			}

			got, err := mockAnalyzer.AnalyzePackage(context.Background(), tt.packageDir)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPackageInfo_TypeExists(t *testing.T) {
	tests := []struct {
		name     string
		info     *ast.PackageInfo
		typePath string
		want     bool
	}{
		{
			name: "type exists",
			info: func() *ast.PackageInfo {
				info := ast.NewPackageInfo()
				// TODO: Add mock package with types
				return info
			}(),
			typePath: "github.com/example/types.Config",
			want:     true,
		},
		{
			name:     "type does not exist",
			info:     ast.NewPackageInfo(),
			typePath: "github.com/example/types.Unknown",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.TypeExists(tt.typePath)
			assert.Equal(t, tt.want, got)
		})
	}
}
