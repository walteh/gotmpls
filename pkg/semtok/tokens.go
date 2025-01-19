package semtok

import (
	"github.com/walteh/gotmpls/pkg/position"
)

// TokenType represents the type of a semantic token
type TokenType int

const (
	TokenVariable TokenType = iota + 1
	TokenFunction
	TokenKeyword
	TokenString
	TokenNumber
	TokenComment
	TokenOperator
	TokenFormatSpecifier
	TokenText
)

// TokenModifier represents a modifier for a semantic token
type TokenModifier int

const (
	ModifierNone TokenModifier = iota
	ModifierDeclaration
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

// Token represents a semantic token in the template
type Token struct {
	Type     TokenType
	Modifier TokenModifier
	Range    position.RawPosition
}
