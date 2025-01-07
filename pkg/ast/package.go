package ast

import (
	"context"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

// DefaultPackageAnalyzer is the default implementation of PackageAnalyzer
type DefaultPackageAnalyzer struct{}

// NewDefaultPackageAnalyzer creates a new DefaultPackageAnalyzer
func NewDefaultPackageAnalyzer() *DefaultPackageAnalyzer {
	return &DefaultPackageAnalyzer{}
}

// AnalyzePackage implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) AnalyzePackage(ctx context.Context, packageDir string) (*TypeRegistry, error) {
	// Create a new type registry
	registry := NewTypeRegistry()

	// Configure the package loader
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		Dir:  packageDir,
	}

	// Load the package
	pkgs, err := packages.Load(config, "./...")
	if err != nil {
		return nil, errors.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in %s", packageDir)
	}

	// for _, pkg := range pkgs {
	// 	// registry.Types[pkg.PkgPath] = pkg.Types
	// 	pp.Println(pkg)
	// }

	// Process each package
	for _, pkg := range pkgs {
		if pkg.Types != nil {
			registry.Types[pkg.ID] = pkg.Types
		}
	}

	return registry, nil
}
