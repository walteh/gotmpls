package ast

import (
	"context"
	"go/types"
)

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
