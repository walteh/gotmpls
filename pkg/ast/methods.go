package ast

import (
	"go/types"
	"reflect"
	"strings"

	"github.com/walteh/gotmpls/pkg/astreflect"
	"github.com/walteh/gotmpls/pkg/std/text/template"
)

func Extras() template.FuncMap {
	return template.FuncMap{
		"upper":      strings.ToUpper,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"join":       strings.Join,
		"trim":       strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
	}
}

// TemplateMethodInfo represents information about a template method
type TemplateMethodInfo struct {
	Name       string
	Parameters []types.Type
	Results    []types.Type
}

// generateBuiltinTemplateMethods generates the BuiltinTemplateMethods map using reflection
func generateBuiltinTemplateMethods() map[string]*TemplateMethodInfo {
	methods := make(map[string]*TemplateMethodInfo)

	// Combine both builtin and extra functions
	allFuncs := template.BuiltinsExported()
	for name, fn := range Extras() {
		allFuncs[name] = fn
	}

	for name, fn := range allFuncs {
		fnType := reflect.TypeOf(fn)
		if fnType == nil {
			continue
		}

		info := &TemplateMethodInfo{
			Name:       name,
			Parameters: make([]types.Type, fnType.NumIn()),
			Results:    make([]types.Type, fnType.NumOut()),
		}

		// Convert parameter types
		for i := 0; i < fnType.NumIn(); i++ {
			info.Parameters[i] = astreflect.Reflect2AST(fnType.In(i))
		}

		// Convert result types
		for i := 0; i < fnType.NumOut(); i++ {
			info.Results[i] = astreflect.Reflect2AST(fnType.Out(i))
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
