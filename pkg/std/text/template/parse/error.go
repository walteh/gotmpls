package parse

import "fmt"

type ParseError struct {
	Message       string
	LegacyMessage string
	Position      Pos
	Line          int
	Type          itemType
	Value         string
	Text          string
}

func (me *ParseError) Error() string {
	return me.LegacyMessage
}

func (t *Tree) NewParseErrorf(format string, args ...any) *ParseError {
	erd := ParseError{
		LegacyMessage: fmt.Sprintf("template: %s:%d: %s", t.ParseName, t.token[0].line, fmt.Sprintf(format, args...)),
		Position:      t.token[0].pos,
		Line:          t.token[0].line,
		Type:          t.token[0].typ,
		Value:         t.token[0].val,
		Text:          t.text,
	}
	return &erd
}

func (t *Tree) NewParseError(err error) *ParseError {
	erd := ParseError{
		LegacyMessage: err.Error(),
		Position:      t.token[0].pos,
		Line:          t.token[0].line,
		Type:          t.token[0].typ,
		Value:         t.token[0].val,
		Text:          t.text,
	}
	return &erd
}

func (t *Tree) AddParseError(err *ParseError) {
	t.errors = append(t.errors, err)
}

func (t *Tree) errorfNoPanic(format string, args ...any) {
	t.AddParseError(t.NewParseErrorf(format, args...))
}

// Errors returns all errors encountered during parsing.
func (t *Tree) Errors() []*ParseError {
	return t.errors
}
