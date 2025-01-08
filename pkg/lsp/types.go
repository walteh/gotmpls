package lsp

// LSP types based on the specification
// https://microsoft.github.io/language-server-protocol/specifications/specification-current/

type InitializeParams struct {
	ProcessID             int         `json:"processId,omitempty"`
	RootURI               string      `json:"rootUri"`
	InitializationOptions interface{} `json:"initializationOptions,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync   TextDocumentSyncKind `json:"textDocumentSync"`
	HoverProvider      bool                 `json:"hoverProvider"`
	CompletionProvider CompletionOptions    `json:"completionProvider"`
}

type TextDocumentSyncKind struct {
	Change int `json:"change"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}
