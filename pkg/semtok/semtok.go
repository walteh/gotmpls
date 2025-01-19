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
	"fmt"

	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/std/text/template/parse"
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

	parser := parse.New("semtok")
	parser.Mode = parse.ParseComments | parse.SkipFuncCheck

	// Parse the template
	tree, err := parser.Parse(string(content), "{{", "}}", map[string]*parse.Tree{})
	if err != nil {
		return nil, err
	}

	// Create a visitor and walk the AST
	visitor, err := newVisitor(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create visitor: %w", err)
	}

	visitor.visitTree(tree)

	if err := visitor.visitNode(tree.Root); err != nil {
		return nil, fmt.Errorf("failed to visit nodes: %w", err)
	}

	return visitor.getTokens(), nil
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

// GetTokensForFile is an alias for GetTokensForText that's specifically
// meant for processing entire files.
func GetTokensForFile(ctx context.Context, content []byte) ([]uint32, error) {
	// TODO(@semtok): Implement file-based token generation
	// This should convert our Token structs to the LSP's uint32 array format
	return nil, nil
}
