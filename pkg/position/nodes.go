package position

import "github.com/walteh/gotmpls/pkg/std/text/template/parse"

// NewIdentifierNodePosition creates a RawPosition from a template parser's IdentifierNode.
// This is used when working with Go's template/parse package to convert AST nodes
// to our position system.
//
// Note: The parser's Position() is 1-based, so we subtract 1 to convert to 0-based.
func NewIdentifierNodePosition(node *parse.IdentifierNode) RawPosition {
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position() - 1),
	}
}

func NewCommentNodePosition(node *parse.CommentNode) RawPosition {
	return RawPosition{
		Text:   node.Text,
		Offset: int(node.Pos) - 1,
	}
}

// NewFieldNodePosition creates a RawPosition from a template parser's FieldNode.
// This handles field access expressions like ".Field.SubField" by focusing on
// the last identifier in the chain.
//
// Example:
//
//	{{.User.Name}} -> focuses on "Name" part
func NewFieldNodePosition(node *parse.FieldNode) RawPosition {
	// the Pos reflects just the final identifier in the field node
	// so we need to calculate the offset based on the entire field text
	ident := node.Ident[len(node.Ident)-1]
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Pos) - (len(node.String()) - len(ident)),
	}
}

// NewStringNodePosition creates a new position from a string node
func NewStringNodePosition(node *parse.StringNode) RawPosition {
	return RawPosition{
		Text:   node.Text,
		Offset: int(node.Pos),
	}
}

func NewGeneralNodePosition(node parse.Node) RawPosition {
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position()) - 1,
	}
}

// NewBranchEndPosition creates a RawPosition for an end keyword at the end of a branch node
func NewBranchEndPosition(node *parse.BranchNode) RawPosition {
	// The end position is after the last node in the list
	// If there's an else list, use that, otherwise use the main list
	list := node.List
	if node.ElseList != nil {
		list = node.ElseList
	}

	// Find the last node's position
	var lastPos parse.Pos
	if len(list.Nodes) > 0 {
		lastPos = list.Nodes[len(list.Nodes)-1].Position()
	} else {
		lastPos = list.Position()
	}

	return RawPosition{
		Text:   "end",
		Offset: int(lastPos),
	}
}

// NewCommandNodePosition creates a RawPosition from a template parser's CommandNode
func NewCommandNodePosition(node *parse.CommandNode) RawPosition {
	return RawPosition{
		Text:   node.String(),
		Offset: int(node.Position()) - 1,
	}
}

func NewKeywordPosition(keyword parse.KeywordNode) RawPosition {
	return RawPosition{Text: string(keyword.Keyword().Val()), Offset: int(keyword.Keyword().Pos()) - 1}
}

func NewTextNodePosition(node *parse.TextNode) RawPosition {
	return RawPosition{Text: string(node.Text), Offset: int(node.Pos) - 1}
}

func NewDotNodePosition(node *parse.DotNode) RawPosition {
	return RawPosition{Text: ".", Offset: int(node.Pos) - 1}
}
