package completion

import (
	"strings"
	"unicode"
)

// CompletionContext represents the context in which completion is being requested
type CompletionContext struct {
	Line       string
	Position   int
	InAction   bool
	AfterDot   bool
	Expression string
}

// NewCompletionContext creates a new completion context for the given line and position
func NewCompletionContext(content string, line, character int) *CompletionContext {
	// Get the current line's content
	lines := strings.Split(content, "\n")
	if line <= 0 || line > len(lines) {
		return &CompletionContext{}
	}
	currentLine := lines[line-1]
	if character <= 0 || character > len(currentLine) {
		return &CompletionContext{}
	}

	// Create the context with 0-based position
	ctx := &CompletionContext{
		Line:     currentLine,
		Position: character,
	}

	// Find the first dot in the line
	firstDot := -1
	for i := 0; i < len(currentLine); i++ {
		if currentLine[i] == '.' {
			firstDot = i
			break
		}
	}

	// If we found a dot, adjust the line to start from there
	if firstDot >= 0 {
		fieldPart := currentLine[firstDot:]
		ctx.Line = fieldPart
		ctx.Position = character - firstDot
	}

	// Set the context flags
	ctx.AfterDot = ctx.isDotCompletion()
	if ctx.AfterDot {
		ctx.Expression = ctx.getExpressionBeforeDot()
	}

	return ctx
}

func (c *CompletionContext) isInTemplateAction() bool {
	// For now, we're not dealing with template actions
	// We'll just focus on field completions
	return false
}

func (c *CompletionContext) isDotCompletion() bool {
	if c.Position <= 0 || c.Position > len(c.Line) {
		return false
	}

	// Check if we're right after a dot
	if c.Position > 0 && c.Line[c.Position-1] == '.' {
		return true
	}

	// Also check if we're at the start of a field reference
	if c.Position == 1 && c.Line[0] == '.' {
		return true
	}

	// Also check if we're at position 2 in a line starting with a dot
	if c.Position == 2 && len(c.Line) >= 2 && c.Line[0] == '.' {
		return true
	}

	return false
}

func (c *CompletionContext) getExpressionBeforeDot() string {
	if c.Position <= 0 || c.Position > len(c.Line) {
		return ""
	}

	// Find the last dot before the position
	dotIndex := -1
	for i := c.Position - 1; i >= 0; i-- {
		if c.Line[i] == '.' {
			dotIndex = i
			break
		}
	}
	if dotIndex == -1 {
		return ""
	}

	// Find the start of the expression before the dot
	start := dotIndex - 1
	for start >= 0 && unicode.IsSpace(rune(c.Line[start])) {
		start--
	}

	// Find the end of the expression
	end := start + 1
	for start >= 0 && (unicode.IsLetter(rune(c.Line[start])) || unicode.IsDigit(rune(c.Line[start])) || c.Line[start] == '_') {
		start--
	}
	start++

	if start >= end {
		return ""
	}

	return c.Line[start:end]
}
