package position_test

import (
	"testing"

	"github.com/walteh/go-tmpl-typer/pkg/position"
)

func TestGetLineAndColumn(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      position.RawPosition
		wantLine int
		wantCol  int
	}{
		{
			name:     "empty text",
			text:     "",
			pos:      position.RawPosition{Offset: 0},
			wantLine: 0,
			wantCol:  0,
		},
		{
			name:     "single line, first position",
			text:     "Hello, World! ",
			pos:      position.RawPosition{Offset: 2},
			wantLine: 0,
			wantCol:  2,
		},
		{
			name:     "single line, middle position",
			text:     "Hello, World!",
			pos:      position.RawPosition{Offset: 7},
			wantLine: 0,
			wantCol:  7,
		},
		{
			name:     "multiple lines, first line",
			text:     "Hello\nWorld\nTest",
			pos:      position.RawPosition{Offset: 3},
			wantLine: 0,
			wantCol:  3,
		},
		{
			name:     "multiple lines, second line",
			text:     "Hello\nWorld\nTest zzz",
			pos:      position.RawPosition{Offset: 8},
			wantLine: 1,
			wantCol:  2,
		},
		{
			name:     "multiple lines with varying lengths",
			text:     "Hello, World!\nThis is a test\nShort\nLonger line here zzz",
			pos:      position.RawPosition{Offset: 16},
			wantLine: 1,
			wantCol:  2,
		},
		{
			name:     "broken example",
			text:     "{{- /*gotype: test.Person*/ -}}\nAddress:\n  Street: {{.Address.Street}}",
			pos:      position.RawPosition{Offset: 61},
			wantLine: 2,
			wantCol:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotCol := tt.pos.GetLineAndColumn(tt.text)
			if gotLine != tt.wantLine || gotCol != tt.wantCol {
				t.Errorf("GetLineAndColumn() = (%v, %v), want (%v, %v)", gotLine, gotCol, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func TestRawPosition(t *testing.T) {
	tests := []struct {
		name     string
		pos      position.RawPosition
		wantText string
		wantID   string
	}{
		{
			name: "basic position",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			wantText: "test",
			wantID:   "test@10",
		},
		{
			name: "empty text",
			pos: position.RawPosition{
				Text:   "",
				Offset: 0,
			},
			wantText: "",
			wantID:   "@0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos.Text; got != tt.wantText {
				t.Errorf("RawPosition.Text = %v, want %v", got, tt.wantText)
			}
			if got := tt.pos.ID(); got != tt.wantID {
				t.Errorf("RawPosition.ID() = %v, want %v", got, tt.wantID)
			}
		})
	}
}

func TestHasRangeOverlap(t *testing.T) {
	tests := []struct {
		name      string
		pos       position.RawPosition
		start     position.RawPosition
		wantMatch bool
	}{
		{
			name: "overlapping ranges",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 5,
			},
			start: position.RawPosition{
				Text:   "testing",
				Offset: 3,
			},
			wantMatch: true,
		},
		{
			name: "non-overlapping ranges",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 0,
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos.HasRangeOverlapWith(tt.start); got != tt.wantMatch {
				t.Errorf("HasRangeOverlapWith() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestRawPosition_HasRangeOverlapWith(t *testing.T) {
	tests := []struct {
		name     string
		pos      position.RawPosition
		start    position.RawPosition
		expected bool
	}{
		{
			name: "exact match",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: true,
		},
		{
			name: "complete containment",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 11,
			},
			start: position.RawPosition{
				Text:   "testing",
				Offset: 10,
			},
			expected: true,
		},
		{
			name: "partial overlap at start",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 8,
			},
			start: position.RawPosition{
				Text:   "testing",
				Offset: 10,
			},
			expected: true,
		},
		{
			name: "partial overlap at end",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 14,
			},
			start: position.RawPosition{
				Text:   "testing",
				Offset: 10,
			},
			expected: true,
		},
		{
			name: "no overlap - before",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 5,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: false,
		},
		{
			name: "no overlap - after",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 15,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: false,
		},
		{
			name: "adjacent - no overlap",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 14,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: false,
		},
		{
			name: "zero-length text",
			pos: position.RawPosition{
				Text:   "",
				Offset: 10,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: true,
		},
		{
			name: "both zero-length text",
			pos: position.RawPosition{
				Text:   "",
				Offset: 10,
			},
			start: position.RawPosition{
				Text:   "",
				Offset: 10,
			},
			expected: true,
		},

		{
			name: "last_is_zero_length",
			pos: position.RawPosition{
				Text:   "hooplah",
				Offset: 46,
			},
			start: position.RawPosition{
				Text:   "",
				Offset: 49,
			},
			expected: true,
		},
		{
			name: "first_is_zero_length",
			pos: position.RawPosition{
				Text:   "",
				Offset: 49,
			},
			start: position.RawPosition{
				Text:   "hooplah",
				Offset: 46,
			},
			expected: true,
		},
		{
			name: "last_is_one_length",
			pos: position.RawPosition{
				Text:   "hooplah",
				Offset: 46,
			},
			start: position.RawPosition{
				Text:   "f",
				Offset: 49,
			},
			expected: true,
		},
		{
			name: "first_is_one_length",
			pos: position.RawPosition{
				Text:   "f",
				Offset: 49,
			},
			start: position.RawPosition{
				Text:   "hooplah",
				Offset: 46,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.HasRangeOverlapWith(tt.start)
			if result != tt.expected {
				t.Errorf("HasRangeOverlapWith() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewRawPositionFromLineAndColumn(t *testing.T) {
	tests := []struct {
		name     string
		line     int
		col      int
		text     string
		fileText string
		want     position.RawPosition
	}{
		{
			name:     "first line, first character",
			line:     0,
			col:      0,
			text:     "H",
			fileText: "Hello\nWorld\nTest",
			want: position.RawPosition{
				Text:   "H",
				Offset: 0,
			},
		},
		{
			name:     "first line, middle character",
			line:     0,
			col:      2,
			text:     "l",
			fileText: "Hello\nWorld\nTest",
			want: position.RawPosition{
				Text:   "l",
				Offset: 2,
			},
		},
		{
			name:     "second line, first character",
			line:     1,
			col:      0,
			text:     "W",
			fileText: "Hello\nWorld\nTest",
			want: position.RawPosition{
				Text:   "W",
				Offset: 6,
			},
		},
		{
			name:     "last line, last character",
			line:     2,
			col:      3,
			text:     "t",
			fileText: "Hello\nWorld\nTest",
			want: position.RawPosition{
				Text:   "t",
				Offset: 15,
			},
		},
		{
			name:     "empty line",
			line:     1,
			col:      0,
			text:     "",
			fileText: "Hello\n\nTest",
			want: position.RawPosition{
				Text:   "",
				Offset: 6,
			},
		},
		{
			name:     "line with spaces",
			line:     1,
			col:      2,
			text:     "x",
			fileText: "Hello\n  World\nTest",
			want: position.RawPosition{
				Text:   "x",
				Offset: 8,
			},
		},
		{
			name:     "template example",
			line:     2,
			col:      20,
			text:     "t",
			fileText: "{{- /*gotype: test.Person*/ -}}\nAddress:\n  Street: {{.Address.Street}}",
			want: position.RawPosition{
				Text:   "t",
				Offset: 61,
			},
		},
		{
			name: "template example",
			line: 2,
			col:  12,
			text: ".",
			fileText: `{{- /*gotype: test.Person*/ -}}
Address:
  Street: {{.Address.Street}}`,
			want: position.RawPosition{
				Text:   "",
				Offset: 61,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := position.NewRawPositionFromLineAndColumn(tt.line, tt.col, tt.text, tt.fileText)
			if got.Offset != tt.want.Offset {
				t.Errorf("NewRawPositionFromLineAndColumn() Offset = %v, want %v", got.Offset, tt.want.Offset)
			}
			if got.Text != tt.want.Text {
				t.Errorf("NewRawPositionFromLineAndColumn() Text = %v, want %v", got.Text, tt.want.Text)
			}
		})
	}
}
