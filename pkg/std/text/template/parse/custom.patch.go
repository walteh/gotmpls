package parse

type DelimitedNode interface {
	Delims() (item, item)
}

// pure go
// var _ DelimitedNode = (*ActionNode)(nil)
// var _ DelimitedNode = (*CommentNode)(nil)
var _ DelimitedNode = (*BranchNode)(nil)

// less pure

var _ DelimitedNode = (*BreakNode)(nil)
var _ DelimitedNode = (*ContinueNode)(nil)
var _ DelimitedNode = (*elseNode)(nil)
var _ DelimitedNode = (*EndNode)(nil)
var _ DelimitedNode = (*NilNode)(nil)
var _ DelimitedNode = (*TemplateNode)(nil)
var _ DelimitedNode = (*BranchNode)(nil)
var _ DelimitedNode = (*WithNode)(nil)
var _ DelimitedNode = (*RangeNode)(nil)
var _ DelimitedNode = (*IfNode)(nil)

// func (me *ActionNode) Delims() (item, item)  { return me.keyword, me.keyword }
// func (me *CommentNode) Delims() (item, item) { return me.keyword, me.keyword }
func (me *BranchNode) Delims() (item, item) { return me.keyword, me.keyword }

func (me *BreakNode) Delims() (item, item)    { return me.keyword, me.keyword }
func (me *ContinueNode) Delims() (item, item) { return me.keyword, me.keyword }
func (me *elseNode) Delims() (item, item)     { return me.keyword, me.keyword }
func (me *EndNode) Delims() (item, item)      { return me.keyword, me.keyword }
func (me *NilNode) Delims() (item, item)      { return me.keyword, me.keyword }
func (me *TemplateNode) Delims() (item, item) { return me.keyword, me.keyword }

func (t *Tree) lastItemReadOnly() item {
	t.lex.backup()
	item := t.lex.item
	t.lex.next()
	return item
}

func (t *Tree) nextItemReadOnly() item {
	t.lex.next()
	item := t.lex.item
	t.lex.backup()
	return item
}

func (t *Tree) thisItemReadOnly() item {
	return t.lex.item
}

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

func (me *TemplateNode) Keyword() item { return me.keyword }
func (me *BreakNode) Keyword() item    { return me.keyword }
func (me *ContinueNode) Keyword() item { return me.keyword }

// covers if, range, with nodes too
func (me *BranchNode) Keyword() item { return me.keyword }
func (me *elseNode) Keyword() item   { return me.keyword }
func (me *EndNode) Keyword() item    { return me.keyword }
func (me *NilNode) Keyword() item    { return me.keyword }
