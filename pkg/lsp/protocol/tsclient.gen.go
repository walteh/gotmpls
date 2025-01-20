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

type Client interface {
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#logTrace
	LogTrace(context.Context, *LogTraceParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#progress
	Progress(context.Context, *ProgressParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#client_registerCapability
	RegisterCapability(context.Context, *RegistrationParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#client_unregisterCapability
	UnregisterCapability(context.Context, *UnregistrationParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#telemetry_event
	Event(context.Context, *any) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#textDocument_publishDiagnostics
	PublishDiagnostics(context.Context, *PublishDiagnosticsParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_logMessage
	LogMessage(context.Context, *LogMessageParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_showDocument
	ShowDocument(context.Context, *ShowDocumentParams) (*ShowDocumentResult, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_showMessage
	ShowMessage(context.Context, *ShowMessageParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_showMessageRequest
	ShowMessageRequest(context.Context, *ShowMessageRequestParams) (*MessageActionItem, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#window_workDoneProgress_create
	WorkDoneProgressCreate(context.Context, *WorkDoneProgressCreateParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_applyEdit
	ApplyEdit(context.Context, *ApplyWorkspaceEditParams) (*ApplyWorkspaceEditResult, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_codeLens_refresh
	CodeLensRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_configuration
	Configuration(context.Context, *ParamConfiguration) ([]LSPAny, error)
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_diagnostic_refresh
	DiagnosticRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_foldingRange_refresh
	FoldingRangeRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_inlayHint_refresh
	InlayHintRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_inlineValue_refresh
	InlineValueRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_semanticTokens_refresh
	SemanticTokensRefresh(context.Context) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_textDocumentContent_refresh
	TextDocumentContentRefresh(context.Context, *TextDocumentContentRefreshParams) error
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification#workspace_workspaceFolders
	WorkspaceFolders(context.Context) ([]WorkspaceFolder, error)
}

func buildClientDispatchMap(client Client) handler.Map {
	return handler.Map{
		"$/logTrace":                            createEmptyResultHandler(client.LogTrace),
		"$/progress":                            createEmptyResultHandler(client.Progress),
		"client/registerCapability":             createEmptyResultHandler(client.RegisterCapability),
		"client/unregisterCapability":           createEmptyResultHandler(client.UnregisterCapability),
		"telemetry/event":                       createEmptyResultHandler(client.Event),
		"textDocument/publishDiagnostics":       createEmptyResultHandler(client.PublishDiagnostics),
		"window/logMessage":                     createEmptyResultHandler(client.LogMessage),
		"window/showDocument":                   createHandler(client.ShowDocument),
		"window/showMessage":                    createEmptyResultHandler(client.ShowMessage),
		"window/showMessageRequest":             createHandler(client.ShowMessageRequest),
		"window/workDoneProgress/create":        createEmptyResultHandler(client.WorkDoneProgressCreate),
		"workspace/applyEdit":                   createHandler(client.ApplyEdit),
		"workspace/codeLens/refresh":            createEmptyHandler(client.CodeLensRefresh),
		"workspace/configuration":               createHandler(client.Configuration),
		"workspace/diagnostic/refresh":          createEmptyHandler(client.DiagnosticRefresh),
		"workspace/foldingRange/refresh":        createEmptyHandler(client.FoldingRangeRefresh),
		"workspace/inlayHint/refresh":           createEmptyHandler(client.InlayHintRefresh),
		"workspace/inlineValue/refresh":         createEmptyHandler(client.InlineValueRefresh),
		"workspace/semanticTokens/refresh":      createEmptyHandler(client.SemanticTokensRefresh),
		"workspace/textDocumentContent/refresh": createEmptyResultHandler(client.TextDocumentContentRefresh),
		"workspace/workspaceFolders":            createEmptyParamsHandler(client.WorkspaceFolders),
	}
}

func (s *ClientDispatcher) LogTrace(ctx context.Context, params *LogTraceParams) error {
	return createNotify(ctx, s, "$/logTrace", params)
}
func (s *ClientDispatcher) Progress(ctx context.Context, params *ProgressParams) error {
	return createNotify(ctx, s, "$/progress", params)
}
func (s *ClientDispatcher) RegisterCapability(ctx context.Context, params *RegistrationParams) error {
	return createEmptyResultCallback(ctx, s, "client/registerCapability", params)
}
func (s *ClientDispatcher) UnregisterCapability(ctx context.Context, params *UnregistrationParams) error {
	return createEmptyResultCallback(ctx, s, "client/unregisterCapability", params)
}
func (s *ClientDispatcher) Event(ctx context.Context, params *any) error {
	return createNotify(ctx, s, "telemetry/event", params)
}
func (s *ClientDispatcher) PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) error {
	return createNotify(ctx, s, "textDocument/publishDiagnostics", params)
}
func (s *ClientDispatcher) LogMessage(ctx context.Context, params *LogMessageParams) error {
	return createNotify(ctx, s, "window/logMessage", params)
}
func (s *ClientDispatcher) ShowDocument(ctx context.Context, params *ShowDocumentParams) (*ShowDocumentResult, error) {
	var result *ShowDocumentResult
	if err := createCallback(ctx, s, "window/showDocument", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ClientDispatcher) ShowMessage(ctx context.Context, params *ShowMessageParams) error {
	return createNotify(ctx, s, "window/showMessage", params)
}
func (s *ClientDispatcher) ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (*MessageActionItem, error) {
	var result *MessageActionItem
	if err := createCallback(ctx, s, "window/showMessageRequest", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ClientDispatcher) WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) error {
	return createEmptyResultCallback(ctx, s, "window/workDoneProgress/create", params)
}
func (s *ClientDispatcher) ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (*ApplyWorkspaceEditResult, error) {
	var result *ApplyWorkspaceEditResult
	if err := createCallback(ctx, s, "workspace/applyEdit", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ClientDispatcher) CodeLensRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/codeLens/refresh")
}
func (s *ClientDispatcher) Configuration(ctx context.Context, params *ParamConfiguration) ([]LSPAny, error) {
	var result []LSPAny
	if err := createCallback(ctx, s, "workspace/configuration", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *ClientDispatcher) DiagnosticRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/diagnostic/refresh")
}
func (s *ClientDispatcher) FoldingRangeRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/foldingRange/refresh")
}
func (s *ClientDispatcher) InlayHintRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/inlayHint/refresh")
}
func (s *ClientDispatcher) InlineValueRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/inlineValue/refresh")
}
func (s *ClientDispatcher) SemanticTokensRefresh(ctx context.Context) error {
	return createEmptyCallback(ctx, s, "workspace/semanticTokens/refresh")
}
func (s *ClientDispatcher) TextDocumentContentRefresh(ctx context.Context, params *TextDocumentContentRefreshParams) error {
	return createEmptyResultCallback(ctx, s, "workspace/textDocumentContent/refresh", params)
}
func (s *ClientDispatcher) WorkspaceFolders(ctx context.Context) ([]WorkspaceFolder, error) {
	var result []WorkspaceFolder
	if err := createEmptyParamsCallback(ctx, s, "workspace/workspaceFolders", &result); err != nil {
		return nil, err
	}
	return result, nil
}
