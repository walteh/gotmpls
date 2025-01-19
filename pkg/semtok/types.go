/*
Token Types and Modifiers:
------------------------
This file defines the core types used for semantic token generation.

Token Types are represented as follows:

	+-------------+     +-----------+
	| TokenType   | --> | Position  |
	+-------------+     +-----------+
	      |                  |
	      v                  v
	[Variable,        [Start, End]
	 Function,         Line:Char
	 Keyword,          Points
	 etc.]

Each token carries both its type and position information.
*/
package semtok

import (
	"github.com/walteh/gotmpls/pkg/position"
)

// TokenType represents the semantic meaning of a token
type TokenType uint32

const (
	// TokenVariable represents a template variable (e.g., .Name)
	TokenVariable TokenType = iota + 1

	// TokenFunction represents a template function (e.g., printf)
	TokenFunction

	// TokenKeyword represents a template keyword (e.g., if, range)
	TokenKeyword

	// TokenOperator represents a template operator (e.g., |)
	TokenOperator

	// TokenString represents a string literal
	TokenString

	// TokenComment represents a template comment
	TokenComment

	// TokenNumber represents a numeric literal (e.g., 0, 1.5)
	TokenNumber
)

// TokenModifier represents additional characteristics of a token
type TokenModifier uint32

const (
	// ModifierNone indicates no special characteristics
	ModifierNone TokenModifier = 0

	// ModifierDeclaration indicates first occurrence/declaration
	ModifierDeclaration TokenModifier = 1 << iota

	// ModifierReadonly indicates the token is constant/readonly
	ModifierReadonly

	// ModifierStatic indicates the token is static/global
	ModifierStatic
)

// Token represents a semantic token with its type, modifiers, and position
type Token struct {
	// Type indicates the semantic meaning of the token
	Type TokenType

	// Modifier indicates any special characteristics
	Modifier TokenModifier

	// Range indicates the token's position in the source
	Range position.RawPosition
}

// String returns a human-readable representation of the token type
func (t TokenType) String() string {
	switch t {
	case TokenVariable:
		return "variable"
	case TokenFunction:
		return "function"
	case TokenKeyword:
		return "keyword"
	case TokenOperator:
		return "operator"
	case TokenString:
		return "string"
	case TokenComment:
		return "comment"
	case TokenNumber:
		return "number"
	default:
		return "unknown"
	}
}

// String returns a human-readable representation of the token modifier
func (m TokenModifier) String() string {
	switch m {
	case ModifierNone:
		return "none"
	case ModifierDeclaration:
		return "declaration"
	case ModifierReadonly:
		return "readonly"
	case ModifierStatic:
		return "static"
	default:
		return "unknown"
	}
}
