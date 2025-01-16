package semantics

import (
	"context"
	"sort"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

// TokenType represents semantic token types
type TokenType = uint32

const (
	TokenTypeNamespace TokenType = iota
	TokenTypeType
	TokenTypeClass
	TokenTypeEnum
	TokenTypeInterface
	TokenTypeStruct
	TokenTypeTypeParameter
	TokenTypeParameter
	TokenTypeVariable
	TokenTypeProperty
	TokenTypeEnumMember
	TokenTypeDecorator
	TokenTypeEvent
	TokenTypeFunction
	TokenTypeMethod
	TokenTypeMacro
	TokenTypeKeyword
	TokenTypeModifier
	TokenTypeComment
	TokenTypeString
	TokenTypeNumber
	TokenTypeRegexp
	TokenTypeOperator
	TokenTypeDelimiter
)

// TokenModifier represents semantic token modifiers
type TokenModifier = uint32

const (
	ModifierDeclaration TokenModifier = iota
	ModifierDefinition
	ModifierReadonly
	ModifierStatic
	ModifierDeprecated
	ModifierAbstract
	ModifierAsync
	ModifierModification
	ModifierDocumentation
	ModifierDefaultLibrary
)

// Token represents a semantic token with its position and modifiers
type Token struct {
	Type      TokenType
	Modifiers []TokenModifier
	Line      uint32
	Start     uint32
	Length    uint32
}

// Provider is the interface for semantic token providers
type Provider interface {
	// GetTokensForFile returns semantic tokens for an entire file
	GetTokensForFile(ctx context.Context, uri string, content string) (*protocol.SemanticTokens, error)

	// GetTokensForRange returns semantic tokens for a specific range in a file
	GetTokensForRange(ctx context.Context, uri string, content string, rng protocol.Range) (*protocol.SemanticTokens, error)
}

// TokensToLSP converts internal tokens to LSP semantic tokens
func TokensToLSP(tokens []Token) *protocol.SemanticTokens {
	if len(tokens) == 0 {
		return &protocol.SemanticTokens{
			Data: []uint32{},
		}
	}

	// Sort tokens by line and start position
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Line != tokens[j].Line {
			return tokens[i].Line < tokens[j].Line
		}
		return tokens[i].Start < tokens[j].Start
	})

	// Convert to LSP format (relative line/character positions)
	var data []uint32
	var prevLine, prevStart uint32

	for _, token := range tokens {
		// Calculate delta line and start
		deltaLine := token.Line - prevLine
		deltaStart := uint32(0)
		if deltaLine == 0 {
			deltaStart = token.Start - prevStart
		} else {
			deltaStart = token.Start
		}

		// Calculate modifiers bitset
		var modifiers uint32
		for _, mod := range token.Modifiers {
			modifiers |= 1 << mod
		}

		// Add token data
		data = append(data, []uint32{
			deltaLine,
			deltaStart,
			token.Length,
			token.Type,
			modifiers,
		}...)

		prevLine = token.Line
		prevStart = token.Start
	}

	return &protocol.SemanticTokens{
		Data: data,
	}
}
