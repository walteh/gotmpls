package ast

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

// AnalyzePackage implements PackageAnalyzer
func AnalyzePackage(ctx context.Context, dir string) (*Registry, error) {
	if strings.HasSuffix(dir, ".tmpl") {
		dir = filepath.Dir(dir)
	}

	// Check if go.mod exists
	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		filesSeen := []string{}
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			filesSeen = append(filesSeen, path)
			return nil
		})
		return nil, errors.Errorf("no packages found in directory: %s (missing go.mod) files seen: %v", dir, filesSeen)
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

	// Load all packages in the module, including examples
	patterns := []string{
		"./...",
	}

	zerolog.Ctx(ctx).Debug().Strs("patterns", patterns).Str("dir", dir).Msg("loading packages")

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

	registry := NewRegistry()

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			zerolog.Ctx(ctx).Debug().Msgf("skipping package %s: no type information\n", pkg.PkgPath)
			continue
		}

		zerolog.Ctx(ctx).Debug().Msgf("adding package to registry: %s\n", pkg.PkgPath)
		registry.AddPackage(pkg.Types)
	}

	return registry, nil
}
