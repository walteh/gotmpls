package position

import (
	"fmt"
	"text/template/parse"
)

type Document struct {
	text string
}

func NewDocument(text string) *Document {
	return &Document{
		text: text,
	}
}

type Location struct {
	StartColumn int
	EndColumn   int
	StartLine   int
	EndLine     int
}

// RawPosition represents a position in the source text
type RawPosition interface {
	// Offset is the byte offset in the source text
	Offset() int
	// Text is the actual text at this position
	Text() string
	// Document is the document that this position is in
	Document() *Document
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
func GetLocation(pos RawPosition) Location {
	startLine, startCol := GetLineAndColumn(pos.Document().text, parse.Pos(pos.Offset()))
	endLine, endCol := GetLineAndColumn(pos.Document().text, parse.Pos(pos.Offset()+PositionLength(pos)))
	return Location{
		StartColumn: startCol,
		EndColumn:   endCol,
		StartLine:   startLine,
		EndLine:     endLine,
	}
}

type IdentifierNodePosition struct {
	identifierNode *parse.IdentifierNode
	documentRef    *Document
}

var _ RawPosition = &IdentifierNodePosition{}

func (me *IdentifierNodePosition) Offset() int {
	return int(me.identifierNode.Position())
}

func (me *IdentifierNodePosition) Text() string {
	return me.identifierNode.String()
}

func (me *IdentifierNodePosition) Document() *Document {
	return me.documentRef
}

func (me *Document) NewIdentifierNodePosition(node *parse.IdentifierNode) *IdentifierNodePosition {
	return &IdentifierNodePosition{
		identifierNode: node,
		documentRef:    me,
	}
}

type BasicPosition struct {
	text        string
	offset      int
	documentRef *Document
}

func (me *Document) NewBasicPosition(text string, offset int) *BasicPosition {
	return &BasicPosition{
		text:        text,
		offset:      offset,
		documentRef: me,
	}
}

var _ RawPosition = &BasicPosition{}

func (me *BasicPosition) Offset() int {
	return me.offset
}

func (me *BasicPosition) Text() string {
	return me.text
}

func (me *BasicPosition) Document() *Document {
	return me.documentRef
}

type FieldNodePosition struct {
	fieldNode   *parse.FieldNode
	documentRef *Document
}

var _ RawPosition = &FieldNodePosition{}

func (me *FieldNodePosition) Offset() int {
	return int(me.fieldNode.Position())
}

func (me *FieldNodePosition) Text() string {
	return me.fieldNode.String()
}

func (me *FieldNodePosition) Document() *Document {
	return me.documentRef
}

func (me *Document) NewFieldNodePosition(node *parse.FieldNode) *FieldNodePosition {
	return &FieldNodePosition{
		fieldNode:   node,
		documentRef: me,
	}
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
