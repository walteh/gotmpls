package completion

import (
	"strings"
)

// CompletionContext holds information about the completion request context
type CompletionContext struct {
	Content   string
	Line      int
	Character int
	AfterDot  bool
}

// NewCompletionContext creates a new completion context
func NewCompletionContext(content string, line, character int) *CompletionContext {
	ctx := &CompletionContext{
		Content:   content,
		Line:      line,
		Character: character,
	}

	// Check if we're after a dot
	lines := strings.Split(content, "\n")
	if line > 0 && line <= len(lines) {
		currentLine := lines[line-1]
		// Convert 1-based character position to 0-based for array indexing
		char := character - 1
		if char >= 0 && char < len(currentLine) {
			// Check if we're at or right after a dot
			if char > 0 && (currentLine[char] == '.' || (currentLine[char-1] == '.' && !strings.ContainsAny(currentLine[char:char+1], " \t"))) {
				ctx.AfterDot = true
			}
		}
	}

	return ctx
}

// IsInTemplateAction checks if the current position is within a template action ({{ }})
func (c *CompletionContext) IsInTemplateAction() bool {
	lines := strings.Split(c.Content, "\n")
	if c.Line > 0 && c.Line <= len(lines) {
		currentLine := lines[c.Line-1]
		// Convert 1-based character position to 0-based for array indexing
		char := c.Character - 1
		if char >= 0 && char < len(currentLine) {
			// Find the last {{ before the current position
			lastOpen := strings.LastIndex(currentLine[:char+1], "{{")
			if lastOpen == -1 {
				return false
			}
			// Find the next }} after the last {{
			nextClose := strings.Index(currentLine[lastOpen:], "}}")
			if nextClose == -1 {
				return true // No closing bracket found, assume we're in a template action
			}
			// Check if we're between {{ and }}
			return char < lastOpen+nextClose
		}
	}
	return false
}

// GetExpressionBeforeDot returns the expression before the current dot
func (c *CompletionContext) GetExpressionBeforeDot() string {
	lines := strings.Split(c.Content, "\n")
	if c.Line > 0 && c.Line <= len(lines) {
		currentLine := lines[c.Line-1]
		// Convert 1-based character position to 0-based for array indexing
		char := c.Character - 1
		if char >= 0 && char < len(currentLine) {
			// Find the last dot before the current position
			lastDot := strings.LastIndex(currentLine[:char+1], ".")
			if lastDot == -1 {
				return ""
			}
			// Find the previous dot or start of the line
			prevDot := strings.LastIndex(currentLine[:lastDot], ".")
			if prevDot == -1 {
				// Look for {{ or start of line
				start := strings.LastIndex(currentLine[:lastDot], "{{")
				if start == -1 {
					start = 0
				} else {
					start += 2 // Skip past {{
				}
				prevDot = start
			} else {
				prevDot++
			}
			// Return the expression between the dots, trimming any whitespace
			return strings.TrimSpace(currentLine[prevDot:lastDot])
		}
	}
	return ""
}

// IsDotCompletion checks if we should provide dot completion
func (c *CompletionContext) IsDotCompletion() bool {
	return c.AfterDot && c.IsInTemplateAction()
}
