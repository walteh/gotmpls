package ast

import (
	"go/types"
)

// TemplateMethodInfo represents information about a template method
type TemplateMethodInfo struct {
	Name       string
	Parameters []types.Type
	Results    []types.Type
}

// BuiltinTemplateMethods contains all the built-in Go template methods
var BuiltinTemplateMethods = map[string]*TemplateMethodInfo{
	"upper": {
		Name:       "upper",
		Parameters: []types.Type{types.Typ[types.String]},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"and": {
		Name:       "and",
		Parameters: []types.Type{types.Typ[types.Bool], types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"call": {
		Name:       "call",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"html": {
		Name:       "html",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"index": {
		Name:       "index",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"slice": {
		Name:       "slice",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.Typ[types.Int], types.Typ[types.Int]},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"js": {
		Name:       "js",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"len": {
		Name:       "len",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Int]},
	},
	"not": {
		Name:       "not",
		Parameters: []types.Type{types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"or": {
		Name:       "or",
		Parameters: []types.Type{types.Typ[types.Bool], types.Typ[types.Bool]},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"print": {
		Name:       "print",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"printf": {
		Name:       "printf",
		Parameters: []types.Type{types.Typ[types.String], types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"println": {
		Name:       "println",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.String]},
	},
	"urlquery": {
		Name:       "urlquery",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.NewInterfaceType(nil, nil)},
	},
	"eq": {
		Name:       "eq",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"ge": {
		Name:       "ge",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"gt": {
		Name:       "gt",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"le": {
		Name:       "le",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"lt": {
		Name:       "lt",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
	"ne": {
		Name:       "ne",
		Parameters: []types.Type{types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)},
		Results:    []types.Type{types.Typ[types.Bool]},
	},
}

// GetBuiltinMethod returns a built-in template method by name
func GetBuiltinMethod(name string) *TemplateMethodInfo {
	return BuiltinTemplateMethods[name]
}
