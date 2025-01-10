package ast

import (
	"context"
	"go/types"
	"path"
	"strings"

	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

// Node represents a template AST node
type Node interface {
	// Position returns the start and end position of the node
	Position() (start, end Position)
}

// Position represents a position in the template source
type Position struct {
	Line   int
	Column int
}

// TemplateNode represents a complete template
type TemplateNode struct {
	TypeHint    *TypeHintNode
	Definitions []*DefinitionNode
	Pos         Position
}

func NewTemplateNode(pos Position) *TemplateNode {
	return &TemplateNode{
		Pos: pos,
	}
}

func (n *TemplateNode) Position() (start, end Position) {
	start = n.Pos
	if len(n.Definitions) > 0 {
		_, end = n.Definitions[len(n.Definitions)-1].Position()
	} else {
		end = start
	}
	return
}

// TypeHintNode represents a gotype hint comment
type TypeHintNode struct {
	TypePath string
	Pos      Position
}

func NewTypeHintNode(typePath string, pos Position) *TypeHintNode {
	return &TypeHintNode{
		TypePath: typePath,
		Pos:      pos,
	}
}

func (n *TypeHintNode) Position() (start, end Position) {
	return n.Pos, n.Pos
}

// DefinitionNode represents a template definition block
type DefinitionNode struct {
	Name   string
	Body   []Node
	Pos    Position
	EndPos Position
}

func NewDefinitionNode(name string, pos, endPos Position) *DefinitionNode {
	return &DefinitionNode{
		Name:   name,
		Pos:    pos,
		EndPos: endPos,
	}
}

func (n *DefinitionNode) Position() (start, end Position) {
	return n.Pos, n.EndPos
}

// ActionNode represents a template action (anything between {{ and }})
type ActionNode struct {
	Pipeline []Node
	Pos      Position
	EndPos   Position
}

func NewActionNode(pos, endPos Position) *ActionNode {
	return &ActionNode{
		Pos:    pos,
		EndPos: endPos,
	}
}

func (n *ActionNode) Position() (start, end Position) {
	return n.Pos, n.EndPos
}

// VariableNode represents a variable reference (e.g., .Name)
type VariableNode struct {
	Name string
	Pos  Position
}

func NewVariableNode(name string, pos Position) *VariableNode {
	return &VariableNode{
		Name: name,
		Pos:  pos,
	}
}

func (n *VariableNode) Position() (start, end Position) {
	return n.Pos, Position{Line: n.Pos.Line, Column: n.Pos.Column + len(n.Name)}
}

// FunctionNode represents a function call
type FunctionNode struct {
	Name      string
	Arguments []Node
	Pos       Position
}

func NewFunctionNode(name string, pos Position) *FunctionNode {
	return &FunctionNode{
		Name: name,
		Pos:  pos,
	}
}

func (n *FunctionNode) Position() (start, end Position) {
	start = n.Pos
	if len(n.Arguments) > 0 {
		_, end = n.Arguments[len(n.Arguments)-1].Position()
	} else {
		end = Position{Line: start.Line, Column: start.Column + len(n.Name)}
	}
	return
}

// PackageAnalyzer is responsible for analyzing Go packages
type PackageAnalyzer interface {
	// AnalyzePackage analyzes a Go package and returns type information
	AnalyzePackage(ctx context.Context, packageDir string) (*TypeRegistry, error)
	// GetPackage returns a package by name
	GetPackage(ctx context.Context, packageName string) (*types.Package, error)
	// GetTypes returns all types in a package
	GetTypes(ctx context.Context, pkgPath string) (map[string]types.Object, error)
}

// TypeRegistry implements PackageAnalyzer
type TypeRegistry struct {
	// Types maps fully qualified type paths to their package information
	Types map[string]*types.Package
	// Error encountered during type resolution, if any
	Err error
}

func (r *TypeRegistry) GetPackage(ctx context.Context, packageName string) (*types.Package, error) {
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
func (r *TypeRegistry) AddPackage(pkg *types.Package) {
	r.Types[pkg.Path()] = pkg
}

// GetTypes retrieves all types from a package
func (r *TypeRegistry) GetTypes(ctx context.Context, pkgPath string) (map[string]types.Object, error) {
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

// NewTypeRegistry creates a new TypeRegistry
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		Types: make(map[string]*types.Package),
	}
}

// TypeExists checks if a type exists in the registry
func (p *TypeRegistry) TypeExists(typePath string) bool {
	// First try exact match
	if _, exists := p.Types[typePath]; exists {
		return true
	}

	// Try to find a package that ends with the requested name
	for pkgPath := range p.Types {
		if pkgPath == typePath || strings.HasSuffix(pkgPath, "/"+typePath) {
			return true
		}
	}

	return false
}

func (r *TypeRegistry) AnalyzePackage(ctx context.Context, packageDir string) (*TypeRegistry, error) {
	// For now, just return the registry itself since we're not doing actual package analysis
	return r, nil
}

// TypeInfo represents information about a Go type
type TypeInfo struct {
	Name   string
	Fields map[string]*FieldInfo
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name string
	Type types.Type
}

// ValidateType validates a type against package information
func (r *TypeRegistry) ValidateType(ctx context.Context, typePath string) (*TypeInfo, error) {
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
	typeInfo := &TypeInfo{
		Name:   typeName,
		Fields: make(map[string]*FieldInfo),
	}

	// Add fields to the type info
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		typeInfo.Fields[field.Name()] = &FieldInfo{
			Name: field.Name(),
			Type: field.Type(),
		}
	}

	// Add methods to the type info
	for i := 0; i < namedType.NumMethods(); i++ {
		method := namedType.Method(i)
		typeInfo.Fields[method.Name()] = &FieldInfo{
			Name: method.Name(),
			Type: method.Type(),
		}
	}

	return typeInfo, nil
}

// GetFieldType returns the type of a field in a struct type
func (r *TypeRegistry) GetFieldType(structType *types.Struct, fieldName string) (types.Type, error) {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Name() == fieldName {
			return field.Type(), nil
		}
	}
	return nil, errors.Errorf("field %s not found", fieldName)
}
