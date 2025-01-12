package ast

import (
	"fmt"
	"go/types"
	"reflect"
)

// TemplateMethodInfo represents information about a template method
type TemplateMethodInfo struct {
	Name       string
	Parameters []types.Type
	Results    []types.Type
}

// convertType converts a reflect.Type to a types.Type
func convertType(t reflect.Type) types.Type {
	fmt.Println("t", t.Kind(), t.String())
	switch t.Kind() {
	case reflect.Bool:
		return types.Typ[types.Bool]
	case reflect.Int:
		return types.Typ[types.Int]
	case reflect.String:
		return types.Typ[types.String]
	default:
		return types.NewInterfaceType(nil, nil)
	}
}

// generateBuiltinTemplateMethods generates the BuiltinTemplateMethods map using reflection
func generateBuiltinTemplateMethods() map[string]*TemplateMethodInfo {
	methods := make(map[string]*TemplateMethodInfo)

	// Combine both builtin and extra functions
	allFuncs := Builtins()
	for name, fn := range Extras() {
		allFuncs[name] = fn
	}

	for name, fn := range allFuncs {
		fnType := reflect.TypeOf(fn)
		if fnType == nil {
			continue
		}

		if name == "and" {
			fmt.Println("hello")
		}

		info := &TemplateMethodInfo{
			Name:       name,
			Parameters: make([]types.Type, fnType.NumIn()),
			Results:    make([]types.Type, fnType.NumOut()),
		}

		// Convert parameter types
		for i := 0; i < fnType.NumIn(); i++ {
			info.Parameters[i] = convertType(fnType.In(i))
		}

		// Convert result types
		for i := 0; i < fnType.NumOut(); i++ {
			info.Results[i] = convertType(fnType.Out(i))
		}

		methods[name] = info
	}

	return methods
}

// BuiltinTemplateMethods contains all the built-in Go template methods
var BuiltinTemplateMethods = generateBuiltinTemplateMethods()

// GetBuiltinMethod returns a built-in template method by name
func GetBuiltinMethod(name string) *TemplateMethodInfo {
	return BuiltinTemplateMethods[name]
}
