package position

// // IsAfterDot checks if the current position is right after a dot
// func IsAfterDot(pos RawPosition) bool {
// 	return strings.HasPrefix(pos.Text(), ".")
// }

// // IsInTemplateAction checks if the current position is within a template action ({{ }})
// func IsInTemplateAction(pos RawPosition) bool {
// 	text := pos.Text()
// 	offset := pos.Offset()

// 	if len(text) == 0 || offset < 0 {
// 		return false
// 	}

// 	// Find the last {{ before the current position
// 	lastOpen := strings.LastIndex(text[:offset+1], "{{")
// 	if lastOpen == -1 {
// 		return false
// 	}

// 	// Find the next }} after the last {{
// 	nextClose := strings.Index(text[lastOpen:], "}}")
// 	if nextClose == -1 {
// 		// No closing bracket found, assume we're in a template action
// 		return true
// 	}

// 	// Check if we're between {{ and }}
// 	return offset < lastOpen+nextClose
// }

// // GetExpressionBeforeDot returns the expression before the current dot
// func GetExpressionBeforeDot(pos RawPosition) string {
// 	text := pos.Text()
// 	offset := pos.Offset()

// 	if len(text) == 0 || offset < 0 {
// 		return ""
// 	}

// 	// Find the last dot before the current position
// 	lastDot := strings.LastIndex(text[:offset+1], ".")
// 	if lastDot == -1 {
// 		return ""
// 	}

// 	// Find the previous dot or start of the template action
// 	start := strings.LastIndex(text[:lastDot], "{{")
// 	if start != -1 {
// 		start += 2 // Skip past {{
// 	} else {
// 		start = 0
// 	}

// 	// Get the text between start and lastDot
// 	expr := text[start:lastDot]
// 	// Skip any leading dots and spaces
// 	expr = strings.TrimLeft(expr, ". \t")
// 	return expr
// }

// // IsDotCompletion checks if we should provide dot completion
// func IsDotCompletion(pos RawPosition) bool {
// 	return IsAfterDot(pos)
// }
