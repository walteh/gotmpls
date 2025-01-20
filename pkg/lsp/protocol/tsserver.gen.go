// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code generated for LSP. DO NOT EDIT.

package protocol

// Code generated from protocol/metaModel.json at ref release/protocol/3.17.6-next.9 (hash tags/release/jsonrpc/9.0.0-next.6).
// https://github.com/microsoft/vscode-languageserver-node/blob/release/protocol/3.17.6-next.9/protocol/metaModel.json
// LSP metaData.version = 3.17.0.

import (
	"context"

	"github.com/creachadair/jrpc2/handler"
)

type Server interface {
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#progress
	Progress(context.Context, *ProgressParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#setTrace
	SetTrace(context.Context, *SetTraceParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#callHierarchy_incomingCalls
	IncomingCalls(context.Context, *CallHierarchyIncomingCallsParams) ([]CallHierarchyIncomingCall, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#callHierarchy_outgoingCalls
	OutgoingCalls(context.Context, *CallHierarchyOutgoingCallsParams) ([]CallHierarchyOutgoingCall, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#codeAction_resolve
	ResolveCodeAction(context.Context, *CodeAction) (*CodeAction, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#codeLens_resolve
	ResolveCodeLens(context.Context, *CodeLens) (*CodeLens, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#completionItem_resolve
	ResolveCompletionItem(context.Context, *CompletionItem) (*CompletionItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#documentLink_resolve
	ResolveDocumentLink(context.Context, *DocumentLink) (*DocumentLink, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#exit
	Exit(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#initialize
	Initialize(context.Context, *ParamInitialize) (*InitializeResult, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#initialized
	Initialized(context.Context, *InitializedParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#inlayHint_resolve
	Resolve(context.Context, *InlayHint) (*InlayHint, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#notebookDocument_didChange
	DidChangeNotebookDocument(context.Context, *DidChangeNotebookDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#notebookDocument_didClose
	DidCloseNotebookDocument(context.Context, *DidCloseNotebookDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#notebookDocument_didOpen
	DidOpenNotebookDocument(context.Context, *DidOpenNotebookDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#notebookDocument_didSave
	DidSaveNotebookDocument(context.Context, *DidSaveNotebookDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#shutdown
	Shutdown(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_codeAction
	CodeAction(context.Context, *CodeActionParams) ([]CodeAction, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_codeLens
	CodeLens(context.Context, *CodeLensParams) ([]CodeLens, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_colorPresentation
	ColorPresentation(context.Context, *ColorPresentationParams) ([]ColorPresentation, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_completion
	Completion(context.Context, *CompletionParams) (*CompletionList, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_declaration
	Declaration(context.Context, *DeclarationParams) (*Or_textDocument_declaration, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_definition
	Definition(context.Context, *DefinitionParams) ([]Location, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_diagnostic
	Diagnostic(context.Context, *DocumentDiagnosticParams) (*DocumentDiagnosticReport, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_didChange
	DidChange(context.Context, *DidChangeTextDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_didClose
	DidClose(context.Context, *DidCloseTextDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_didOpen
	DidOpen(context.Context, *DidOpenTextDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_didSave
	DidSave(context.Context, *DidSaveTextDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_documentColor
	DocumentColor(context.Context, *DocumentColorParams) ([]ColorInformation, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_documentHighlight
	DocumentHighlight(context.Context, *DocumentHighlightParams) ([]DocumentHighlight, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_documentLink
	DocumentLink(context.Context, *DocumentLinkParams) ([]DocumentLink, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_documentSymbol
	DocumentSymbol(context.Context, *DocumentSymbolParams) ([]any, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_foldingRange
	FoldingRange(context.Context, *FoldingRangeParams) ([]FoldingRange, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_formatting
	Formatting(context.Context, *DocumentFormattingParams) ([]TextEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_hover
	Hover(context.Context, *HoverParams) (*Hover, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_implementation
	Implementation(context.Context, *ImplementationParams) ([]Location, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_inlayHint
	InlayHint(context.Context, *InlayHintParams) ([]InlayHint, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_inlineCompletion
	InlineCompletion(context.Context, *InlineCompletionParams) (*Or_Result_textDocument_inlineCompletion, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_inlineValue
	InlineValue(context.Context, *InlineValueParams) ([]InlineValue, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_linkedEditingRange
	LinkedEditingRange(context.Context, *LinkedEditingRangeParams) (*LinkedEditingRanges, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_moniker
	Moniker(context.Context, *MonikerParams) ([]Moniker, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_onTypeFormatting
	OnTypeFormatting(context.Context, *DocumentOnTypeFormattingParams) ([]TextEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_prepareCallHierarchy
	PrepareCallHierarchy(context.Context, *CallHierarchyPrepareParams) ([]CallHierarchyItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_prepareRename
	PrepareRename(context.Context, *PrepareRenameParams) (*PrepareRenameResult, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_prepareTypeHierarchy
	PrepareTypeHierarchy(context.Context, *TypeHierarchyPrepareParams) ([]TypeHierarchyItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_rangeFormatting
	RangeFormatting(context.Context, *DocumentRangeFormattingParams) ([]TextEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_rangesFormatting
	RangesFormatting(context.Context, *DocumentRangesFormattingParams) ([]TextEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_references
	References(context.Context, *ReferenceParams) ([]Location, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_rename
	Rename(context.Context, *RenameParams) (*WorkspaceEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_selectionRange
	SelectionRange(context.Context, *SelectionRangeParams) ([]SelectionRange, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_semanticTokens_full
	SemanticTokensFull(context.Context, *SemanticTokensParams) (*SemanticTokens, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_semanticTokens_full_delta
	SemanticTokensFullDelta(context.Context, *SemanticTokensDeltaParams) (any, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_semanticTokens_range
	SemanticTokensRange(context.Context, *SemanticTokensRangeParams) (*SemanticTokens, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_signatureHelp
	SignatureHelp(context.Context, *SignatureHelpParams) (*SignatureHelp, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_typeDefinition
	TypeDefinition(context.Context, *TypeDefinitionParams) ([]Location, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_willSave
	WillSave(context.Context, *WillSaveTextDocumentParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_willSaveWaitUntil
	WillSaveWaitUntil(context.Context, *WillSaveTextDocumentParams) ([]TextEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#typeHierarchy_subtypes
	Subtypes(context.Context, *TypeHierarchySubtypesParams) ([]TypeHierarchyItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#typeHierarchy_supertypes
	Supertypes(context.Context, *TypeHierarchySupertypesParams) ([]TypeHierarchyItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_workDoneProgress_cancel
	WorkDoneProgressCancel(context.Context, *WorkDoneProgressCancelParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_diagnostic
	DiagnosticWorkspace(context.Context, *WorkspaceDiagnosticParams) (*WorkspaceDiagnosticReport, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didChangeConfiguration
	DidChangeConfiguration(context.Context, *DidChangeConfigurationParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didChangeWatchedFiles
	DidChangeWatchedFiles(context.Context, *DidChangeWatchedFilesParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didChangeWorkspaceFolders
	DidChangeWorkspaceFolders(context.Context, *DidChangeWorkspaceFoldersParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didCreateFiles
	DidCreateFiles(context.Context, *CreateFilesParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didDeleteFiles
	DidDeleteFiles(context.Context, *DeleteFilesParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_didRenameFiles
	DidRenameFiles(context.Context, *RenameFilesParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_executeCommand
	ExecuteCommand(context.Context, *ExecuteCommandParams) (any, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_symbol
	Symbol(context.Context, *WorkspaceSymbolParams) ([]SymbolInformation, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_textDocumentContent
	TextDocumentContent(context.Context, *TextDocumentContentParams) (*string, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_willCreateFiles
	WillCreateFiles(context.Context, *CreateFilesParams) (*WorkspaceEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_willDeleteFiles
	WillDeleteFiles(context.Context, *DeleteFilesParams) (*WorkspaceEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_willRenameFiles
	WillRenameFiles(context.Context, *RenameFilesParams) (*WorkspaceEdit, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspaceSymbol_resolve
	ResolveWorkspaceSymbol(context.Context, *WorkspaceSymbol) (*WorkspaceSymbol, error)
}

func buildServerDispatchMap(server Server) handler.Map {
	return handler.Map{
		"$/progress":                             createEmptyResultHandler(server.Progress),
		"$/setTrace":                             createEmptyResultHandler(server.SetTrace),
		"callHierarchy/incomingCalls":            createHandler(server.IncomingCalls),
		"callHierarchy/outgoingCalls":            createHandler(server.OutgoingCalls),
		"codeAction/resolve":                     createHandler(server.ResolveCodeAction),
		"codeLens/resolve":                       createHandler(server.ResolveCodeLens),
		"completionItem/resolve":                 createHandler(server.ResolveCompletionItem),
		"documentLink/resolve":                   createHandler(server.ResolveDocumentLink),
		"exit":                                   createEmptyHandler(server.Exit),
		"initialize":                             createHandler(server.Initialize),
		"initialized":                            createEmptyResultHandler(server.Initialized),
		"inlayHint/resolve":                      createHandler(server.Resolve),
		"notebookDocument/didChange":             createEmptyResultHandler(server.DidChangeNotebookDocument),
		"notebookDocument/didClose":              createEmptyResultHandler(server.DidCloseNotebookDocument),
		"notebookDocument/didOpen":               createEmptyResultHandler(server.DidOpenNotebookDocument),
		"notebookDocument/didSave":               createEmptyResultHandler(server.DidSaveNotebookDocument),
		"shutdown":                               createEmptyHandler(server.Shutdown),
		"textDocument/codeAction":                createHandler(server.CodeAction),
		"textDocument/codeLens":                  createHandler(server.CodeLens),
		"textDocument/colorPresentation":         createHandler(server.ColorPresentation),
		"textDocument/completion":                createHandler(server.Completion),
		"textDocument/declaration":               createHandler(server.Declaration),
		"textDocument/definition":                createHandler(server.Definition),
		"textDocument/diagnostic":                createHandler(server.Diagnostic),
		"textDocument/didChange":                 createEmptyResultHandler(server.DidChange),
		"textDocument/didClose":                  createEmptyResultHandler(server.DidClose),
		"textDocument/didOpen":                   createEmptyResultHandler(server.DidOpen),
		"textDocument/didSave":                   createEmptyResultHandler(server.DidSave),
		"textDocument/documentColor":             createHandler(server.DocumentColor),
		"textDocument/documentHighlight":         createHandler(server.DocumentHighlight),
		"textDocument/documentLink":              createHandler(server.DocumentLink),
		"textDocument/documentSymbol":            createHandler(server.DocumentSymbol),
		"textDocument/foldingRange":              createHandler(server.FoldingRange),
		"textDocument/formatting":                createHandler(server.Formatting),
		"textDocument/hover":                     createHandler(server.Hover),
		"textDocument/implementation":            createHandler(server.Implementation),
		"textDocument/inlayHint":                 createHandler(server.InlayHint),
		"textDocument/inlineCompletion":          createHandler(server.InlineCompletion),
		"textDocument/inlineValue":               createHandler(server.InlineValue),
		"textDocument/linkedEditingRange":        createHandler(server.LinkedEditingRange),
		"textDocument/moniker":                   createHandler(server.Moniker),
		"textDocument/onTypeFormatting":          createHandler(server.OnTypeFormatting),
		"textDocument/prepareCallHierarchy":      createHandler(server.PrepareCallHierarchy),
		"textDocument/prepareRename":             createHandler(server.PrepareRename),
		"textDocument/prepareTypeHierarchy":      createHandler(server.PrepareTypeHierarchy),
		"textDocument/rangeFormatting":           createHandler(server.RangeFormatting),
		"textDocument/rangesFormatting":          createHandler(server.RangesFormatting),
		"textDocument/references":                createHandler(server.References),
		"textDocument/rename":                    createHandler(server.Rename),
		"textDocument/selectionRange":            createHandler(server.SelectionRange),
		"textDocument/semanticTokens/full":       createHandler(server.SemanticTokensFull),
		"textDocument/semanticTokens/full/delta": createHandler(server.SemanticTokensFullDelta),
		"textDocument/semanticTokens/range":      createHandler(server.SemanticTokensRange),
		"textDocument/signatureHelp":             createHandler(server.SignatureHelp),
		"textDocument/typeDefinition":            createHandler(server.TypeDefinition),
		"textDocument/willSave":                  createEmptyResultHandler(server.WillSave),
		"textDocument/willSaveWaitUntil":         createHandler(server.WillSaveWaitUntil),
		"typeHierarchy/subtypes":                 createHandler(server.Subtypes),
		"typeHierarchy/supertypes":               createHandler(server.Supertypes),
		"window/workDoneProgress/cancel":         createEmptyResultHandler(server.WorkDoneProgressCancel),
		"workspace/diagnostic":                   createHandler(server.DiagnosticWorkspace),
		"workspace/didChangeConfiguration":       createEmptyResultHandler(server.DidChangeConfiguration),
		"workspace/didChangeWatchedFiles":        createEmptyResultHandler(server.DidChangeWatchedFiles),
		"workspace/didChangeWorkspaceFolders":    createEmptyResultHandler(server.DidChangeWorkspaceFolders),
		"workspace/didCreateFiles":               createEmptyResultHandler(server.DidCreateFiles),
		"workspace/didDeleteFiles":               createEmptyResultHandler(server.DidDeleteFiles),
		"workspace/didRenameFiles":               createEmptyResultHandler(server.DidRenameFiles),
		"workspace/executeCommand":               createHandler(server.ExecuteCommand),
		"workspace/symbol":                       createHandler(server.Symbol),
		"workspace/textDocumentContent":          createHandler(server.TextDocumentContent),
		"workspace/willCreateFiles":              createHandler(server.WillCreateFiles),
		"workspace/willDeleteFiles":              createHandler(server.WillDeleteFiles),
		"workspace/willRenameFiles":              createHandler(server.WillRenameFiles),
		"workspaceSymbol/resolve":                createHandler(server.ResolveWorkspaceSymbol),
	}
}

func (s *ServerDispatcher) Progress(ctx context.Context, params *ProgressParams) error {
	return createNotify(ctx, s, "$/progress", params)
}
func (s *ServerDispatcher) SetTrace(ctx context.Context, params *SetTraceParams) error {
	return createNotify(ctx, s, "$/setTrace", params)
}
func (s *ServerDispatcher) IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) ([]CallHierarchyIncomingCall, error) {
	var result []CallHierarchyIncomingCall
	if err := createCallback(ctx, s, "callHierarchy/incomingCalls", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) ([]CallHierarchyOutgoingCall, error) {
	var result []CallHierarchyOutgoingCall
	if err := createCallback(ctx, s, "callHierarchy/outgoingCalls", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ResolveCodeAction(ctx context.Context, params *CodeAction) (*CodeAction, error) {
	var result *CodeAction
	if err := createCallback(ctx, s, "codeAction/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ResolveCodeLens(ctx context.Context, params *CodeLens) (*CodeLens, error) {
	var result *CodeLens
	if err := createCallback(ctx, s, "codeLens/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ResolveCompletionItem(ctx context.Context, params *CompletionItem) (*CompletionItem, error) {
	var result *CompletionItem
	if err := createCallback(ctx, s, "completionItem/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ResolveDocumentLink(ctx context.Context, params *DocumentLink) (*DocumentLink, error) {
	var result *DocumentLink
	if err := createCallback(ctx, s, "documentLink/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Exit(ctx context.Context) error {
	return createEmptyNotify(ctx, s, "exit")
}
func (s *ServerDispatcher) Initialize(ctx context.Context, params *ParamInitialize) (*InitializeResult, error) {
	var result *InitializeResult
	if err := createCallback(ctx, s, "initialize", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Initialized(ctx context.Context, params *InitializedParams) error {
	return createNotify(ctx, s, "initialized", params)
}
func (s *ServerDispatcher) Resolve(ctx context.Context, params *InlayHint) (*InlayHint, error) {
	var result *InlayHint
	if err := createCallback(ctx, s, "inlayHint/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DidChangeNotebookDocument(ctx context.Context, params *DidChangeNotebookDocumentParams) error {
	return createNotify(ctx, s, "notebookDocument/didChange", params)
}
func (s *ServerDispatcher) DidCloseNotebookDocument(ctx context.Context, params *DidCloseNotebookDocumentParams) error {
	return createNotify(ctx, s, "notebookDocument/didClose", params)
}
func (s *ServerDispatcher) DidOpenNotebookDocument(ctx context.Context, params *DidOpenNotebookDocumentParams) error {
	return createNotify(ctx, s, "notebookDocument/didOpen", params)
}
func (s *ServerDispatcher) DidSaveNotebookDocument(ctx context.Context, params *DidSaveNotebookDocumentParams) error {
	return createNotify(ctx, s, "notebookDocument/didSave", params)
}
func (s *ServerDispatcher) Shutdown(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "shutdown")
}
func (s *ServerDispatcher) CodeAction(ctx context.Context, params *CodeActionParams) ([]CodeAction, error) {
	var result []CodeAction
	if err := createCallback(ctx, s, "textDocument/codeAction", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) CodeLens(ctx context.Context, params *CodeLensParams) ([]CodeLens, error) {
	var result []CodeLens
	if err := createCallback(ctx, s, "textDocument/codeLens", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ColorPresentation(ctx context.Context, params *ColorPresentationParams) ([]ColorPresentation, error) {
	var result []ColorPresentation
	if err := createCallback(ctx, s, "textDocument/colorPresentation", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Completion(ctx context.Context, params *CompletionParams) (*CompletionList, error) {
	var result *CompletionList
	if err := createCallback(ctx, s, "textDocument/completion", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Declaration(ctx context.Context, params *DeclarationParams) (*Or_textDocument_declaration, error) {
	var result *Or_textDocument_declaration
	if err := createCallback(ctx, s, "textDocument/declaration", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Definition(ctx context.Context, params *DefinitionParams) ([]Location, error) {
	var result []Location
	if err := createCallback(ctx, s, "textDocument/definition", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Diagnostic(ctx context.Context, params *DocumentDiagnosticParams) (*DocumentDiagnosticReport, error) {
	var result *DocumentDiagnosticReport
	if err := createCallback(ctx, s, "textDocument/diagnostic", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DidChange(ctx context.Context, params *DidChangeTextDocumentParams) error {
	return createNotify(ctx, s, "textDocument/didChange", params)
}
func (s *ServerDispatcher) DidClose(ctx context.Context, params *DidCloseTextDocumentParams) error {
	return createNotify(ctx, s, "textDocument/didClose", params)
}
func (s *ServerDispatcher) DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) error {
	return createNotify(ctx, s, "textDocument/didOpen", params)
}
func (s *ServerDispatcher) DidSave(ctx context.Context, params *DidSaveTextDocumentParams) error {
	return createNotify(ctx, s, "textDocument/didSave", params)
}
func (s *ServerDispatcher) DocumentColor(ctx context.Context, params *DocumentColorParams) ([]ColorInformation, error) {
	var result []ColorInformation
	if err := createCallback(ctx, s, "textDocument/documentColor", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) ([]DocumentHighlight, error) {
	var result []DocumentHighlight
	if err := createCallback(ctx, s, "textDocument/documentHighlight", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DocumentLink(ctx context.Context, params *DocumentLinkParams) ([]DocumentLink, error) {
	var result []DocumentLink
	if err := createCallback(ctx, s, "textDocument/documentLink", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) ([]any, error) {
	var result []any
	if err := createCallback(ctx, s, "textDocument/documentSymbol", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) FoldingRange(ctx context.Context, params *FoldingRangeParams) ([]FoldingRange, error) {
	var result []FoldingRange
	if err := createCallback(ctx, s, "textDocument/foldingRange", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Formatting(ctx context.Context, params *DocumentFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := createCallback(ctx, s, "textDocument/formatting", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Hover(ctx context.Context, params *HoverParams) (*Hover, error) {
	var result *Hover
	if err := createCallback(ctx, s, "textDocument/hover", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Implementation(ctx context.Context, params *ImplementationParams) ([]Location, error) {
	var result []Location
	if err := createCallback(ctx, s, "textDocument/implementation", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) InlayHint(ctx context.Context, params *InlayHintParams) ([]InlayHint, error) {
	var result []InlayHint
	if err := createCallback(ctx, s, "textDocument/inlayHint", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) InlineCompletion(ctx context.Context, params *InlineCompletionParams) (*Or_Result_textDocument_inlineCompletion, error) {
	var result *Or_Result_textDocument_inlineCompletion
	if err := createCallback(ctx, s, "textDocument/inlineCompletion", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) InlineValue(ctx context.Context, params *InlineValueParams) ([]InlineValue, error) {
	var result []InlineValue
	if err := createCallback(ctx, s, "textDocument/inlineValue", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (*LinkedEditingRanges, error) {
	var result *LinkedEditingRanges
	if err := createCallback(ctx, s, "textDocument/linkedEditingRange", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Moniker(ctx context.Context, params *MonikerParams) ([]Moniker, error) {
	var result []Moniker
	if err := createCallback(ctx, s, "textDocument/moniker", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := createCallback(ctx, s, "textDocument/onTypeFormatting", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) ([]CallHierarchyItem, error) {
	var result []CallHierarchyItem
	if err := createCallback(ctx, s, "textDocument/prepareCallHierarchy", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) PrepareRename(ctx context.Context, params *PrepareRenameParams) (*PrepareRenameResult, error) {
	var result *PrepareRenameResult
	if err := createCallback(ctx, s, "textDocument/prepareRename", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) PrepareTypeHierarchy(ctx context.Context, params *TypeHierarchyPrepareParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := createCallback(ctx, s, "textDocument/prepareTypeHierarchy", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := createCallback(ctx, s, "textDocument/rangeFormatting", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) RangesFormatting(ctx context.Context, params *DocumentRangesFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := createCallback(ctx, s, "textDocument/rangesFormatting", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) References(ctx context.Context, params *ReferenceParams) ([]Location, error) {
	var result []Location
	if err := createCallback(ctx, s, "textDocument/references", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Rename(ctx context.Context, params *RenameParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := createCallback(ctx, s, "textDocument/rename", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) SelectionRange(ctx context.Context, params *SelectionRangeParams) ([]SelectionRange, error) {
	var result []SelectionRange
	if err := createCallback(ctx, s, "textDocument/selectionRange", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (*SemanticTokens, error) {
	var result *SemanticTokens
	if err := createCallback(ctx, s, "textDocument/semanticTokens/full", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (any, error) {
	var result any
	if err := createCallback(ctx, s, "textDocument/semanticTokens/full/delta", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (*SemanticTokens, error) {
	var result *SemanticTokens
	if err := createCallback(ctx, s, "textDocument/semanticTokens/range", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) SignatureHelp(ctx context.Context, params *SignatureHelpParams) (*SignatureHelp, error) {
	var result *SignatureHelp
	if err := createCallback(ctx, s, "textDocument/signatureHelp", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) TypeDefinition(ctx context.Context, params *TypeDefinitionParams) ([]Location, error) {
	var result []Location
	if err := createCallback(ctx, s, "textDocument/typeDefinition", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) WillSave(ctx context.Context, params *WillSaveTextDocumentParams) error {
	return createNotify(ctx, s, "textDocument/willSave", params)
}
func (s *ServerDispatcher) WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := createCallback(ctx, s, "textDocument/willSaveWaitUntil", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Subtypes(ctx context.Context, params *TypeHierarchySubtypesParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := createCallback(ctx, s, "typeHierarchy/subtypes", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Supertypes(ctx context.Context, params *TypeHierarchySupertypesParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := createCallback(ctx, s, "typeHierarchy/supertypes", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) error {
	return createNotify(ctx, s, "window/workDoneProgress/cancel", params)
}
func (s *ServerDispatcher) DiagnosticWorkspace(ctx context.Context, params *WorkspaceDiagnosticParams) (*WorkspaceDiagnosticReport, error) {
	var result *WorkspaceDiagnosticReport
	if err := createCallback(ctx, s, "workspace/diagnostic", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) error {
	return createNotify(ctx, s, "workspace/didChangeConfiguration", params)
}
func (s *ServerDispatcher) DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) error {
	return createNotify(ctx, s, "workspace/didChangeWatchedFiles", params)
}
func (s *ServerDispatcher) DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) error {
	return createNotify(ctx, s, "workspace/didChangeWorkspaceFolders", params)
}
func (s *ServerDispatcher) DidCreateFiles(ctx context.Context, params *CreateFilesParams) error {
	return createNotify(ctx, s, "workspace/didCreateFiles", params)
}
func (s *ServerDispatcher) DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) error {
	return createNotify(ctx, s, "workspace/didDeleteFiles", params)
}
func (s *ServerDispatcher) DidRenameFiles(ctx context.Context, params *RenameFilesParams) error {
	return createNotify(ctx, s, "workspace/didRenameFiles", params)
}
func (s *ServerDispatcher) ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (any, error) {
	var result any
	if err := createCallback(ctx, s, "workspace/executeCommand", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) Symbol(ctx context.Context, params *WorkspaceSymbolParams) ([]SymbolInformation, error) {
	var result []SymbolInformation
	if err := createCallback(ctx, s, "workspace/symbol", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) TextDocumentContent(ctx context.Context, params *TextDocumentContentParams) (*string, error) {
	var result *string
	if err := createCallback(ctx, s, "workspace/textDocumentContent", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) WillCreateFiles(ctx context.Context, params *CreateFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := createCallback(ctx, s, "workspace/willCreateFiles", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := createCallback(ctx, s, "workspace/willDeleteFiles", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) WillRenameFiles(ctx context.Context, params *RenameFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := createCallback(ctx, s, "workspace/willRenameFiles", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ServerDispatcher) ResolveWorkspaceSymbol(ctx context.Context, params *WorkspaceSymbol) (*WorkspaceSymbol, error) {
	var result *WorkspaceSymbol
	if err := createCallback(ctx, s, "workspaceSymbol/resolve", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
