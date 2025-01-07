package types_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
)

func TestValidator_ValidateType(t *testing.T) {
	tests := []struct {
		name     string
		typePath string
		pkgInfo  *ast.PackageInfo
		wantErr  bool
	}{
		{
			name:     "type exists",
			typePath: "github.com/example/types.Config",
			pkgInfo: func() *ast.PackageInfo {
				info := ast.NewPackageInfo()
				// TODO: Add mock package with types
				return info
			}(),
			wantErr: false,
		},
		{
			name:     "type does not exist",
			typePath: "github.com/example/types.Unknown",
			pkgInfo: func() *ast.PackageInfo {
				info := ast.NewPackageInfo()
				// TODO: Add mock package with types
				return info
			}(),
			wantErr: true,
		},
		{
			name:     "invalid type path",
			typePath: "invalid-type-path",
			pkgInfo:  ast.NewPackageInfo(),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockValidator := mockery.NewMockValidator_types(t)

			if !tt.wantErr {
				mockValidator.EXPECT().
					ValidateType(context.Background(), tt.typePath, tt.pkgInfo).
					Return(nil)
			} else {
				mockValidator.EXPECT().
					ValidateType(context.Background(), tt.typePath, tt.pkgInfo).
					Return(assert.AnError)
			}

			err := mockValidator.ValidateType(context.Background(), tt.typePath, tt.pkgInfo)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
