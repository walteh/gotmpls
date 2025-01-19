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

	"github.com/walteh/gotmpls/pkg/parser"
	"github.com/walteh/gotmpls/pkg/position"
	"github.com/walteh/gotmpls/pkg/std/text/template/parse"
	"gitlab.com/tozd/go/errors"
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
func GetTokensForText(ctx context.Context, text []byte) ([]Token, error) {
	// Parse with our diagnostic parser
	parsedFile, err := parser.Parse(ctx, "", text)
	if err != nil {
		return nil, errors.Errorf("parsing template: %w", err)
	}

	// Parse with standard parser
	tree := parse.New("")
	tree.Mode = parse.ParseComments | parse.SkipFuncCheck
	treeSet := make(map[string]*parse.Tree)
	_, err = tree.Parse(string(text), "{{", "}}", treeSet)
	if err != nil {
		return nil, errors.Errorf("parsing template: %w", err)
	}

	// Create visitor for the root block
	visitor := newTokenVisitor(&parsedFile.Blocks[0])

	visitor.visitTree(tree)

	// Walk the tree
	for _, node := range tree.Root.Nodes {
		visitor.Visit(node)
	}

	return visitor.tokens, nil
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
