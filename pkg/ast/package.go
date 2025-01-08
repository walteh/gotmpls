package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"

	"github.com/walteh/go-tmpl-typer/pkg/debug"
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
	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		return nil, errors.Errorf("no packages found in directory: %s (missing go.mod)", dir)
	}

	debug.Printf("analyzing packages in directory: %s\n", dir)

	// Read go.mod to get module name
	modContent, err := os.ReadFile(modPath)
	if err != nil {
		return nil, errors.Errorf("failed to read go.mod: %v", err)
	}
	debug.Printf("go.mod content:\n%s\n", string(modContent))

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule | packages.NeedImports | packages.NeedDeps,
		Dir:  dir,
		Env:  append(os.Environ(), "GO111MODULE=on"),
	}

	debug.Printf("loading packages with config: %+v\n", cfg)

	// Load all packages in the module, including examples
	patterns := []string{
		"./...",
	}

	debug.Printf("loading packages with patterns: %v\n", patterns)

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, errors.Errorf("failed to load package: err: %v", err)
	}

	debug.Printf("loaded %d packages\n", len(pkgs))
	for _, pkg := range pkgs {
		debug.Printf("package: %s (path: %s)\n", pkg.Name, pkg.PkgPath)
		if len(pkg.Errors) > 0 {
			debug.Printf("  errors:\n")
			for _, err := range pkg.Errors {
				debug.Printf("    - %v\n", err)
			}
		}
		if pkg.Module != nil {
			debug.Printf("  module: %s\n", pkg.Module.Path)
		}
	}

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in directory: %s", dir)
	}

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			debug.Printf("skipping package %s: no type information\n", pkg.PkgPath)
			continue
		}

		debug.Printf("adding package to registry: %s\n", pkg.PkgPath)
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
