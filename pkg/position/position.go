package position

import (
	"fmt"
	"strings"
	"text/template/parse"
)

type Place struct {
	Line      int
	Character int
}

type Range struct {
	Start Place
	End   Place
}

// RawPosition represents a position in the source text
type RawPosition struct {
	// Offset is the byte offset in the source text
	Offset int
	// Text is the actual text at this position
	Text string
}

// ID returns a unique identifier for this position based on offset and text
func (p *RawPosition) ID() string {
	return fmt.Sprintf("%s@%d", p.Text, p.Offset)
}

// GetLength returns the length of the text at this position
func (p *RawPosition) Length() int {
	return len(p.Text)
}

func NewBasicPosition(text string, offset int) RawPosition {
	return RawPosition{Text: text, Offset: offset}
}

func NewRawPositionFromLineAndColumn(line, col int, text, fileText string) RawPosition {
	split := strings.Split(fileText, "\n")
	offset := 0
	for i := 0; i < line; i++ {
		offset += len(split[i]) + 1
	}
	offset += col
	return RawPosition{Text: text, Offset: offset}
}

func NewIdentifierNodePosition(node *parse.IdentifierNode) RawPosition {
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position()),
	}
}

func NewFieldNodePosition(node *parse.FieldNode) RawPosition {
	ident := node.Ident[len(node.Ident)-1]
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Pos) - (len(node.String()) - len(ident)),
	}
}

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
	return startOffset < posEndOffset && endOffset > posOffset
}

// GetLineAndColumn calculates the line and column number for a given position in the text
// Returns zero-based line and column numbers
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
	col = p.Offset - lastNewline - 1

	return line, col
}

func (p RawPosition) GetEndPosition() RawPosition {
	return RawPosition{
		Text:   "",
		Offset: p.Offset + p.Length(),
	}
}

// GetLineColumnRange calculates the line/column range for a RawPosition
func (p RawPosition) GetRange(fileText string) Range {
	startLine, startCol := p.GetLineAndColumn(fileText)
	endLine, endCol := p.GetEndPosition().GetLineAndColumn(fileText)
	return Range{
		Start: Place{Line: startLine, Character: startCol + 1},
		End:   Place{Line: endLine, Character: endCol},
	}
}

func (p RawPosition) String() string {
	return fmt.Sprintf("%s@%d", p.Text, p.Offset)
}

type RawPositionArray []RawPosition

func (me RawPositionArray) ToStrings() []string {
	var texts []string
	for _, pos := range me {
		texts = append(texts, pos.String())
	}
	return texts
}
