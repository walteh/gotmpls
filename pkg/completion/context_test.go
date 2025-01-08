package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCompletionContext(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		line      int
		character int
		want      *CompletionContext
	}{
		{
			name:      "empty content",
			content:   "",
			line:      1,
			character: 1,
			want:      &CompletionContext{},
		},
		{
			name:      "simple field",
			content:   ".Name",
			line:      1,
			character: 2,
			want: &CompletionContext{
				Line:       ".Name",
				Position:   2,
				InAction:   false,
				AfterDot:   true,
				Expression: "",
			},
		},
		{
			name:      "nested field",
			content:   ".User.Name",
			line:      1,
			character: 6,
			want: &CompletionContext{
				Line:       ".User.Name",
				Position:   6,
				InAction:   false,
				AfterDot:   true,
				Expression: "User",
			},
		},
		{
			name:      "before dot",
			content:   ".User.Name",
			line:      1,
			character: 5,
			want: &CompletionContext{
				Line:       ".User.Name",
				Position:   5,
				InAction:   false,
				AfterDot:   false,
				Expression: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCompletionContext(tt.content, tt.line, tt.character)
			assert.Equal(t, tt.want.Line, got.Line, "line is not correct")
			assert.Equal(t, tt.want.Position, got.Position, "position is not correct")
			assert.Equal(t, tt.want.InAction, got.InAction, "in action is not correct")
			assert.Equal(t, tt.want.AfterDot, got.AfterDot, "after dot is not correct")
			assert.Equal(t, tt.want.Expression, got.Expression, "expression is not correct")
		})
	}
}

func TestCompletionContext_IsInTemplateAction(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		position int
		want     bool
	}{
		{
			name:     "empty line",
			line:     "",
			position: 0,
			want:     false,
		},
		{
			name:     "simple field",
			line:     ".Name",
			position: 2,
			want:     false,
		},
		{
			name:     "nested field",
			line:     ".User.Name",
			position: 6,
			want:     false,
		},
		{
			name:     "before dot",
			line:     ".User.Name",
			position: 5,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &CompletionContext{
				Line:     tt.line,
				Position: tt.position,
			}
			got := ctx.isInTemplateAction()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletionContext_IsDotCompletion(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		position int
		want     bool
	}{
		{
			name:     "empty line",
			line:     "",
			position: 0,
			want:     false,
		},
		{
			name:     "after dot",
			line:     "{{ .Name.",
			position: 9,
			want:     true,
		},
		{
			name:     "after dot with space",
			line:     "{{ .Name. ",
			position: 10,
			want:     false,
		},
		{
			name:     "before dot",
			line:     "{{ .Name.",
			position: 8,
			want:     false,
		},
		{
			name:     "no dot",
			line:     "{{ .Name",
			position: 8,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &CompletionContext{
				Line:     tt.line,
				Position: tt.position,
			}
			got := ctx.isDotCompletion()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletionContext_GetExpressionBeforeDot(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		position int
		want     string
	}{
		{
			name:     "empty line",
			line:     "",
			position: 0,
			want:     "",
		},
		{
			name:     "simple field",
			line:     "{{ .Name.",
			position: 9,
			want:     "Name",
		},
		{
			name:     "nested field",
			line:     "{{ .User.Name.",
			position: 14,
			want:     "Name",
		},
		{
			name:     "with spaces",
			line:     "{{   .User  .  Name.",
			position: 19,
			want:     "User",
		},
		{
			name:     "position before dot",
			line:     "{{ .User.",
			position: 9,
			want:     "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &CompletionContext{
				Line:     tt.line,
				Position: tt.position,
			}
			got := ctx.getExpressionBeforeDot()
			assert.Equal(t, tt.want, got)
		})
	}
}
