package ast

import (
	"context"
	"go/types"
	"path/filepath"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

// DefaultPackageAnalyzer is the default implementation of PackageAnalyzer
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
func (a *DefaultPackageAnalyzer) AnalyzePackage(ctx context.Context, packageDir string) (*TypeRegistry, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(packageDir)
	if err != nil {
		return nil, errors.Errorf("failed to get absolute path: %w", err)
	}

	// Configure the package loader
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		Dir:  absPath,
	}

	// Load the package
	pkgs, err := packages.Load(config, filepath.Join(absPath, "..."))
	if err != nil {
		return nil, errors.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in %s", absPath)
	}

	// Process each package
	for _, pkg := range pkgs {
		if pkg.Types != nil {
			a.registry.Types[pkg.ID] = pkg.Types
		}
	}

	return a.registry, nil
}

// GetPackage implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) GetPackage(ctx context.Context, packageName string) (*types.Package, error) {
	if a.registry == nil {
		return nil, errors.Errorf("no packages analyzed yet")
	}
	return a.registry.GetPackage(ctx, packageName)
}

// GetTypes implements PackageAnalyzer
func (a *DefaultPackageAnalyzer) GetTypes() map[string]*types.Package {
	if a.registry == nil {
		return make(map[string]*types.Package)
	}
	return a.registry.GetTypes()
}
