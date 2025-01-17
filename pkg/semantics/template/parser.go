package template

import (
	"context"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/semantics"
)

var (
	// Lexer rules for the template parser
	templateLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "OpenDelim", Pattern: `{{-?`},
		{Name: "CloseDelim", Pattern: `-?}}`},
		{Name: "CommentText", Pattern: `/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`},
		{Name: "Number", Pattern: `\d+`},
		{Name: "String", Pattern: `"[^"]*"`},
		{Name: "DotIdent", Pattern: `\.[a-zA-Z][a-zA-Z0-9]*`},
		{Name: "Ident", Pattern: `[a-zA-Z][a-zA-Z0-9]*`},
		{Name: "Text", Pattern: `[^{]+|{[^{]`}, // Match any non-{ char or a single { not followed by another {
		{Name: "whitespace", Pattern: `[ \t]+`},
	})

	// Parser instance for the template grammar
	templateParser = participle.MustBuild[Template](
		participle.Lexer(templateLexer),
		participle.Elide("whitespace"),
	)
)

// TemplateTokenParser implements the semantics.Provider interface
type TemplateTokenParser struct{}

var _ semantics.Provider = &TemplateTokenParser{}

// GetTokensForFile returns semantic tokens for a file
func (p *TemplateTokenParser) GetTokensForFile(ctx context.Context, uri string, content string) (*protocol.SemanticTokens, error) {
	tokens, err := p.Parse(content)
	if err != nil {
		return nil, err
	}
	return semantics.TokensToLSP(tokens, uri), nil
}

// GetTokensForRange returns semantic tokens for a range within a file
func (p *TemplateTokenParser) GetTokensForRange(ctx context.Context, uri string, content string, ranged protocol.Range) (*protocol.SemanticTokens, error) {
	tokens, err := p.Parse(content)
	if err != nil {
		return nil, err
	}
	rnge := position.NewRangeFromLSPRange(ranged)
	start := rnge.ToRawPosition(content)
	end := start.GetEndPosition()
	var rangeTokens []semantics.Token
	for _, token := range tokens {
		pos := token.Position.Offset
		if pos >= start.Offset && pos+len(token.Position.Text) <= end.Offset {
			rangeTokens = append(rangeTokens, token)
		}
	}
	return semantics.TokensToLSP(rangeTokens, uri), nil
}

// Parse parses a template string and returns semantic tokens
func (p *TemplateTokenParser) Parse(content string) ([]semantics.Token, error) {
	template, err := templateParser.ParseString("", content)
	if err != nil {
		// If parsing fails, fall back to basic tokenization
		return p.fallbackParse(content)
	}

	var tokens []semantics.Token

	// Process each node
	for _, node := range template.Nodes {
		if node.Text != nil {
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeText,
				Position: position.RawPosition{Offset: int(node.Pos.Offset), Text: *node.Text},
			})
		}

		if node.Action != nil {
			// Add opening delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: int(node.Action.Pos.Offset), Text: node.Action.OpenDelim},
			})

			// Add pipeline content
			if node.Action.Pipeline != nil {
				pipelineTokens := p.processPipeline(node.Action.Pipeline)
				tokens = append(tokens, pipelineTokens...)
			}

			// Add closing delimiter
			tokens = append(tokens, semantics.Token{
				Type: semantics.TokenTypeDelimiter,
				Position: position.RawPosition{
					Offset: int(node.Action.Pos.Offset) + len(node.Action.OpenDelim) + len(node.Action.Pipeline.ToString()),
					Text:   node.Action.CloseDelim,
				},
			})
		}

		if node.Comment != nil {
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeComment,
				Position: position.RawPosition{Offset: int(node.Comment.Pos.Offset), Text: node.Comment.Content},
			})
		}

		if node.Control != nil {
			// Add opening delimiter
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeDelimiter,
				Position: position.RawPosition{Offset: int(node.Control.Pos.Offset), Text: node.Control.OpenDelim},
			})

			// Add keyword
			tokens = append(tokens, semantics.Token{
				Type: semantics.TokenTypeKeyword,
				Position: position.RawPosition{
					Offset: int(node.Control.Pos.Offset) + len(node.Control.OpenDelim),
					Text:   node.Control.Keyword,
				},
			})

			// Add pipeline content if present
			if node.Control.Pipeline != nil {
				pipelineTokens := p.processPipeline(node.Control.Pipeline)
				tokens = append(tokens, pipelineTokens...)
			}

			// Add closing delimiter
			tokens = append(tokens, semantics.Token{
				Type: semantics.TokenTypeDelimiter,
				Position: position.RawPosition{
					Offset: int(node.Control.Pos.Offset) + len(node.Control.OpenDelim) + len(node.Control.Keyword),
					Text:   node.Control.CloseDelim,
				},
			})
		}
	}

	return tokens, nil
}

