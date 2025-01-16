package template

import (
	"context"
	"strings"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
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
	return semantics.TokensToLSP(tokens), nil
}

// GetTokensForRange returns semantic tokens for a specific range in a file
func (p *TemplateTokenParser) GetTokensForRange(ctx context.Context, uri string, content string, rng protocol.Range) (*protocol.SemanticTokens, error) {
	// For now, we'll just return tokens for the entire file
	// In the future, we can optimize this to only parse the requested range
	return p.GetTokensForFile(ctx, uri, content)
}

// Parse parses the template content and returns semantic tokens
func (p *TemplateTokenParser) Parse(content string) []semantics.Token {
	var tokens []semantics.Token
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		pos := 0
		for {
			// Find opening delimiter
			openIdx := strings.Index(line[pos:], "{{")
			if openIdx == -1 {
				break
			}
			openIdx += pos

			// Add delimiter token
			tokens = append(tokens, semantics.Token{
				Type:   semantics.TokenTypeDelimiter,
				Line:   uint32(lineNum),
				Start:  uint32(openIdx),
				Length: 2,
			})

			// Find closing delimiter
			closeIdx := strings.Index(line[openIdx:], "}}")
			if closeIdx == -1 {
				break
			}
			closeIdx += openIdx

			// Parse content between delimiters
			if closeIdx > openIdx+2 {
				blockTokens := p.parseTemplateBlock(line[openIdx+2:closeIdx], uint32(lineNum), uint32(openIdx+2))
				tokens = append(tokens, blockTokens...)
			}

			// Add closing delimiter token
			tokens = append(tokens, semantics.Token{
				Type:   semantics.TokenTypeDelimiter,
				Line:   uint32(lineNum),
				Start:  uint32(closeIdx),
				Length: 2,
			})

			pos = closeIdx + 2
		}
	}

	return tokens
}

var templateKeywords = map[string]bool{
	"if":       true,
	"else":     true,
	"range":    true,
	"with":     true,
	"end":      true,
	"define":   true,
	"block":    true,
	"template": true,
}

var builtinFunctions = map[string]bool{
	"len":      true,
	"print":    true,
	"printf":   true,
	"println":  true,
	"html":     true,
	"js":       true,
	"urlquery": true,
}

func (p *TemplateTokenParser) parseTemplateBlock(content string, lineNum uint32, startPos uint32) []semantics.Token {
	var tokens []semantics.Token

	// Split content into words
	words := strings.Fields(content)
	currentPos := startPos

	for i, word := range words {
		if i == 0 && templateKeywords[word] {
			// Keywords are readonly
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeKeyword,
				Line:      lineNum,
				Start:     currentPos,
				Length:    uint32(len(word)),
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
			})
		} else if i == 0 && builtinFunctions[word] {
			// Builtin functions are readonly
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeFunction,
				Line:      lineNum,
				Start:     currentPos,
				Length:    uint32(len(word)),
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly, semantics.ModifierDefaultLibrary},
			})
		} else if strings.HasPrefix(word, "$") {
			// Variable declarations and references
			if i > 0 && words[i-1] == ":=" {
				tokens = append(tokens, semantics.Token{
					Type:      semantics.TokenTypeVariable,
					Line:      lineNum,
					Start:     currentPos,
					Length:    uint32(len(word)),
					Modifiers: []semantics.TokenModifier{semantics.ModifierDeclaration},
				})
			} else {
				tokens = append(tokens, semantics.Token{
					Type:      semantics.TokenTypeVariable,
					Line:      lineNum,
					Start:     currentPos,
					Length:    uint32(len(word)),
					Modifiers: []semantics.TokenModifier{semantics.ModifierDefinition},
				})
			}
		} else if strings.HasPrefix(word, ".") {
			// Field access
			tokens = append(tokens, semantics.Token{
				Type:   semantics.TokenTypeVariable,
				Line:   lineNum,
				Start:  currentPos,
				Length: uint32(len(word)),
			})
		} else if word == ":=" {
			// Assignment operator is readonly
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeOperator,
				Line:      lineNum,
				Start:     currentPos,
				Length:    uint32(len(word)),
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
			})
		} else if strings.HasPrefix(word, `"`) && strings.HasSuffix(word, `"`) {
			// String literals
			tokens = append(tokens, semantics.Token{
				Type:      semantics.TokenTypeString,
				Line:      lineNum,
				Start:     currentPos,
				Length:    uint32(len(word)),
				Modifiers: []semantics.TokenModifier{semantics.ModifierReadonly},
			})
		}

		currentPos += uint32(len(word) + 1) // +1 for space
	}

	return tokens
}
