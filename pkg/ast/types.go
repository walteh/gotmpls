package ast

import (
	"go/types"
)

// TypeInfo represents information about a Go type
type TypeInfo struct {
	Name   string
	Fields map[string]*FieldInfo
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name     string
	Type     types.Type
	FullName string
}
