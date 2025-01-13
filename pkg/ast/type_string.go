package ast

import (
	"fmt"
	"go/types"
	"strings"
)

// NestedMultiLineTypeString returns a multi-line string representation of the type hierarchy
func (f *FieldInfo) NestedMultiLineTypeString() string {
	if f == nil {
		return ""
	}

	// For methods, return the method signature
	if f.Type.Func != nil {
		sig := f.Type.Type().(*types.Signature)
		out := "```go\nfunc "

		// Add receiver if present
		if sig.Recv() != nil {
			recvType := sig.Recv().Type().String()
			// Strip package name from receiver type (e.g., "*test.Person" -> "*Person")
			if idx := strings.LastIndex(recvType, "."); idx != -1 {
				recvType = "*" + recvType[idx+1:]
			}
			out += "(" + recvType + ") "
		}

		out += f.Type.Func.Name() + "("

		// Parameters
		params := make([]string, sig.Params().Len())
		for i := 0; i < sig.Params().Len(); i++ {
			params[i] = sig.Params().At(i).Type().String()
		}
		out += strings.Join(params, ", ")
		out += ")"

		// Results
		if sig.Results().Len() > 0 {
			results := make([]string, sig.Results().Len())
			for i := 0; i < sig.Results().Len(); i++ {
				results[i] = sig.Results().At(i).Type().String()
			}
			out += " (" + strings.Join(results, ", ") + ")"
		}
		out += "\n```"
		return out
	}

	// Start building the type hierarchy
	var parents []*TypeHintDefinition
	current := f.Parent
	for current != nil {
		parents = append(parents, current)
		current = current.MyFieldInfo.Parent
	}

	// Reverse the parents slice to start from the root
	for i := 0; i < len(parents)/2; i++ {
		parents[i], parents[len(parents)-1-i] = parents[len(parents)-1-i], parents[i]
	}

	out := "```go\n"
	if len(parents) > 0 {
		out += fmt.Sprintf("type %s struct {\n", parents[0].MyFieldInfo.Name)
		for i := 1; i < len(parents); i++ {
			out += strings.Repeat("\t", i)
			out += fmt.Sprintf("%s struct {\n", parents[i].MyFieldInfo.Name)
		}

		// If the field type is a struct, format it with proper indentation
		if structType, ok := f.Type.Type().Underlying().(*types.Struct); ok {
			out += strings.Repeat("\t", len(parents))
			out += fmt.Sprintf("%s struct {\n", f.Name)
			for i := 0; i < structType.NumFields(); i++ {
				field := structType.Field(i)
				out += strings.Repeat("\t", len(parents)+1)
				out += fmt.Sprintf("%s %s\n", field.Name(), field.Type().String())
			}
			out += strings.Repeat("\t", len(parents)) + "}\n"
		} else {
			out += strings.Repeat("\t", len(parents))
			if f.Name != "" {
				out += fmt.Sprintf("%s %s\n", f.Name, f.Type.String())
			} else if f.Type.Var != nil {
				out += fmt.Sprintf("%s %s\n", f.Type.Var.Name(), f.Type.String())
			} else {
				out += fmt.Sprintf("%s\n", f.Type.String())
			}
		}

		for i := len(parents) - 1; i >= 0; i-- {
			out += strings.Repeat("\t", i)
			out += "}\n"
		}
	} else if f.Type.Var != nil {
		if f.Name != "" {
			out += fmt.Sprintf("%s %s\n", f.Name, f.Type.String())
		} else {
			out += fmt.Sprintf("%s\n", f.Type.String())
		}
	}
	out += "```"
	return out
}
