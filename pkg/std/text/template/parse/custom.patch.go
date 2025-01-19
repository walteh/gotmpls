package parse

type KeywordNode interface {
	Keyword() item
}

func (i item) Pos() Pos {
	return i.pos
}

func (i item) Val() string {
	return i.val
}

func (i item) Typ() itemType {
	return i.typ
}

func (i item) Line() int {
	return i.line
}

var _ KeywordNode = (*TemplateNode)(nil)
var _ KeywordNode = (*BreakNode)(nil)
var _ KeywordNode = (*ContinueNode)(nil)
var _ KeywordNode = (*elseNode)(nil)
var _ KeywordNode = (*EndNode)(nil)
var _ KeywordNode = (*NilNode)(nil)
var _ KeywordNode = (*TemplateNode)(nil)
var _ KeywordNode = (*BranchNode)(nil)
var _ KeywordNode = (*WithNode)(nil)
var _ KeywordNode = (*RangeNode)(nil)
var _ KeywordNode = (*IfNode)(nil)
var _ KeywordNode = (*Tree)(nil) // the "define" node

func (me *TemplateNode) Keyword() item { return me.keyword }
func (me *BreakNode) Keyword() item    { return me.keyword }
func (me *ContinueNode) Keyword() item { return me.keyword }

// covers if, range, with nodes too
func (me *BranchNode) Keyword() item    { return me.keyword }
func (me *elseNode) Keyword() item      { return me.keyword }
func (me *EndNode) Keyword() item       { return me.keyword }
func (me *NilNode) Keyword() item       { return me.keyword }
func (me *Tree) Keyword() item          { return me.defineNodeKeyword }
func (me *Tree) CanRelyOnKeyword() bool { return me.hasDefineNode }
