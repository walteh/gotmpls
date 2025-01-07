package ast

import (
	"context"
	"go/types"
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

// PackageAnalyzer is responsible for analyzing Go packages and extracting type information
type PackageAnalyzer interface {
	// AnalyzePackage analyzes a Go package and returns type information
	AnalyzePackage(ctx context.Context, packageDir string) (*PackageInfo, error)
}

// PackageInfo contains information about a Go package
type PackageInfo struct {
	Types map[string]*types.Package
	Err   error
}

// NewPackageInfo creates a new PackageInfo
func NewPackageInfo() *PackageInfo {
	return &PackageInfo{
		Types: make(map[string]*types.Package),
	}
}

// TypeExists checks if a type exists in the package
func (p *PackageInfo) TypeExists(typePath string) bool {
	_, exists := p.Types[typePath]
	return exists
}
