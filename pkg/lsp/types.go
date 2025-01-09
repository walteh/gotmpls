package lsp

import (
	"encoding/json"

	"github.com/rs/zerolog"
)

// LSP types based on the specification
// https://microsoft.github.io/language-server-protocol/specifications/specification-current/

// MessageType represents the type of a message
type MessageType int

const (
	Error      MessageType = 1
	Warning    MessageType = 2
	Info       MessageType = 3
	Debug      MessageType = 4
	Trace      MessageType = 5
	Dependency MessageType = 6
	Unknown    MessageType = 7
)

func (mt MessageType) String() string {
	switch mt {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Info:
		return "info"
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	case Dependency:
		return "dependency"
	default:
		return "unknown"
	}
}

// LogMessageParams represents the parameters for a window/logMessage notification
type LogMessageParams struct {
	Type    MessageType    `json:"type"`
	Message string         `json:"message"`
	Source  string         `json:"source"`
	Raw     string         `json:"raw"`
	Extra   map[string]any `json:"extra"`
	Time    string         `json:"time"`
}

func MustParseLogMessageParams(msg any) LogMessageParams {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	var params LogMessageParams
	err = json.Unmarshal(msgBytes, &params)
	if err != nil {
		panic(err)
	}
	return params
}

func ParseMessageTypeFromZerolog(level string) MessageType {
	zlgLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return Unknown
	}
	switch zlgLevel {
	case zerolog.InfoLevel:
		return Info
	case zerolog.ErrorLevel:
		return Error
	case zerolog.WarnLevel:
		return Warning
	case zerolog.DebugLevel:
		return Debug
	case zerolog.TraceLevel:
		return Trace
	default:
		return Unknown
	}
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

func (ds DiagnosticSeverity) String() string {
	switch ds {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInformation:
		return "information"
	case SeverityHint:
		return "hint"
	default:
		return "unknown"
	}
}

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

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}
