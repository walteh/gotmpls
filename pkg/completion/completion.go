package completion

// // CompletionItem represents a single completion suggestion
// type CompletionItem struct {
// 	Label         string `json:"label"`
// 	Kind          string `json:"kind"`
// 	Detail        string `json:"detail,omitempty"`
// 	Documentation string `json:"documentation,omitempty"`
// }

// // GetCompletions returns completion items for the given position
// func GetCompletions(ctx context.Context, pos position.RawPosition, typeInfo *bridge.TypeInfo) []CompletionItem {
// 	var items []CompletionItem

// 	if position.IsDotCompletion(pos) {
// 		// Get the expression before the dot to determine the type
// 		expr := position.GetExpressionBeforeDot(pos)
// 		if expr == "" {
// 			// If no expression before dot, we're at the root
// 			// Add all fields from the root type
// 			for name, field := range typeInfo.Fields {
// 				items = append(items, CompletionItem{
// 					Label:         name,
// 					Kind:          "field",
// 					Detail:        field.Type.String(),
// 					Documentation: "Field of type: " + field.Type.String(),
// 				})
// 			}
// 		} else {
// 			// Get field completions for the type of the expression
// 			field, err := bridge.ValidateField(ctx, typeInfo, pos)
// 			if err == nil && field != nil {
// 				// Get the underlying struct type if this is a struct field
// 				if structType, ok := field.Type.Underlying().(*types.Struct); ok {
// 					// Add all fields from the struct
// 					for i := 0; i < structType.NumFields(); i++ {
// 						f := structType.Field(i)
// 						items = append(items, CompletionItem{
// 							Label:         f.Name(),
// 							Kind:          "field",
// 							Detail:        f.Type().String(),
// 							Documentation: "Field of type: " + f.Type().String(),
// 						})
// 					}
// 				}
// 			}
// 		}
// 	}

// 	return items
// }
