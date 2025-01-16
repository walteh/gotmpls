package template

import (
	"context"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"github.com/walteh/go-tmpl-typer/pkg/position"
	"github.com/walteh/go-tmpl-typer/pkg/semantics"
)

// TemplateTokenParser is responsible for parsing Go templates and extracting semantic tokens
type TemplateTokenParser struct{}

// NewTemplateTokenParser creates a new template token parser
func NewTemplateTokenParser() *TemplateTokenParser {
	return &TemplateTokenParser{}
}

// GetTokensForFile returns semantic tokens for an entire file
func (p *TemplateTokenParser) GetTokensForFile(ctx context.Context, uri string, content string) (*protocol.SemanticTokens, error) {
	tokens := p.Parse(content)
	return semantics.TokensToLSP(tokens, content), nil
}

// GetTokensForRange returns semantic tokens for a specific range in a file
func (p *TemplateTokenParser) GetTokensForRange(ctx context.Context, uri string, content string, rng protocol.Range) (*protocol.SemanticTokens, error) {
	// For now, we'll just return tokens for the entire file
	// In the future, we can optimize this to only parse the requested range
	return p.GetTokensForFile(ctx, uri, content)
}

// Parse parses the template content and returns semantic tokens
func (p *TemplateTokenParser) Parse(content string) []semantics.Token {
	ast, err := Parser.ParseString("", content)
	if err != nil {
		return nil
	}

	var tokens []semantics.Token
	offset := 0

	for _, node := range ast.Nodes {
		if node.Action != nil {
			// Opening delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: offset, Text: node.Action.OpenDelim},
			})
			offset += len(node.Action.OpenDelim) + 1 // +1 for space

			// Process pipeline
			pipelineTokens := p.convertPipelineToTokens(node.Action.Pipeline, offset)
			tokens = append(tokens, pipelineTokens...)

			// Calculate pipeline length by summing up all command and argument lengths
			pipelineLen := 0
			pipeline := node.Action.Pipeline
			for pipeline != nil {
				pipelineLen += len(pipeline.Cmd.Identifier)
				for _, arg := range pipeline.Cmd.Args {
					if arg.Number != "" {
						pipelineLen += len(arg.Number) + 1 // +1 for space
					} else if arg.String != "" {
						pipelineLen += len(arg.String) + 1
					} else if arg.Variable != "" {
						pipelineLen += len(arg.Variable) + 1
					}
				}
				if pipeline.Next != nil {
					pipelineLen += 3 // Space + | + Space
				}
				pipeline = pipeline.Next
			}
			offset += pipelineLen

			// Closing delimiter (add space before)
			offset += 1 // Space before closing delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: offset, Text: node.Action.CloseDelim},
			})
			offset += len(node.Action.CloseDelim)
		} else if node.Comment != nil {
			// Opening delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: offset, Text: node.Comment.OpenDelim},
			})
			offset += len(node.Comment.OpenDelim) + 1 // +1 for space

			// Comment text
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeComment,
				Position: position.RawPosition{Offset: offset, Text: node.Comment.Content},
			})
			offset += len(node.Comment.Content) + 1 // +1 for space

			// Closing delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: offset, Text: node.Comment.CloseDelim},
			})
			offset += len(node.Comment.CloseDelim)
		}
	}

	return tokens
}

func (p *TemplateTokenParser) convertPipelineToTokens(pipeline *Pipeline, offset int) []semantics.Token {
	var tokens []semantics.Token
	for pipeline != nil {
		cmd := pipeline.Cmd
		cmdStart := offset

		// Check for operators first (including eq, ne, etc)
		if isOperator(cmd.Identifier) {
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeOperator,
				Position: position.RawPosition{Offset: cmdStart, Text: cmd.Identifier},
			})
			offset += len(cmd.Identifier) + 1
		} else if isKeyword(cmd.Identifier) {
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeKeyword,
				Position:  position.RawPosition{Offset: cmdStart, Text: cmd.Identifier},
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
			})
			offset += len(cmd.Identifier) + 1
		} else if isBuiltinFunc(cmd.Identifier) {
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeFunction,
				Position:  position.RawPosition{Offset: cmdStart, Text: cmd.Identifier},
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly, semantics.ModifierDefaultLibrary},
			})
			offset += len(cmd.Identifier) + 1
		} else {
			// Variable or function call
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeVariable,
				Position: position.RawPosition{Offset: cmdStart, Text: cmd.Identifier},
			})
			offset += len(cmd.Identifier) + 1
		}

		// Process arguments
		for _, arg := range cmd.Args {
			if arg.Number != "" {
				tokens = append(tokens, semantics.Token{
					Type:      semantics.TokenTypeNumber,
					Position:  position.RawPosition{Offset: offset, Text: arg.Number},
					Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
				})
				offset += len(arg.Number) + 1
			} else if arg.String != "" {
				tokens = append(tokens, semantics.Token{
					Type:     semantics.TokenTypeString,
					Position: position.RawPosition{Offset: offset, Text: arg.String},
				})
				offset += len(arg.String) + 1
			} else if arg.Variable != "" {
				// Check if the variable is actually an operator or built-in function
				if isOperator(arg.Variable) {
					tokens = append(tokens, semantics.Token{
						Type:     semantics.TokenTypeOperator,
						Position: position.RawPosition{Offset: offset, Text: arg.Variable},
					})
				} else if isBuiltinFunc(arg.Variable) {
					tokens = append(tokens, semantics.Token{
						Type:      semantics.TokenTypeFunction,
						Position:  position.RawPosition{Offset: offset, Text: arg.Variable},
						Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly, semantics.ModifierDefaultLibrary},
					})
				} else {
					tokens = append(tokens, semantics.Token{
						Type:     semantics.TokenTypeVariable,
						Position: position.RawPosition{Offset: offset, Text: arg.Variable},
					})
				}
				offset += len(arg.Variable) + 1
			}
		}

		// Process pipe operator if there's a next command
		if pipeline.Next != nil {
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeOperator,
				Position: position.RawPosition{Offset: offset, Text: "|"},
			})
			offset += 3 // Space + | + Space
		}

		pipeline = pipeline.Next
	}
	return tokens
}

var operators = map[string]bool{
	"|":   true,
	":=":  true,
	"eq":  true,
	"ne":  true,
	"lt":  true,
	"le":  true,
	"gt":  true,
	"ge":  true,
	"and": true,
	"or":  true,
	"not": true,
}

func isOperator(s string) bool {
	return operators[s]
}

var keywords = map[string]bool{
	"if":     true,
	"else":   true,
	"range":  true,
	"with":   true,
	"define": true,
	"block":  true,
	"end":    true,
}

func isKeyword(s string) bool {
	return keywords[s]
}

var builtinFuncs = map[string]bool{
	"len":      true,
	"printf":   true,
	"print":    true,
	"println":  true,
	"html":     true,
	"js":       true,
	"urlquery": true,
}

func isBuiltinFunc(s string) bool {
	return builtinFuncs[s]
}
