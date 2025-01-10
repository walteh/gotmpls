package ast

import (
	"context"
	"go/types"
	"path"
	"strings"

	"github.com/rs/zerolog"
	"github.com/walteh/go-tmpl-typer/pkg/bridge"
	"gitlab.com/tozd/go/errors"
)

// Registry manages Go package and type information
type Registry struct {
	// Types maps fully qualified type paths to their package information
	Types map[string]*types.Package
	// Error encountered during type resolution, if any
	Err error
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return &Registry{
		Types: make(map[string]*types.Package),
	}
}

// GetPackage returns a package by name
func (r *Registry) GetPackage(ctx context.Context, packageName string) (*types.Package, error) {
	zerolog.Ctx(ctx).Debug().Str("packageName", packageName).Interface("packages", r.Types).Msg("looking for package")

	// First, try to find an exact match
	if pkg, ok := r.Types[packageName]; ok {
		zerolog.Ctx(ctx).Debug().Str("package", packageName).Msg("found exact match")
		return pkg, nil
	}

	// Try to find by package name
	for pkgPath, pkg := range r.Types {
		if path.Base(pkgPath) == packageName {
			zerolog.Ctx(ctx).Debug().Str("packageName", packageName).Str("path", pkgPath).Msg("found by name")
			return pkg, nil
		}
	}

	// Try to find by path suffix
	for pkgPath, pkg := range r.Types {
		if strings.HasSuffix(pkgPath, "/"+packageName) {
			zerolog.Ctx(ctx).Debug().Str("packageName", packageName).Str("path", pkgPath).Msg("found by suffix")
			return pkg, nil
		}
	}

	zerolog.Ctx(ctx).Debug().Str("packageName", packageName).Msg("not found")
	return nil, errors.Errorf("package %s not found", packageName)
}

// AddPackage adds a package to the registry
func (r *Registry) AddPackage(pkg *types.Package) {
	r.Types[pkg.Path()] = pkg
}

// GetTypes retrieves all types from a package
func (r *Registry) GetTypes(ctx context.Context, pkgPath string) (map[string]types.Object, error) {
	pkg, err := r.GetPackage(ctx, pkgPath)
	if err != nil {
		return nil, err
	}

	types := make(map[string]types.Object)
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		types[name] = scope.Lookup(name)
	}

	return types, nil
}

// TypeExists checks if a type exists in the registry
func (r *Registry) TypeExists(typePath string) bool {
	// First try exact match
	if _, exists := r.Types[typePath]; exists {
		return true
	}

	// Try to find a package that ends with the requested name
	for pkgPath := range r.Types {
		if pkgPath == typePath || strings.HasSuffix(pkgPath, "/"+typePath) {
			return true
		}
	}

	return false
}

// AnalyzePackage analyzes a Go package and returns type information
func (r *Registry) AnalyzePackage(ctx context.Context, packageDir string) (*Registry, error) {
	// For now, just return the registry itself since we're not doing actual package analysis
	return r, nil
}

// ValidateType validates a type against package information
func (r *Registry) ValidateType(ctx context.Context, typePath string) (*bridge.TypeInfo, error) {
	// Split the type path into package and type name
	lastDot := strings.LastIndex(typePath, ".")
	if lastDot == -1 {
		return nil, errors.Errorf("invalid type path: %s", typePath)
	}

	pkgName, typeName := typePath[:lastDot], typePath[lastDot+1:]

	// Get the package from the registry
	pkg, err := r.GetPackage(ctx, pkgName)
	if err != nil {
		return nil, errors.Errorf("package not found in registry: %w", err)
	}

	// Find the type in the package scope
	obj := pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil, errors.Errorf("type %s not found in package %s", typeName, pkgName)
	}

	// Get the type information
	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("type %s is not a named type", typeName)
	}

	// Get the underlying struct type
	structType, ok := namedType.Underlying().(*types.Struct)
	if !ok {
		return nil, errors.Errorf("type %s is not a struct type", typeName)
	}

	// Create TypeInfo with fields
	typeInfo := &bridge.TypeInfo{
		Name:   typeName,
		Fields: make(map[string]*bridge.FieldInfo),
	}

	// Add fields to the type info
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		typeInfo.Fields[field.Name()] = &bridge.FieldInfo{
			Name: field.Name(),
			Type: field.Type(),
		}
	}

	// Add methods to the type info
	for i := 0; i < namedType.NumMethods(); i++ {
		method := namedType.Method(i)
		typeInfo.Fields[method.Name()] = &bridge.FieldInfo{
			Name: method.Name(),
			Type: method.Type(),
		}
	}

	return typeInfo, nil
}

// GetFieldType returns the type of a field in a struct type
func (r *Registry) GetFieldType(structType *types.Struct, fieldName string) (types.Type, error) {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Name() == fieldName {
			return field.Type(), nil
		}
	}
	return nil, errors.Errorf("field %s not found", fieldName)
}
