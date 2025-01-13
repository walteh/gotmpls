package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

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

type PackageWithTemplates struct {
	Package   *packages.Package
	Templates map[string]*template.Template
}

type PackageWithTemplateFiles struct {
	Package       *packages.Package
	TemplateFiles map[string]string
}

// var supportedTemplateExtensions = []string{"tmpl", "go"}

func LoadPackageTypesFromFs(ctx context.Context, dir string) ([]*PackageWithTemplateFiles, error) {

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
		Mode: loadMode,
		Dir:  dir,
		Env:  append(os.Environ(), "GO111MODULE=on"),
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

	zerolog.Ctx(ctx).Debug().Msgf("loaded %d packages\n", len(pkgs))
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

	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found in directory: %s", dir)
	}

	if len(pkgs) == 1 && pkgs[0].ID == "./..." {
		return nil, errors.Errorf("no main module found for directory '%s', please ensure a parent directory with a go.mod file is provided", dir)
	}

	pkgWithTemplateFilesList := []*PackageWithTemplateFiles{}

	for _, pkg := range pkgs {

		pkgWithTemplateFiles := &PackageWithTemplateFiles{
			Package: pkg,
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

func ParseTree(name, text string) (map[string]*parse.Tree, error) {
	treeSet := make(map[string]*parse.Tree)
	t := parse.New(name)
	t.Mode = parse.ParseComments | parse.SkipFuncCheck
	_, err := t.Parse(text, "{{", "}}", treeSet)
	return treeSet, err
}

func ParseTemplate(ctx context.Context, fileName, content string) (*template.Template, error) {
	tmpl := template.New(fileName)
	tmpl.Tree = parse.New(fileName)
	tmpl.Mode = parse.ParseComments | parse.SkipFuncCheck

	treeSet, err := ParseTree(fileName, content)
	if err != nil {
		return nil, errors.Errorf("failed to parse template: %w", err)
	}

	for name, tree := range treeSet {
		if _, err := tmpl.AddParseTree(name, tree); err != nil {
			return nil, err
		}
	}

	return tmpl, nil
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
func AnalyzePackage(ctx context.Context, dir string) (*Registry, error) {
	if strings.HasSuffix(dir, ".tmpl") || strings.HasSuffix(dir, ".go") {
		dir = filepath.Dir(dir)
	}

	pkgWithTemplateFilesList, err := LoadPackageTypesFromFs(ctx, dir)
	if err != nil {
		return nil, errors.Errorf("failed to load package: %w", err)
	}

	pkgWithTemplatesList := []*PackageWithTemplates{}

	for _, pkgWithTemplateFiles := range pkgWithTemplateFilesList {
		pkgWithTemplates := &PackageWithTemplates{
			Package: pkgWithTemplateFiles.Package,
		}

		for fileName, content := range pkgWithTemplateFiles.TemplateFiles {
			tmpl, err := ParseTemplate(ctx, fileName, content)
			if err != nil {
				return nil, errors.Errorf("failed to parse template: %w", err)
			}
			pkgWithTemplates.Templates[fileName] = tmpl
		}

		pkgWithTemplatesList = append(pkgWithTemplatesList, pkgWithTemplates)
	}

	registry := NewRegistry(pkgWithTemplatesList)

	return registry, nil
}
