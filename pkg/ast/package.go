package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
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

	zerolog.Ctx(ctx).Debug().Msgf("analyzing packages in directory: %s\n", dir)

	// Read go.mod to get module name
	modContent, err := os.ReadFile(modPath)
	if err != nil {
		return nil, errors.Errorf("failed to read go.mod: %v", err)
	}
	zerolog.Ctx(ctx).Debug().Msgf("go.mod content:\n%s\n", string(modContent))

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule | packages.NeedImports | packages.NeedDeps,
		Dir:  dir,
		Env:  append(os.Environ(), "GO111MODULE=on"),
	}

	zerolog.Ctx(ctx).Debug().Msgf("loading packages with config: %+v\n", cfg)

	// Load all packages in the module, including examples
	patterns := []string{
		"./...",
	}

	zerolog.Ctx(ctx).Debug().Msgf("loading packages with patterns: %v\n", patterns)

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, errors.Errorf("failed to load package: err: %v", err)
	}

	zerolog.Ctx(ctx).Debug().Msgf("loaded %d packages\n", len(pkgs))
	for _, pkg := range pkgs {
		zerolog.Ctx(ctx).Debug().Msgf("package: %s (path: %s)\n", pkg.Name, pkg.PkgPath)
		if len(pkg.Errors) > 0 {
			zerolog.Ctx(ctx).Debug().Msgf("  errors:\n")
			for _, err := range pkg.Errors {
				zerolog.Ctx(ctx).Debug().Msgf("    - %v\n", err)
			}
		}
		if pkg.Module != nil {
			zerolog.Ctx(ctx).Debug().Msgf("  module: %s\n", pkg.Module.Path)
		}
	}

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in directory: %s", dir)
	}

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			zerolog.Ctx(ctx).Debug().Msgf("skipping package %s: no type information\n", pkg.PkgPath)
			continue
		}

		zerolog.Ctx(ctx).Debug().Msgf("adding package to registry: %s\n", pkg.PkgPath)
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
