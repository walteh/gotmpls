package position_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			wantCol:  3,
		},
		{
			name:     "single line, middle position",
			text:     "Hello, World!",
			pos:      position.RawPosition{Offset: 7},
			wantLine: 0,
			wantCol:  8,
		},
		{
			name:     "multiple lines, first line",
			text:     "Hello\nWorld\nTest",
			pos:      position.RawPosition{Offset: 3},
			wantLine: 0,
			wantCol:  4,
		},
		{
			name:     "multiple lines, second line",
			text:     "Hello\nWorld\nTest zzz",
			pos:      position.RawPosition{Offset: 8},
			wantLine: 1,
			wantCol:  3,
		},
		{
			name:     "multiple lines with varying lengths",
			text:     "Hello, World!\nThis is a test\nShort\nLonger line here zzz",
			pos:      position.RawPosition{Offset: 16},
			wantLine: 1,
			wantCol:  3,
		},
		{
			name:     "template example",
			text:     "{{- /*gotype: test.Person*/ -}}\nAddress:\n  Street: {{.Address.Street}}",
			pos:      position.RawPosition{Offset: 61},
			wantLine: 2,
			wantCol:  21,
		},
		{
			name:     "empty lines between text",
			text:     "First\n\n\nLast",
			pos:      position.RawPosition{Offset: 7},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "position at newline",
			text:     "Hello\nWorld",
			pos:      position.RawPosition{Offset: 5},
			wantLine: 0,
			wantCol:  6,
		},
		{
			name:     "last position in file",
			text:     "Hello\nWorld",
			pos:      position.RawPosition{Offset: 10},
			wantLine: 1,
			wantCol:  5,
		},
		{
			name:     "position after last newline",
			text:     "Hello\nWorld\n",
			pos:      position.RawPosition{Offset: 11},
			wantLine: 1,
			wantCol:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotCol := tt.pos.GetLineAndColumn(tt.text)
			assert.Equal(t, tt.wantLine, gotLine, "incorrect line number")
			assert.Equal(t, tt.wantCol, gotCol, "incorrect column number")
		})
	}
}

func TestRawPosition_HasRangeOverlapWith(t *testing.T) {
	tests := []struct {
		name     string
		pos      position.RawPosition
		start    position.RawPosition
		expected bool
		message  string
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
			message:  "positions with exact same range should overlap",
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
			message:  "position contained within another should overlap",
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
			message:  "positions overlapping at start should overlap",
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
			message:  "positions overlapping at end should overlap",
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
			message:  "positions before each other should not overlap",
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
			message:  "positions after each other should not overlap",
		},
		{
			name: "adjacent - touching",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 14,
			},
			start: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			expected: true,
			message:  "adjacent positions should overlap when they touch",
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
			message:  "zero-length position should overlap with position at same offset",
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
			message:  "zero-length positions at same offset should overlap",
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
			message:  "zero-length position should overlap with range containing its offset",
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
			message:  "zero-length position should overlap with range containing its offset",
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
			message:  "single-character position should overlap with range containing its offset",
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
			message:  "single-character position should overlap with range containing its offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.HasRangeOverlapWith(tt.start)
			assert.Equal(t, tt.expected, result, tt.message)
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
		message  string
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
			message: "position at start of file should have offset 0",
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
			message: "position in middle of first line should have correct offset",
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
			message: "position at start of second line should account for newline",
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
			message: "position at end of file should have correct offset",
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
			message: "position on empty line should have correct offset",
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
			message: "position after spaces should have correct offset",
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
			message: "position in template should have correct offset",
		},
		{
			name:     "template example with empty text",
			line:     2,
			col:      12,
			text:     "",
			fileText: "{{- /*gotype: test.Person*/ -}}\nAddress:\n  Street: {{.Address.Street}}",
			want: position.RawPosition{
				Text:   "",
				Offset: 53,
			},
			message: "empty position in template should have correct offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := position.NewRawPositionFromLineAndColumn(tt.line, tt.col, tt.text, tt.fileText)
			assert.Equal(t, tt.want.Offset, got.Offset, tt.message+" (offset)")
			assert.Equal(t, tt.want.Text, got.Text, tt.message+" (text)")
		})
	}
}

func TestRawPosition_GetRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      position.RawPosition
		fileText string
		want     position.Range
		message  string
	}{
		{
			name: "single line range",
			pos: position.RawPosition{
				Text:   "Hello",
				Offset: 0,
			},
			fileText: "Hello World",
			want: position.Range{
				Start: position.Place{Line: 0, Character: 0},
				End:   position.Place{Line: 0, Character: 6},
			},
			message: "range on single line should have correct start and end",
		},
		{
			name: "multi-line range",
			pos: position.RawPosition{
				Text:   "Hello\nWorld",
				Offset: 0,
			},
			fileText: "Hello\nWorld\nTest",
			want: position.Range{
				Start: position.Place{Line: 0, Character: 0},
				End:   position.Place{Line: 1, Character: 6},
			},
			message: "range spanning multiple lines should have correct start and end",
		},
		{
			name: "empty range",
			pos: position.RawPosition{
				Text:   "",
				Offset: 5,
			},
			fileText: "Hello World",
			want: position.Range{
				Start: position.Place{Line: 0, Character: 6},
				End:   position.Place{Line: 0, Character: 6},
			},
			message: "empty range should have same start and end position",
		},
		{
			name: "range at end of line",
			pos: position.RawPosition{
				Text:   "World",
				Offset: 6,
			},
			fileText: "Hello\nWorld\n",
			want: position.Range{
				Start: position.Place{Line: 1, Character: 1},
				End:   position.Place{Line: 1, Character: 6},
			},
			message: "range at end of line should have correct line numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pos.GetRange(tt.fileText)
			assert.Equal(t, tt.want, got, tt.message)
		})
	}
}

func TestRawPosition_ID(t *testing.T) {
	tests := []struct {
		name    string
		pos     position.RawPosition
		want    string
		message string
	}{
		{
			name: "basic position",
			pos: position.RawPosition{
				Text:   "test",
				Offset: 10,
			},
			want:    "test@10",
			message: "ID should combine text and offset with @",
		},
		{
			name: "empty text",
			pos: position.RawPosition{
				Text:   "",
				Offset: 0,
			},
			want:    "@0",
			message: "ID with empty text should just show offset",
		},
		{
			name: "text with special characters",
			pos: position.RawPosition{
				Text:   "hello\nworld",
				Offset: 5,
			},
			want:    "hello\nworld@5",
			message: "ID should preserve special characters in text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pos.ID()
			assert.Equal(t, tt.want, got, tt.message)
		})
	}
}

func TestRawPositionArray_ToStrings(t *testing.T) {
	tests := []struct {
		name    string
		arr     position.RawPositionArray
		want    []string
		message string
	}{
		{
			name: "multiple positions",
			arr: position.RawPositionArray{
				{Text: "hello", Offset: 0},
				{Text: "world", Offset: 6},
			},
			want: []string{
				"hello@0",
				"world@6",
			},
			message: "should convert all positions to strings",
		},
		{
			name:    "empty array",
			arr:     position.RawPositionArray{},
			want:    nil,
			message: "empty array should return nil slice",
		},
		{
			name: "array with empty positions",
			arr: position.RawPositionArray{
				{Text: "", Offset: 0},
				{Text: "", Offset: 5},
			},
			want: []string{
				"@0",
				"@5",
			},
			message: "should handle empty positions correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.arr.ToStrings()
			assert.Equal(t, tt.want, got, tt.message)
		})
	}
}
