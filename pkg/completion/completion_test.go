package completion

// func TestGetCompletions(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		content  string
// 		offset   int
// 		typeInfo *bridge.TypeInfo
// 		want     []CompletionItem
// 	}{
// 		{
// 			name:    "empty content",
// 			content: "",
// 			offset:  0,
// 			typeInfo: &bridge.TypeInfo{
// 				Name: "User",
// 				Fields: map[string]*bridge.FieldInfo{
// 					"Name": {
// 						Name: "Name",
// 						Type: types.Typ[types.String],
// 					},
// 					"Age": {
// 						Name: "Age",
// 						Type: types.Typ[types.Int],
// 					},
// 				},
// 			},
// 			want: nil,
// 		},
// 		{
// 			name:    "field completion",
// 			content: "{{ . }}",
// 			offset:  5,
// 			typeInfo: &bridge.TypeInfo{
// 				Name: "User",
// 				Fields: map[string]*bridge.FieldInfo{
// 					"Name": {
// 						Name: "Name",
// 						Type: types.Typ[types.String],
// 					},
// 					"Age": {
// 						Name: "Age",
// 						Type: types.Typ[types.Int],
// 					},
// 				},
// 			},
// 			want: []CompletionItem{
// 				{
// 					Label:         "Name",
// 					Kind:          "field",
// 					Detail:        "string",
// 					Documentation: "Field of type: string",
// 				},
// 				{
// 					Label:         "Age",
// 					Kind:          "field",
// 					Detail:        "int",
// 					Documentation: "Field of type: int",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			doc := position.NewDocument(tt.content)
// 			pos := doc.NewBasicPosition(tt.content, tt.offset)
// 			fmt.Printf("Test %s: content=%q, offset=%d\n", tt.name, tt.content, tt.offset)
// 			fmt.Printf("IsAfterDot=%v, IsInTemplateAction=%v, IsDotCompletion=%v\n",
// 				position.IsAfterDot(pos),
// 				position.IsInTemplateAction(pos),
// 				position.IsDotCompletion(pos))
// 			got := GetCompletions(context.Background(), pos, tt.typeInfo)
// 			assert.Equal(t, tt.want, got, "GetCompletions should match expected value")
// 		})
// 	}
// }
