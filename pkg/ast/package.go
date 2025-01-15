package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedEmbedFiles |
	packages.NeedModule |
	packages.NeedTypesSizes |
	packages.NeedTarget

// type TemplateWithContent struct {
// 	*template.Template
// 	Content string
// 	Parsed  *parser.ParsedTemplateFile
// }

// type PackageWithTemplates struct {
// 	Package   *packages.Package
// 	Templates map[string]*parser.ParsedTemplateFile
// }

type PackageWithTemplateFiles struct {
	Package       *packages.Package
	TemplateFiles map[string]string
}

// var supportedTemplateExtensions = []string{"tmpl", "go"}

func LoadPackageTypesFromFs(ctx context.Context, dir string, overlay map[string][]byte) ([]*PackageWithTemplateFiles, error) {

	// // Check if go.mod exists
	// modPath := filepath.Join(dir, "go.mod")
	// if _, err := os.Stat(modPath); err != nil {
	// 	filesSeen := []string{}
	// 	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
	// 		if err != nil {
	// 			return err
	// 		}
	// 		filesSeen = append(filesSeen, path)
	// 		return nil
	// 	})
	// 	return nil, errors.Errorf("no packages found in directory: %s (missing go.mod) files seen: %v", dir, filesSeen)
	// }

	// zerolog.Ctx(ctx).Debug().Msgf("analyzing packages in directory: %s\n", dir)

	// // Read go.mod to get module name
	// modContent, err := os.ReadFile(modPath)
	// if err != nil {
	// 	return nil, errors.Errorf("failed to read go.mod: %v", err)
	// }
	// zerolog.Ctx(ctx).Debug().Msgf("go.mod content:\n%s\n", string(modContent))

	cfg := &packages.Config{
		Mode:    loadMode,
		Dir:     dir,
		Env:     append(os.Environ(), "GO111MODULE=on"),
		Overlay: overlay,
	}

	// Load all packages in the module, including examples
	patterns := []string{
		"./...",
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, errors.Errorf("failed to load package: err: %v", err)
	}

	pkgd := make(map[string]*types.Package)

	if len(pkgs) == 0 {
		// try loading the base package
		cfg.Dir = filepath.Dir(dir)
		pkg, err := packages.Load(cfg, "./...")
		if err != nil {
			return nil, errors.Errorf("no packages found in directory: %s", dir)
		}
		pkgs = append(pkgs, pkg...)
		if len(pkgs) == 0 {
			return nil, errors.Errorf("no packages found in directory: %s", dir)
		}
	}

	pkgNames := []string{}
	for _, pkg := range pkgs {
		pkgNames = append(pkgNames, pkg.Name)
	}
	zerolog.Ctx(ctx).Debug().Msgf("loaded %d packages: %v", len(pkgs), pkgNames)
	for _, pkg := range pkgs {
		zerolog.Ctx(ctx).Debug().Msgf("package: %s (path: %s)\n", pkg.Name, pkg.PkgPath)
		if len(pkg.Errors) > 0 {
			zerolog.Ctx(ctx).Debug().Msgf(" errors:\n")
			for _, err := range pkg.Errors {
				zerolog.Ctx(ctx).Debug().Msgf("    - %v\n", err)
			}
		}
		if pkg.Module != nil {
			zerolog.Ctx(ctx).Debug().Msgf("  module: %s\n", pkg.Module.Path)
		}

		pkgd[pkg.PkgPath] = pkg.Types
	}

	if len(pkgs) == 1 && pkgs[0].ID == "./..." {
		return nil, errors.Errorf("no main module found for directory '%s', please ensure a parent directory with a go.mod file is provided", dir)
	}

	pkgWithTemplateFilesList := []*PackageWithTemplateFiles{}

	for _, pkg := range pkgs {

		pkgWithTemplateFiles := &PackageWithTemplateFiles{
			Package:       pkg,
			TemplateFiles: make(map[string]string),
		}
		for _, file := range pkg.EmbedFiles {
			ext := filepath.Ext(file)
			// == tmpl, contains .tmpl., starts with tmpl., ends with .tmpl
			if ext != ".tmpl" && !strings.Contains(file, ".tmpl.") && !strings.HasSuffix(file, ".tmpl") && !strings.HasPrefix(file, "tmpl.") {
				continue
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, errors.Errorf("failed to read file: %w", err)
			}
			pkgWithTemplateFiles.TemplateFiles[file] = string(content)
		}

		pkgWithTemplateFilesList = append(pkgWithTemplateFilesList, pkgWithTemplateFiles)
	}

	return pkgWithTemplateFilesList, nil
}

func (me *PackageWithTemplateFiles) LoadTypeByPath(ctx context.Context, path string) (types.Object, error) {
	final := filepath.Base(path)
	pkg := me.Package.Types.Scope().Lookup(final)
	if pkg == nil {
		return nil, errors.Errorf("type not found: %s", final)
	}
	return pkg, nil
}

// func LoadTemplatesFromFs(ctx context.Context, dir string) (map[string]*template.Template, error) {

// 	files, err := filepath.Glob(filepath.Join(dir, "**", "*.tmpl"))
// 	if err != nil {
// 		return nil, errors.Errorf("failed to read directory: %w", err)
// 	}

// 	templates := make(map[string]*template.Template)

// 	for _, file := range files {
// 		tmpl, err := ParseTemplate(ctx, file, string(content))
// 		if err != nil {
// 			return nil, errors.Errorf("failed to parse template: %w", err)
// 		}
// 		templates[file] = tmpl
// 	}

// 	return templates, nil
// }

// AnalyzePackage implements PackageAnalyzer
func AnalyzePackage(ctx context.Context, dir string, overlay map[string][]byte) (*Registry, error) {
	if strings.HasSuffix(dir, ".tmpl") || strings.HasSuffix(dir, ".go") {
		dir = filepath.Dir(dir)
	}

	pkgWithTemplateFilesList, err := LoadPackageTypesFromFs(ctx, dir, overlay)
	if err != nil {
		return nil, errors.Errorf("failed to load package: %w", err)
	}

	// pkgWithTemplateFilesList := []*PackageWithTemplateFiles{}

	// for _, pkgWithTemplateFiles := range pkgWithTemplateFilesList {
	// 	pkgWithTemplateFiles := &PackageWithTemplateFiles{
	// 		Package: pkgWithTemplateFiles.Package,
	// 	}

	// 	for fileName, content := range pkgWithTemplateFiles.TemplateFiles {
	// 		// tmpl, err := parser.Parse(ctx, fileName, []byte(content))
	// 		// if err != nil {
	// 		// 	return nil, errors.Errorf("failed to parse template: %w", err)
	// 		// }
	// 		pkgWithTemplateFiles.TemplateFiles[fileName] = content
	// 	}

	// 	pkgWithTemplateFilesList = append(pkgWithTemplateFilesList, pkgWithTemplateFiles)
	// }

	registry := NewRegistry(pkgWithTemplateFilesList)

	return registry, nil
}
