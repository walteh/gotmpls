package lsp

import (
	"context"

	"github.com/stretchr/testify/mock"
	pkg_types "github.com/walteh/go-tmpl-typer/pkg/types"
)

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateType(ctx context.Context, typePath string, analyzer interface{}) (*pkg_types.TypeInfo, error) {
	args := m.Called(ctx, typePath, analyzer)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkg_types.TypeInfo), args.Error(1)
}

func (m *MockValidator) ValidateField(ctx context.Context, typeInfo *pkg_types.TypeInfo, fieldPath string) (*pkg_types.FieldInfo, error) {
	args := m.Called(ctx, typeInfo, fieldPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkg_types.FieldInfo), args.Error(1)
}
