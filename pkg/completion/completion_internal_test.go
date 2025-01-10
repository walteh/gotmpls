package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExpressionBeforeDot(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      string
	}{
		{
			name:      "empty line",
			content:   "",
			line:      1,
			character: 1,
			want:      "",
		},
		{
			name:      "simple field",
			content:   "{{ .Name }}",
			line:      1,
			character: 5,
			want:      "",
		},
		{
			name:      "nested field",
			content:   "{{ .User.Name }}",
			line:      1,
			character: 9,
			want:      "User",
		},
		{
			name:      "before dot",
			content:   "{{ .User.Name }}",
			line:      1,
			character: 3,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.want, ctx.GetExpressionBeforeDot(), "GetExpressionBeforeDot should match expected value")
		})
	}
}

func TestIsInTemplateAction(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      bool
	}{
		{
			name:      "empty line",
			content:   "",
			line:      1,
			character: 1,
			want:      false,
		},
		{
			name:      "in template action",
			content:   "{{ .Name }}",
			line:      1,
			character: 5,
			want:      true,
		},
		{
			name:      "outside template action",
			content:   "text {{ .Name }} text",
			line:      1,
			character: 1,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.want, ctx.IsInTemplateAction(), "IsInTemplateAction should match expected value")
		})
	}
}

func TestIsDotCompletion(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      bool
	}{
		{
			name:      "empty line",
			content:   "",
			line:      1,
			character: 1,
			want:      false,
		},
		{
			name:      "after dot",
			content:   "{{ .Name }}",
			line:      1,
			character: 5,
			want:      true,
		},
		{
			name:      "after dot with space",
			content:   "{{ . Name }}",
			line:      1,
			character: 5,
			want:      false,
		},
		{
			name:      "before dot",
			content:   "{{ .Name }}",
			line:      1,
			character: 3,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.want, ctx.IsDotCompletion(), "IsDotCompletion should match expected value")
		})
	}
}
