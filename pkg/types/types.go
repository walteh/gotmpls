package types

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"gitlab.com/tozd/go/errors"
)

// Validator is responsible for validating types in templates
type Validator interface {
	// ValidateType validates a type against package information
	ValidateType(ctx context.Context, typePath string, pkgInfo *ast.PackageInfo) error
}

// DefaultValidator is the default implementation of Validator
type DefaultValidator struct{}

// NewDefaultValidator creates a new DefaultValidator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// ValidateType implements Validator
func (v *DefaultValidator) ValidateType(ctx context.Context, typePath string, pkgInfo *ast.PackageInfo) error {
	if !pkgInfo.TypeExists(typePath) {
		return errors.Errorf("type %s not found in package", typePath)
	}
	return nil
}
