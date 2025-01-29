package semtok

import (
	"github.com/walteh/gotmpls/pkg/position"
)

// TokenType represents the type of a semantic token
type TokenType string

const (
	TokenVariable  TokenType = "variable"
	TokenFunction  TokenType = "function"
	TokenKeyword   TokenType = "keyword"
	TokenString    TokenType = "string"
	TokenNumber    TokenType = "number"
	TokenComment   TokenType = "comment"
	TokenOperator  TokenType = "operator"
	TokenMacro     TokenType = "macro"
	TokenNamespace TokenType = "namespace"
	TokenParameter TokenType = "parameter"
	TokenTypeKind  TokenType = "type"
	TokenTypeParam TokenType = "typeParameter"
	TokenMethod    TokenType = "method"
	TokenLabel     TokenType = "label"
)

// TokenModifier represents a modifier for a semantic token
type TokenModifier string

const (
	ModifierNone           TokenModifier = ""
	ModifierDeclaration    TokenModifier = "declaration"
	ModifierDefinition     TokenModifier = "definition"
	ModifierReadonly       TokenModifier = "readonly"
	ModifierStatic         TokenModifier = "static"
	ModifierDeprecated     TokenModifier = "deprecated"
	ModifierAbstract       TokenModifier = "abstract"
	ModifierAsync          TokenModifier = "async"
	ModifierModification   TokenModifier = "modification"
	ModifierDocumentation  TokenModifier = "documentation"
	ModifierDefaultLibrary TokenModifier = "defaultLibrary"
)

// Token represents a semantic token in the template
type Token struct {
	Type     TokenType
	Modifier TokenModifier
	Range    position.RawPosition
}
