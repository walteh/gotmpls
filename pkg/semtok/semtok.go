/*
Package semtok provides semantic token support for Go templates.

Core Functions:
-------------

	       Input
	         |
	         v
	  +------------+
	  | Template   |
	  | Text      |
	  +------------+
	         |
	  Parse & Visit
	         |
	         v
	  +------------+
	  | AST Nodes  |
	  +------------+
	         |
	Convert to Tokens
	         |
	         v
	  +------------+
	  | Semantic   |
	  | Tokens     |
	  +------------+
*/
package semtok

import (
	"context"

	"github.com/walteh/gotmpls/pkg/position"
)

// GetTokensForText returns semantic tokens for the given template text.
// This is the main entry point for semantic token generation.
//
//	Example:
//	   tokens, err := GetTokensForText(ctx, []byte("{{ .Name }}"))
//	   if err != nil {
//	       return err
//	   }
//	   // Use tokens...
func GetTokensForText(ctx context.Context, content []byte) ([]Token, error) {
	// TODO(@semtok): Implement token generation
	// 1. Parse template
	// 2. Visit AST nodes
	// 3. Convert to semantic tokens
	return nil, nil
}

// GetTokensForRange returns semantic tokens for a specific range in the template.
// This is used for incremental updates in the LSP server.
//
//	Example:
//	   tokens, err := GetTokensForRange(ctx, content, &position.RawPosition{...})
//	   if err != nil {
//	       return err
//	   }
//	   // Use tokens...
func GetTokensForRange(ctx context.Context, content []byte, ranged *position.RawPosition) ([]Token, error) {
	// TODO(@semtok): Implement range-based token generation
	// 1. Parse template
	// 2. Filter nodes in range
	// 3. Convert to semantic tokens
	return nil, nil
}
