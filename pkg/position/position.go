// Package position provides utilities for handling text positions and ranges in Go template files.
//
// Understanding Offsets and Zero-Based Indexing:
//
// 1. Single Line Example:
//
//	Text:     "H  e  l  l  o"
//	Offset:    0  1  2  3  4
//	          ↑  ↑  ↑  ↑  ↑
//	          |  |  |  |  └─ RawPosition{Text: "o", Offset: 4}
//	          |  |  |  └─── RawPosition{Text: "lo", Offset: 3}
//	          |  |  └────── RawPosition{Text: "llo", Offset: 2}
//	          |  └───────── RawPosition{Text: "ello", Offset: 1}
//	          └──────────── RawPosition{Text: "Hello", Offset: 0}
//
// 2. Multi-Line Example:
//
//	Text:     "H  e  l  l  o  \n  W  o  r  l  d"
//	Offset:    0  1  2  3  4  5   6  7  8  9  10
//	Line:      0  0  0  0  0  0   1  1  1  1  1
//	Column:    0  1  2  3  4  5   0  1  2  3  4
//	                              ↑
//	                              └─ RawPosition{Text: "World", Offset: 6}
//
// 3. Template Example:
//
//	Text:     "{  {  .  U  s  e  r  .  N  a  m  e  }  }"
//	Offset:    0  1  2  3  4  5  6  7  8  9  10 11 12 13
//	                     ↑──────────┘     ↑──────┘
//	                     |                └─ RawPosition{Text: "Name", Offset: 8}
//	                     └─ RawPosition{Text: "User", Offset: 3}
//
// Key Points:
// - Offset is the byte position from the start of the text (zero-based)
// - Each character position corresponds to its offset
// - Newlines (\n) count as a single offset position
// - Line numbers start at 0 for the first line
// - Column numbers start at 0 for each line
// - RawPosition's Text can be any length, starting at the Offset
//
// Common Operations:
//
// 1. Finding End Position:
//
//	Text:     "H  e  l  l  o"
//	Start:     ↑     Length: 3
//	Offset:    0  1  2  3  4
//	          [------)          <- Range from offset 0, length 3
//	End:              ↑
//	          Start.Offset = 0
//	          End.Offset = Start.Offset + Length = 3
//
// 2. Range Overlap:
//
//	Text:     "a  b  c  d  e  f"
//	Offset:    0  1  2  3  4  5
//	          [------)             Range1: Offset=0, Length=3 ("abc")
//	             [------)          Range2: Offset=1, Length=3 ("bcd")
//	             [---)             Overlap: Offset=1, Length=2 ("bc")
//
// 3. Zero-Length Positions (Cursors):
//
//	Text:     "a  b  c  d  e"
//	Offset:    0  1  2  3  4
//	          [------)          Range: Offset=0, Length=3 ("abc")
//	             ↑              Cursor: Offset=1, Length=0
//	             Cursor overlaps with range if its offset falls within or at edges
package position

import (
	"fmt"
	"strings"

	"github.com/walteh/gotmpls/pkg/std/text/template/parse"

	"github.com/walteh/gotmpls/pkg/lsp/protocol"
)

// Place represents a position in text using line and character (column) numbers.
// Both Line and Character are zero-based.
//
// Example:
//
//	"Hello\nWorld"
//	 ^- Place{Line: 0, Character: 0}
//	     ^- Place{Line: 0, Character: 4}
//	       ^- Place{Line: 1, Character: 0}
type Place struct {
	Line      int
	Character int
}

// Range represents a text range with start and end positions.
// The range is inclusive of Start and exclusive of End.
//
// Example:
//
//	"Hello\nWorld"
//	 [---)  - Range{Start: Place{0,0}, End: Place{0,3}}
type Range struct {
	Start Place
	End   Place
}

// RawPosition represents a position in the source text using byte offset and actual text.
// This is the primary type for position calculations and is more precise than line/column
// positions because it accounts for multi-byte characters and preserves the actual text.
//
// Example:
//
//	"Hello"
//	 ^- RawPosition{Offset: 0, Text: "Hello"}
//	   ^- RawPosition{Offset: 2, Text: "llo"}
type RawPosition struct {
	// Offset is the byte offset in the source text
	Offset int
	// Text is the actual text at this position
	Text string
}

