package position

// func TestIsAfterDot(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		content string
// 		offset  int
// 		want    bool
// 	}{
// 		{
// 			name:    "empty content",
// 			content: "",
// 			offset:  0,
// 			want:    false,
// 		},
// 		{
// 			name:    "after dot",
// 			content: "{{ .Name }}",
// 			offset:  4,
// 			want:    true,
// 		},
// 		{
// 			name:    "before dot",
// 			content: "{{ .Name }}",
// 			offset:  3,
// 			want:    false,
// 		},
// 		{
// 			name:    "not at dot",
// 			content: "{{ .Name }}",
// 			offset:  5,
// 			want:    false,
// 		},
// 		{
// 			name:    "after dot with space",
// 			content: "{{ . Name }}",
// 			offset:  4,
// 			want:    false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			doc := NewDocument(tt.content)
// 			pos := doc.NewBasicPosition(tt.content, tt.offset)
// 			assert.Equal(t, tt.want, IsAfterDot(pos), "IsAfterDot should match expected value")
// 		})
// 	}
// }

// func TestIsInTemplateAction(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		content string
// 		offset  int
// 		want    bool
// 	}{
// 		{
// 			name:    "empty content",
// 			content: "",
// 			offset:  0,
// 			want:    false,
// 		},
// 		{
// 			name:    "in template action",
// 			content: "{{ .Name }}",
// 			offset:  4,
// 			want:    true,
// 		},
// 		{
// 			name:    "outside template action",
// 			content: "text {{ .Name }} text",
// 			offset:  1,
// 			want:    false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			doc := NewDocument(tt.content)
// 			pos := doc.NewBasicPosition(tt.content, tt.offset)
// 			assert.Equal(t, tt.want, IsInTemplateAction(pos), "IsInTemplateAction should match expected value")
// 		})
// 	}
// }

// func TestGetExpressionBeforeDot(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		content string
// 		offset  int
// 		want    string
// 	}{
// 		{
// 			name:    "empty content",
// 			content: "",
// 			offset:  0,
// 			want:    "",
// 		},
// 		{
// 			name:    "simple field",
// 			content: "{{ .Name }}",
// 			offset:  4,
// 			want:    "",
// 		},
// 		{
// 			name:    "nested field",
// 			content: "{{ .User.Name }}",
// 			offset:  9,
// 			want:    "User",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			doc := NewDocument(tt.content)
// 			pos := doc.NewBasicPosition(tt.content, tt.offset)
// 			assert.Equal(t, tt.want, GetExpressionBeforeDot(pos), "GetExpressionBeforeDot should match expected value")
// 		})
// 	}
// }

// func TestIsDotCompletion(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		content string
// 		offset  int
// 		want    bool
// 	}{
// 		{
// 			name:    "empty content",
// 			content: "",
// 			offset:  0,
// 			want:    false,
// 		},
// 		{
// 			name:    "after dot in template",
// 			content: "{{ .Name }}",
// 			offset:  4,
// 			want:    true,
// 		},
// 		{
// 			name:    "after dot outside template",
// 			content: ".Name",
// 			offset:  2,
// 			want:    false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			doc := NewDocument(tt.content)
// 			pos := doc.NewBasicPosition(tt.content, tt.offset)
// 			assert.Equal(t, tt.want, IsDotCompletion(pos), "IsDotCompletion should match expected value")
// 		})
// 	}
// }
