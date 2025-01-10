package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCompletionContext_AfterDot(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		wantDot   bool
	}{
		{
			name:      "empty content",
			content:   "",
			line:      1,
			character: 1,
			wantDot:   false,
		},
		{
			name:      "after dot",
			content:   "{{ .Name }}",
			line:      1,
			character: 4,
			wantDot:   true,
		},
		{
			name:      "before dot",
			content:   "{{ .Name }}",
			line:      1,
			character: 3,
			wantDot:   false,
		},
		{
			name:      "not at dot",
			content:   "{{ .Name }}",
			line:      1,
			character: 5,
			wantDot:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.wantDot, ctx.AfterDot, "AfterDot should match expected value")
		})
	}
}

func TestCompletionContext_IsInTemplateAction(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      bool
	}{
		{
			name:      "empty content",
			content:   "",
			line:      1,
			character: 1,
			want:      false,
		},
		{
			name:      "in template action",
			content:   "{{ .Name }}",
			line:      1,
			character: 4,
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

func TestCompletionContext_GetExpressionBeforeDot(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      string
	}{
		{
			name:      "empty content",
			content:   "",
			line:      1,
			character: 1,
			want:      "",
		},
		{
			name:      "simple field",
			content:   "{{ .Name }}",
			line:      1,
			character: 4,
			want:      "",
		},
		{
			name:      "nested field",
			content:   "{{ .User.Name }}",
			line:      1,
			character: 9,
			want:      "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.want, ctx.GetExpressionBeforeDot(), "GetExpressionBeforeDot should match expected value")
		})
	}
}

func TestCompletionContext_IsDotCompletion(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      bool
	}{
		{
			name:      "empty content",
			content:   "",
			line:      1,
			character: 1,
			want:      false,
		},
		{
			name:      "after dot in template",
			content:   "{{ .Name }}",
			line:      1,
			character: 4,
			want:      true,
		},
		{
			name:      "after dot outside template",
			content:   ".Name",
			line:      1,
			character: 2,
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