// ID returns a unique identifier for this position based on offset and text.
// The format is "text@offset", which is useful for debugging and equality checks.
//
// Example:
//
//	pos := RawPosition{Offset: 5, Text: "World"}
//	pos.ID() // returns "World@5"
func (p *RawPosition) ID() string {
	return fmt.Sprintf("%s@%d", p.Text, p.Offset)
}

// Length returns the length of the text at this position.
// This is used in range calculations and for determining the end position.
func (p *RawPosition) Length() int {
	return len(p.Text)
}

// NewBasicPosition creates a new RawPosition with the given text and offset.
// This is the simplest way to create a position when you know both values.
//
// Example:
//
//	pos := NewBasicPosition("Hello", 0)  // start of "Hello"
func NewBasicPosition(text string, offset int) RawPosition {
	return RawPosition{Text: text, Offset: offset}
}

// NewRawPositionFromLineAndColumn creates a RawPosition from line/column coordinates.
// This is useful when converting from editor coordinates to byte offsets.
//
// Parameters:
//   - line: zero-based line number
//   - col: zero-based column number
//   - text: the text at the position
//   - fileText: the entire file content
//
// Example:
//
//	text := "Hello\nWorld"
//	pos := NewRawPositionFromLineAndColumn(1, 0, "World", text)
//	// pos.Offset == 6, pos.Text == "World"
func NewRawPositionFromLineAndColumn(line, col int, text, fileText string) RawPosition {
	split := strings.Split(fileText, "\n")
	offset := 0
	for i := 0; i < line; i++ {
		offset += len(split[i]) + 1
	}
	offset += col
	return RawPosition{Text: text, Offset: offset}
}

// ToRawPosition creates a RawPosition from a range of text.
// This is useful when you need to create a position from a range of text.
//
// Example:
//
//	content := "Hello\nWorld"
//	ranged := Range{Start: Place{Line: 0, Character: 0}, End: Place{Line: 0, Character: 5}}
//	pos := ranged.ToRawPosition(content)
//	// pos.Offset == 0, pos.Text == "Hello"
func (ranged Range) ToRawPosition(content string) RawPosition {
	start := NewRawPositionFromLineAndColumn(int(ranged.Start.Line), int(ranged.Start.Character), content, "")
	end := NewRawPositionFromLineAndColumn(int(ranged.End.Line), int(ranged.End.Character), content, "")
	contentz := content[start.Offset:end.Offset]
	start.Text = contentz
	return start
}

// NewIdentifierNodePosition creates a RawPosition from a template parser's IdentifierNode.
// This is used when working with Go's template/parse package to convert AST nodes
// to our position system.
//
// Note: The parser's Position() is 1-based, so we subtract 1 to convert to 0-based.
func NewIdentifierNodePosition(node *parse.IdentifierNode) RawPosition {
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position() - 1),
	}
}

// NewFieldNodePosition creates a RawPosition from a template parser's FieldNode.
// This handles field access expressions like ".Field.SubField" by focusing on
// the last identifier in the chain.
//
// Example:
//
//	{{.User.Name}} -> focuses on "Name" part
func NewFieldNodePosition(node *parse.FieldNode) RawPosition {
	ident := node.Ident[len(node.Ident)-1]
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Pos) - (len(node.String()) - len(ident)),
	}
}

// HasRangeOverlapWith determines if this position overlaps with another position's range.
// The overlap calculation considers both positions as ranges and handles zero-length
// positions specially.
//
// Examples:
//
//	Text: "abcdefg"
//	pos1: RawPosition{Offset: 1, Text: "bcd"}  // range [1,4)
//	pos2: RawPosition{Offset: 2, Text: "cde"}  // range [2,5)
//	pos1.HasRangeOverlapWith(pos2) == true     // overlaps at "cd"
//
//	Zero-length example:
//	pos1: RawPosition{Offset: 2, Text: ""}     // cursor at 'c'
//	pos2: RawPosition{Offset: 1, Text: "bc"}   // range [1,3)
//	pos1.HasRangeOverlapWith(pos2) == true     // cursor within range
func (p RawPosition) HasRangeOverlapWith(start RawPosition) bool {
	// Calculate the bounds for both ranges
	startOffset := start.Offset
	endOffset := startOffset + start.Length()

	posOffset := p.Offset
	posEndOffset := posOffset + p.Length()

	// Handle zero-length ranges
	if p.Length() == 0 {
		// A zero-length position overlaps if it falls within the other range
		return posOffset >= startOffset && posOffset <= endOffset
	}
	if start.Length() == 0 {
		// A zero-length position overlaps if it falls within our range
		return startOffset >= posOffset && startOffset <= posEndOffset
	}

	// Two ranges overlap if one range's start position is before the other range's end position
	// AND its end position is after the other range's start position
	return (startOffset < posEndOffset || startOffset == posEndOffset) && (endOffset > posOffset || endOffset == posOffset)
}

