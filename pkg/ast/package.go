package ast

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/tools/go/packages"
)

// GoPackageInfo holds information about a Go package
type GoPackageInfo struct {
	// Package is the loaded Go package
	Package *types.Package
	// TypesInfo contains type information for the package
	TypesInfo *types.Info
	// Files contains the AST for each file in the package
	Files []*ast.File
}

// PackageLoader handles loading and parsing Go packages
type PackageLoader interface {
	// LoadPackage loads a package by its import path
	LoadPackage(ctx context.Context, importPath string) (*GoPackageInfo, error)
	// ResolveType resolves a type by its full path (e.g., "pkg/path.TypeName")
	ResolveType(ctx context.Context, typePath string) (types.Type, error)
}

// DefaultPackageLoader is the default implementation of PackageLoader
type DefaultPackageLoader struct {
	// fset is the file set used for parsing
	fset *token.FileSet
	// cache holds loaded packages
	cache map[string]*GoPackageInfo
}

// NewDefaultPackageLoader creates a new DefaultPackageLoader
func NewDefaultPackageLoader() *DefaultPackageLoader {
	return &DefaultPackageLoader{
		fset:  token.NewFileSet(),
		cache: make(map[string]*GoPackageInfo),
	}
}

// LoadPackage implements PackageLoader
func (l *DefaultPackageLoader) LoadPackage(ctx context.Context, importPath string) (*GoPackageInfo, error) {
	// Check cache first
	if pkg, ok := l.cache[importPath]; ok {
		return pkg, nil
	}

	// Configure package loading
	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Context: ctx,
	}

	// Load the package
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, errors.Errorf("failed to load package %s: %w", importPath, err)
	}
	if len(pkgs) == 0 {
		return nil, errors.Errorf("no packages found for import path %s", importPath)
	}
	if len(pkgs[0].Errors) > 0 {
		return nil, errors.Errorf("errors loading package %s: %v", importPath, pkgs[0].Errors)
	}

	// Create package info
	info := &GoPackageInfo{
		Package:   pkgs[0].Types,
		TypesInfo: pkgs[0].TypesInfo,
		Files:     pkgs[0].Syntax,
	}

	// Cache the result
	l.cache[importPath] = info

	return info, nil
}

// ResolveType implements PackageLoader
func (l *DefaultPackageLoader) ResolveType(ctx context.Context, typePath string) (types.Type, error) {
	// Split package path and type name
	dir, typeName := filepath.Split(typePath)
	if dir == "" || typeName == "" {
		return nil, errors.Errorf("invalid type path: %s", typePath)
	}

	// Remove trailing slash from dir
	dir = filepath.Clean(dir)

	// Load the package
	pkgInfo, err := l.LoadPackage(ctx, dir)
	if err != nil {
		return nil, err
	}

	// Look up the type
	obj := pkgInfo.Package.Scope().Lookup(typeName)
	if obj == nil {
		return nil, errors.Errorf("type %s not found in package %s", typeName, dir)
	}

	typeObj, ok := obj.(*types.TypeName)
	if !ok {
		return nil, errors.Errorf("%s is not a type", typePath)
	}

	return typeObj.Type(), nil
}
