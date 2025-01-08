package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExpressionBeforeDot(t *testing.T) {
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
			line:     ".Name.",
			position: 6,
			want:     "Name",
		},
		{
			name:     "nested field",
			line:     ".User.Name.",
			position: 11,
			want:     "Name",
		},
		{
			name:     "with spaces",
			line:     ".User  .  Name.",
			position: 14,
			want:     "User",
		},
		{
			name:     "position before dot",
			line:     ".User.",
			position: 6,
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

func TestIsInTemplateAction(t *testing.T) {
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

func TestIsDotCompletion(t *testing.T) {
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
			line:     ".Name.",
			position: 6,
			want:     true,
		},
		{
			name:     "after dot with space",
			line:     ".Name. ",
			position: 7,
			want:     false,
		},
		{
			name:     "before dot",
			line:     ".Name.",
			position: 5,
			want:     false,
		},
		{
			name:     "no dot",
			line:     ".Name",
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
			got := ctx.isDotCompletion()
			assert.Equal(t, tt.want, got)
		})
	}
}