// GetLineAndColumn calculates the zero-based line and column numbers for a position.
// This is useful for converting byte offsets to editor-friendly coordinates.
//
// Example:
//
//	Text: "Hello\nWorld"
//	pos:  RawPosition{Offset: 7, Text: "orld"}
//	line, col := pos.GetLineAndColumn(text)  // returns 1, 1
//
// Visual representation:
//
//	H e l l o \n W o r l d
//	0,0 0,1 ... 0,4  1,0 1,1 ...
func (p RawPosition) GetLineAndColumn(text string) (line, col int) {
	if p.Offset == 0 {
		return 0, 0
	}

	// Count newlines up to pos to get line number
	line = 0 // Start at line 0
	lastNewline := -1
	for i := 0; i < p.Offset; i++ {
		if text[i] == '\n' {
			line++
			lastNewline = i
		}
	}

	// Column is just the distance from the last newline
	col = p.Offset - lastNewline

	return line, col
}

// GetEndPosition returns a RawPosition representing the end of this position's range.
// The returned position is zero-length and located at offset + length.
//
// Example:
//
//	pos := RawPosition{Offset: 0, Text: "Hello"}
//	end := pos.GetEndPosition()  // RawPosition{Offset: 5, Text: ""}
func (p RawPosition) GetEndPosition() RawPosition {
	return RawPosition{
		Text:   "",
		Offset: p.Offset + p.Length(),
	}
}

// deprecated: use ToRange instead
func (p RawPosition) GetRange(fileText string) Range {
	return p.ToRange(fileText)
}

// ToRange converts a RawPosition to a Range using line/column coordinates.
// This is useful when you need to represent the position in editor-friendly coordinates.
//
// Example:
//
//	pos := RawPosition{Offset: 0, Text: "Hello"}
//	rng := pos.ToRange(fileText)
//	// rng.Start == Place{0,0}, rng.End == Place{0,5}
func (p RawPosition) ToRange(fileText string) Range {
	startLine, startCol := p.GetLineAndColumn(fileText)
	endLine, endCol := p.GetEndPosition().GetLineAndColumn(fileText)
	return Range{
		Start: Place{Line: startLine, Character: startCol},
		End:   Place{Line: endLine, Character: endCol},
	}
}

func (p RawPosition) ToLSPPosition(fileText string) protocol.Position {
	rnge := p.GetRange(fileText)
	return protocol.Position{Line: uint32(rnge.Start.Line), Character: uint32(rnge.Start.Character)}
}

func (p RawPosition) ToLSPRange(fileText string) protocol.Range {
	rnge := p.GetRange(fileText)
	return protocol.Range{
		Start: protocol.Position{Line: uint32(rnge.Start.Line), Character: uint32(rnge.Start.Character)},
		End:   protocol.Position{Line: uint32(rnge.End.Line), Character: uint32(rnge.End.Character)},
	}
}

// String returns a string representation of the position.
// This is the same as ID() and is useful for debugging and logging.
func (p RawPosition) String() string {
	return fmt.Sprintf("%s@%d", p.Text, p.Offset)
}

type RawPositionArray []RawPosition

// ToStrings converts a RawPositionArray to a slice of strings using String().
// This is useful for debugging and logging multiple positions.
func (me RawPositionArray) ToStrings() []string {
	var texts []string
	for _, pos := range me {
		texts = append(texts, pos.String())
	}
	return texts
}

// NewStringNodePosition creates a new position from a string node
func NewStringNodePosition(node *parse.StringNode) RawPosition {
	return RawPosition{
		Text:   node.Text,
		Offset: int(node.Pos),
	}
}

func NewRangeFromLSPRange(ranged protocol.Range) Range {
	return Range{
		Start: Place{Line: int(ranged.Start.Line), Character: int(ranged.Start.Character)},
		End:   Place{Line: int(ranged.End.Line), Character: int(ranged.End.Character)},
	}
}
