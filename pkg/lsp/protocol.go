package lsp

// MarkupContent represents a marked up content.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// CompletionContext represents additional information about the context in which a completion request is triggered.
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind represents how a completion was triggered.
type CompletionTriggerKind int

const (
	// Invoked indicates the completion was triggered by typing an identifier.
	CompletionTriggerInvoked CompletionTriggerKind = 1
	// TriggerCharacter indicates the completion was triggered by a trigger character.
	CompletionTriggerCharacter CompletionTriggerKind = 2
	// TriggerForIncompleteCompletions indicates the completion was re-triggered as the current completion list is incomplete.
	CompletionTriggerIncomplete CompletionTriggerKind = 3
)

// CompletionItemKind represents the kind of a completion item.
type CompletionItemKind int

const (
	CompletionItemText          = 1
	CompletionItemMethod        = 2
	CompletionItemFunction      = 3
	CompletionItemConstructor   = 4
	CompletionItemField         = 5
	CompletionItemVariable      = 6
	CompletionItemClass         = 7
	CompletionItemInterface     = 8
	CompletionItemModule        = 9
	CompletionItemProperty      = 10
	CompletionItemUnit          = 11
	CompletionItemValue         = 12
	CompletionItemEnum          = 13
	CompletionItemKeyword       = 14
	CompletionItemSnippet       = 15
	CompletionItemColor         = 16
	CompletionItemFile          = 17
	CompletionItemReference     = 18
	CompletionItemFolder        = 19
	CompletionItemEnumMember    = 20
	CompletionItemConstant      = 21
	CompletionItemStruct        = 22
	CompletionItemEvent         = 23
	CompletionItemOperator      = 24
	CompletionItemTypeParameter = 25
)

// InsertTextFormat represents the format of the insert text.
type InsertTextFormat int

const (
	InsertTextFormatPlainText InsertTextFormat = 1
	InsertTextFormatSnippet   InsertTextFormat = 2
)

// Command represents a reference to a command.
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// TextEdit represents a textual edit applicable to a text document.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// CompletionItem represents a completion item.
type CompletionItem struct {
	Label               string             `json:"label"`
	Kind                CompletionItemKind `json:"kind,omitempty"`
	Detail              string             `json:"detail,omitempty"`
	Documentation       interface{}        `json:"documentation,omitempty"`
	Deprecated          bool               `json:"deprecated,omitempty"`
	Preselect           bool               `json:"preselect,omitempty"`
	SortText            string             `json:"sortText,omitempty"`
	FilterText          string             `json:"filterText,omitempty"`
	InsertText          string             `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat   `json:"insertTextFormat,omitempty"`
	TextEdit            *TextEdit          `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit         `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string           `json:"commitCharacters,omitempty"`
	Command             *Command           `json:"command,omitempty"`
	Data                interface{}        `json:"data,omitempty"`
}
