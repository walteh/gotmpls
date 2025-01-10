package position

import (
	"fmt"
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
type RawPosition interface {
	// Offset is the byte offset in the source text
	Offset() int
	// Text is the actual text at this position
	Text() string
}

// ID returns a unique identifier for this position based on offset and text
func PositionID(p RawPosition) string {
	return fmt.Sprintf("%s@%d", p.Text(), p.Offset())
}

// GetLength returns the length of the text at this position
func PositionLength(p RawPosition) int {
	return len(p.Text())
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
func GetLocation(pos RawPosition, fileText string) Location {
	startLine, startCol := GetLineAndColumn(fileText, parse.Pos(pos.Offset()))
	endLine, endCol := GetLineAndColumn(fileText, parse.Pos(pos.Offset()+PositionLength(pos)))
	return Location{
		Start: Place{Line: startLine, Character: startCol},
		End:   Place{Line: endLine, Character: endCol},
	}
}

type IdentifierNodePosition struct {
	identifierNode *parse.IdentifierNode
}

var _ RawPosition = &IdentifierNodePosition{}

func (me *IdentifierNodePosition) Offset() int {
	return int(me.identifierNode.Position())
}

func (me *IdentifierNodePosition) Text() string {
	return me.identifierNode.String()
}

// func NewIdentifierNodePosition(node *parse.IdentifierNode) *IdentifierNodePosition {
// 	return &IdentifierNodePosition{
// 		identifierNode: node,
// 	}
// }

func NewIdentifierNodePosition(node *parse.IdentifierNode) *BasicPosition {
	return NewBasicPosition(node.String(), int(node.Position()))
}

type BasicPosition struct {
	text   string
	offset int
}

func NewBasicPosition(text string, offset int) *BasicPosition {
	return &BasicPosition{
		text:   text,
		offset: offset,
	}
}

var _ RawPosition = &BasicPosition{}

func (me *BasicPosition) Offset() int {
	return me.offset
}

func (me *BasicPosition) Text() string {
	return me.text
}

type FieldNodePosition struct {
	fieldNode *parse.FieldNode
}

var _ RawPosition = &FieldNodePosition{}

func (me *FieldNodePosition) Offset() int {
	return int(me.fieldNode.Position())
}

func (me *FieldNodePosition) Text() string {
	return me.fieldNode.String()
}

// func NewFieldNodePosition(node *parse.FieldNode) *FieldNodePosition {
// 	return &FieldNodePosition{
// 		fieldNode: node,
// 	}
// }

func NewFieldNodePosition(node *parse.FieldNode) *BasicPosition {
	return NewBasicPosition(node.String(), int(node.Position()))
}

func RawPositionToString(pos RawPosition) string {
	return fmt.Sprintf("%s@%d", pos.Text(), pos.Offset())
}

type RawPositionArray []RawPosition

func (me RawPositionArray) ToStrings() []string {
	var texts []string
	for _, pos := range me {
		texts = append(texts, RawPositionToString(pos))
	}
	return texts
}

func ConvertToBasicPosition(pos RawPosition) *BasicPosition {
	return NewBasicPosition(pos.Text(), pos.Offset())
}
