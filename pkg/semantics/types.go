package semantics

import (
	"context"

	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/position"
)

// TokenType represents the type of a semantic token
type TokenType uint32

// TokenModifier represents a modifier for a semantic token
type TokenModifier uint32

const (
	TokenTypeDelimiter TokenType = iota
	TokenTypeKeyword
	TokenTypeFunction
	TokenTypeVariable
	TokenTypeOperator
	TokenTypeString
	TokenTypeComment
	TokenTypeNumber
	TokenTypeText
	TokenTypeField
)

const (
	ModifierDeclaration TokenModifier = iota
	ModifierDefinition
	ModifierReadonly
	ModifierDefaultLibrary
)

// Token represents a semantic token in the source code
type Token struct {
	Type      TokenType
	Position  position.RawPosition
	Modifiers []TokenModifier
}

// Provider is an interface for getting semantic tokens from a document
type Provider interface {
	// GetTokensForFile returns semantic tokens for an entire file
	GetTokensForFile(ctx context.Context, uri string, content string) (*protocol.SemanticTokens, error)

	// GetTokensForRange returns semantic tokens for a specific range in a file
	GetTokensForRange(ctx context.Context, uri string, content string, rng protocol.Range) (*protocol.SemanticTokens, error)
}

// TokensToLSP converts our internal token format to LSP semantic tokens
func TokensToLSP(tokens []Token, fileText string) *protocol.SemanticTokens {
	if len(tokens) == 0 {
		return &protocol.SemanticTokens{Data: []uint32{}}
	}

	// Sort tokens by line and start position
	sortTokens(tokens)

	// Convert to LSP format
	var result []uint32
	prevLine := uint32(0)
	prevStart := uint32(0)

	for _, token := range tokens {
		line, col := token.Position.GetLineAndColumn(fileText)
		deltaLine := uint32(line) - prevLine
		deltaStart := uint32(0)
		if deltaLine == 0 {
			deltaStart = uint32(col) - prevStart
		} else {
			deltaStart = uint32(col)
		}

		// Encode token data: deltaLine, deltaStart, length, tokenType, tokenModifiers
		result = append(result,
			deltaLine,
			deltaStart,
			uint32(token.Position.Length()),
			uint32(token.Type),
			encodeModifiers(token.Modifiers),
		)

		prevLine = uint32(line)
		prevStart = uint32(col)
	}

	return &protocol.SemanticTokens{Data: result}
}

// encodeModifiers combines token modifiers into a single uint32
func encodeModifiers(modifiers []TokenModifier) uint32 {
	var result uint32
	for _, mod := range modifiers {
		result |= 1 << uint32(mod)
	}
	return result
}

// sortTokens sorts tokens by line and start position
func sortTokens(tokens []Token) {
	// Implementation will be added later if needed
}
