package parser

import (
	"strings"
	"text/template/parse"

	"gitlab.com/tozd/go/errors"
)

// Converter converts Go template AST nodes to our custom AST nodes
type Converter struct {
	// source is the original template source code
	source string
}

// NewConverter creates a new Converter
func NewConverter(source string) *Converter {
	return &Converter{
		source: source,
	}
}

// extractTypeHints extracts type hints from the source code
func (c *Converter) extractTypeHints(source string) []TypeHint {
	var hints []TypeHint
	if matches := typeHintRegex.FindStringSubmatch(source); len(matches) > 1 {
		typePath := strings.TrimSpace(matches[1])
		// Type hint is always on the first line, column is fixed at 12 (after "/*gotype: ")
		hints = append(hints, TypeHint{
			TypePath: typePath,
			Line:     1,
			Column:   12,
		})
	}
	return hints
}

// positionFromOffset calculates the line and column for a given offset in the source
func positionFromOffset(source string, offset int) (line, col int) {
	if offset < 0 {
		return 1, 1
	}

	line = 1
	lastNL := -1

	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
			lastNL = i
		}
	}

	if lastNL == -1 {
		col = offset + 1
	} else {
		col = offset - lastNL
	}

	return line, col
}

// ConvertTree converts a parse.Node tree into a TemplateInfo structure
func (c *Converter) ConvertTree(node parse.Node, source string) (*TemplateInfo, error) {
	if node == nil {
		return nil, errors.Errorf("cannot convert nil node")
	}

	info := &TemplateInfo{
		Filename: "test.tmpl",
	}

	// Extract type hints first
	if typeHints := c.extractTypeHints(source); len(typeHints) > 0 {
		info.TypeHints = typeHints
	}

	// Convert the node tree
	if err := c.convertNode(node, source, info); err != nil {
		return nil, errors.Errorf("failed to convert node tree: %w", err)
	}

	return info, nil
}

// convertNode recursively converts a parse.Node into TemplateInfo fields
func (c *Converter) convertNode(node parse.Node, source string, info *TemplateInfo) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.ActionNode:
		if n.Pipe != nil {
			for _, cmd := range n.Pipe.Cmds {
				for _, node := range cmd.Args {
					switch argNode := node.(type) {
					case *parse.FieldNode:
						// Variable reference
						pos := int(argNode.Position())
						line, col := positionFromOffset(source, pos)
						endPos := pos + len(argNode.String()) - 1 // Subtract 1 to get the last character position
						endLine, endCol := positionFromOffset(source, endPos)
						if info.Variables == nil {
							info.Variables = make([]VariableLocation, 0, 1)
						}
						info.Variables = append(info.Variables, VariableLocation{
							Name:    argNode.Ident[0],
							Line:    line,
							Column:  col,
							EndLine: endLine,
							EndCol:  endCol,
						})
					case *parse.IdentifierNode:
						// Function call
						pos := int(argNode.Position())
						line, col := positionFromOffset(source, pos)
						endPos := pos + len(argNode.String()) - 1 // Subtract 1 to get the last character position
						endLine, endCol := positionFromOffset(source, endPos)
						if info.Functions == nil {
							info.Functions = make([]FunctionLocation, 0, 1)
						}
						info.Functions = append(info.Functions, FunctionLocation{
							Name:    argNode.Ident,
							Line:    line,
							Column:  col,
							EndLine: endLine,
							EndCol:  endCol,
						})
					}
				}
			}
		}

	case *parse.ListNode:
		// If this is the root node, create a main definition
		if len(n.Nodes) > 0 {
			// Find the end position by scanning for {{end}}
			endLine, endCol := 1, 1
			if idx := strings.LastIndex(source, "{{end}}"); idx >= 0 {
				beforeEnd := source[:idx+7] // include {{end}}
				endLine = strings.Count(beforeEnd, "\n") + 1
				if lastNL := strings.LastIndex(beforeEnd, "\n"); lastNL >= 0 {
					endCol = len(beforeEnd) - lastNL - 1
				} else {
					endCol = len(beforeEnd)
				}
			}

			if info.Definitions == nil {
				info.Definitions = make([]DefinitionInfo, 0, 1)
			}
			info.Definitions = append(info.Definitions, DefinitionInfo{
				Name:     "main",
				Line:     2, // Start after type hint
				Column:   1,
				EndLine:  endLine,
				EndCol:   endCol,
				NodeType: "definition",
			})
		}

		// Process each child node
		for _, child := range n.Nodes {
			if err := c.convertNode(child, source, info); err != nil {
				return errors.Errorf("failed to convert child node: %w", err)
			}
		}

	case *parse.TemplateNode:
		// Named template definition
		startLine, startCol := positionFromOffset(source, int(n.Position()))
		endLine, endCol := startLine, startCol

		// Find the end by scanning for {{end}}
		text := source[n.Position():]
		if idx := strings.Index(text, "{{end}}"); idx >= 0 {
			beforeEnd := text[:idx+7] // include {{end}}
			endLine = startLine + strings.Count(beforeEnd, "\n")
			if lastNL := strings.LastIndex(beforeEnd, "\n"); lastNL >= 0 {
				endCol = len(beforeEnd) - lastNL - 1
			} else {
				endCol = startCol + len(beforeEnd)
			}
		}

		if info.Definitions == nil {
			info.Definitions = make([]DefinitionInfo, 0, 1)
		}
		info.Definitions = append(info.Definitions, DefinitionInfo{
			Name:     n.Name,
			Line:     startLine,
			Column:   startCol,
			EndLine:  endLine,
			EndCol:   endCol,
			NodeType: "definition",
		})
	}

	return nil
}