// processPipeline converts a pipeline node into semantic tokens
func (p *TemplateTokenParser) processPipeline(pipeline *Pipeline) []semantics.Token {
	var tokens []semantics.Token

	current := pipeline
	for current != nil {
		// Add the command identifier
		tokens = append(tokens, semantics.Token{
			Type:     semantics.TokenTypeFunction,
			Position: position.RawPosition{Offset: int(current.Cmd.Pos.Offset), Text: current.Cmd.Identifier},
		})

		// Add arguments
		for _, arg := range current.Cmd.Args {
			var token semantics.Token
			switch {
			case arg.Number != nil:
				token = semantics.Token{
					Type:     semantics.TokenTypeNumber,
					Position: position.RawPosition{Offset: int(arg.Pos.Offset), Text: *arg.Number},
				}
			case arg.String != nil:
				token = semantics.Token{
					Type:     semantics.TokenTypeString,
					Position: position.RawPosition{Offset: int(arg.Pos.Offset), Text: *arg.String},
				}
			case arg.Variable != nil:
				token = semantics.Token{
					Type:     semantics.TokenTypeVariable,
					Position: position.RawPosition{Offset: int(arg.Pos.Offset), Text: *arg.Variable},
				}
			}
			tokens = append(tokens, token)
		}

		current = current.Next
	}

	return tokens
}

// fallbackParse provides basic tokenization for incomplete templates
func (p *TemplateTokenParser) fallbackParse(content string) ([]semantics.Token, error) {
	var tokens []semantics.Token
	currentPos := 0

	for currentPos < len(content) {
		// Look for the next {{
		openIdx := strings.Index(content[currentPos:], "{{")

		if openIdx == -1 {
			// No more delimiters, add the rest as text
			if currentPos < len(content) {
				text := content[currentPos:]
				if len(text) > 0 {
					tokens = append(tokens, semantics.Token{
						Type:     semantics.TokenTypeText,
						Position: position.RawPosition{Offset: currentPos, Text: text},
					})
				}
			}
			break
		}

		// Add text before the delimiter if any
		if openIdx > 0 {
			text := content[currentPos : currentPos+openIdx]
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeText,
				Position: position.RawPosition{Offset: currentPos, Text: text},
			})
		}

		// Look for the closing delimiter
		startDelim := currentPos + openIdx
		afterOpen := startDelim + 2
		closeIdx := strings.Index(content[afterOpen:], "}}")

		// Add the opening delimiter
		tokens = append(tokens, semantics.Token{
			Type:     semantics.TokenTypeDelimiter,
			Position: position.RawPosition{Offset: startDelim, Text: "{{"},
		})

		if closeIdx == -1 {
			// No closing delimiter, treat the rest as an incomplete action
			currentPos = len(content)
			continue
		}

		// Add the content between delimiters as a variable
		actionEnd := afterOpen + closeIdx + 2
		actionContent := content[afterOpen : afterOpen+closeIdx]
		actionContent = strings.TrimSpace(actionContent)

		if len(actionContent) > 0 {
			tokens = append(tokens, semantics.Token{
				Type:     semantics.TokenTypeVariable,
				Position: position.RawPosition{Offset: afterOpen, Text: actionContent},
			})
		}

		// Add the closing delimiter
		tokens = append(tokens, semantics.Token{
			Type:     semantics.TokenTypeDelimiter,
			Position: position.RawPosition{Offset: afterOpen + closeIdx, Text: "}}"},
		})

		currentPos = actionEnd
	}

	return tokens, nil
}
