package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

// DefaultPackageAnalyzer implements PackageAnalyzer
type DefaultPackageAnalyzer struct {
	registry *TypeRegistry
}

// NewDefaultPackageAnalyzer creates a new DefaultPackageAnalyzer
func NewDefaultPackageAnalyzer() *DefaultPackageAnalyzer {
	return &DefaultPackageAnalyzer{
		registry: NewTypeRegistry(),
	}
}

// AnalyzePackage implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) AnalyzePackage(ctx context.Context, dir string) (*TypeRegistry, error) {
	// Check if go.mod exists
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return nil, errors.Errorf("no packages found in directory: %s (missing go.mod)", dir)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, errors.Errorf("failed to load package: err: %v", err)
	}

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in directory: %s", dir)
	}

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}

		a.registry.AddPackage(pkg.Types)
	}

	return a.registry, nil
}

// GetPackage implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) GetPackage(ctx context.Context, pkgPath string) (*types.Package, error) {
	return a.registry.GetPackage(ctx, pkgPath)
}

// GetTypes implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) GetTypes(ctx context.Context, pkgPath string) (map[string]types.Object, error) {
	return a.registry.GetTypes(ctx, pkgPath)
}
