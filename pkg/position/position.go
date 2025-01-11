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

type Location struct {
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

// func hackGetLineTextFromParsePos(pos parse.Pos) (int, string) {
// 	// Access the unexported skipCaller field
// 	v := reflect.ValueOf(pos).Elem() // Get the value of the pointer
// 	field := v.FieldByName("line")
// 	fieldText := v.FieldByName("text")

// 	if field.IsValid() && field.CanAddr() {
// 		// Use unsafe to bypass field access restrictions
// 		return int(field.Int()), fieldText.String()
// 	}

// 	return 0, ""
// }

// func NewParserPosition(text string, pos parse.Pos) RawPosition {
// 	line, text := hackGetLineTextFromParsePos(pos)
// 	return RawPosition{Text: text, Offset: line}
// }

func NewRawPositionFromLineAndColumn(line, col int, text, fileText string) RawPosition {
	split := strings.Split(fileText, "\n")
	offset := 0
	for i := 0; i < line-1; i++ {
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
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position()),
	}
}

func (p RawPosition) HasRangeOverlapWith(start RawPosition) bool {
	startOffset := start.Offset
	endOffset := startOffset + start.Length()

	posOffset := p.Offset
	posEndOffset := posOffset + p.Length()

	return posOffset >= startOffset && posOffset <= endOffset || posEndOffset >= startOffset && posEndOffset <= endOffset
}

// GetLineAndColumn calculates the line and column number for a given position in the text
// pos is 0-based, but we return 1-based line and column numbers as per editor/IDE conventions
func GetLineAndColumn(text string, pos parse.Pos) (line, col int) {
	if pos == 0 {
		return 1, 1
	}

	// Count newlines up to pos to get line number
	line = 1 // Start at line 1
	lastNewline := -1
	for i := 0; i < int(pos); i++ {
		if text[i] == '\n' {
			line++
			lastNewline = i
		}
	}

	// Column is just the distance from the last newline + 1 (for 1-based column)
	col = int(pos) - lastNewline

	return line, col
}

// GetLineColumnRange calculates the line/column range for a RawPosition
func (p RawPosition) GetLocation(fileText string) Location {
	startLine, startCol := GetLineAndColumn(fileText, parse.Pos(p.Offset))
	endLine, endCol := GetLineAndColumn(fileText, parse.Pos(p.Offset+p.Length()))
	return Location{
		Start: Place{Line: startLine, Character: startCol},
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
