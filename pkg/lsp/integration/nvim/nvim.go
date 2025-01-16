package nvim

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/neovim/go-client/nvim"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

// Helper function to convert [][]byte to []string
func bytesSliceToStringSlice(b [][]byte) []string {
	s := make([]string, len(b))
	for i, v := range b {
		s[i] = string(v)
	}
	return s
}

func lastN[T any](vals []T, n int) []T {
	if len(vals) <= n {
		return vals
	}
	return vals[len(vals)-n:]
}

// // withFile ensures atomic and consistent file handling for LSP operations
// func (s *NvimIntegrationTestRunner) withFile(uri protocol.DocumentURI, operation func(buffer nvim.Buffer) error) error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	// Get buffer for URI
// 	buffers, err := s.nvimInstance.Buffers()
// 	if err != nil {
// 		return errors.Errorf("failed to get buffers: %w", err)
// 	}

// 	var buffer nvim.Buffer
// 	for _, b := range buffers {
// 		name, err := s.nvimInstance.BufferName(b)
// 		if err != nil {
// 			continue
// 		}
// 		if strings.HasSuffix(name, string(uri)) {
// 			buffer = b
// 			break
// 		}
// 	}

// 	// If buffer not found, try to open the file
// 	if buffer == 0 {
// 		buffer, cleanup, err = s.OpenFileWithLock(s.t, uri)
// 		if err != nil {
// 			return errors.Errorf("failed to open file: %w", err)
// 		}

// 		// Attach LSP
// 		if err := s.attachLSP(buffer); err != nil {
// 			return errors.Errorf("failed to attach LSP: %w", err)
// 		}
// 	}

// 	// Execute the function with the buffer
// 	return operation(buffer)
// }

// Helper method to wait for LSP to initialize
func (s *NvimIntegrationTestRunner) WaitForLSP(t *testing.T) error {
	t.Helper()
	prelogCount := 0
	waitForLSP := func() bool {
		prelogCount++
		var hasClients bool
		err := s.nvimInstance.Eval(`luaeval('vim.lsp.get_active_clients() ~= nil and #vim.lsp.get_active_clients() > 0')`, &hasClients)
		if err != nil {
			s.t.Logf("Error checking LSP clients: %v", err)
			return false
		}

		if prelogCount > 10 {
			s.t.Logf("LSP clients count: %v", hasClients)
		}

		if hasClients {
			// Log client info for debugging
			var clientInfo string
			err = s.nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_active_clients())')`, &clientInfo)
			if err == nil {
				// s.t.Logf("LSP client info: %v", clientInfo)
			}
		}

		return hasClients
	}

	var success bool
	for i := 0; i < 50; i++ {
		if success = waitForLSP(); success {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !success {
		return errors.Errorf("LSP client failed to attach")
	}
	return nil
}

func (s *NvimIntegrationTestRunner) attachLSP(t *testing.T, buf nvim.Buffer) error {
	t.Helper()

	if err := s.WaitForLSP(t); err != nil {
		return errors.Errorf("failed to wait for LSP: %w", err)
	}

	luaCmd := fmt.Sprintf(`
local client = vim.lsp.get_active_clients()[1]
vim.lsp.buf_attach_client(%d, client.id)
`, buf)

	s.MustExecLua(t, luaCmd)

	// Wait for LSP server to process the file
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *NvimIntegrationTestRunner) MustOpenFileWithLock(t *testing.T, path protocol.DocumentURI) (nvim.Buffer, func(), time.Time) {
	t.Helper()
	buf, cleanup, err := s.OpenFileWithLock(t, path)
	require.NoError(t, err, "failed to open file: %v", err)
	return buf, cleanup, time.Now()
}

func (s *NvimIntegrationTestRunner) OpenFileWithLock(t *testing.T, path protocol.DocumentURI) (nvim.Buffer, func(), error) {
	t.Helper()
	s.mu.Lock()
	cleanup := func() {
		s.mu.Unlock()
	}
	// If there's a file already open, close it first
	if s.currentBuffer != nil {
		s.t.Logf("Closing previously open file: %s", s.currentBuffer.uri)
		if err := s.nvimInstance.Command("bd!"); err != nil {
			cleanup()
			return 0, nil, errors.Errorf("failed to close previous buffer: %w", err)
		}
		s.currentBuffer = nil
	}

	pathStr := strings.TrimPrefix(string(path.Path()), "file://")
	// s.t.Logf("Opening file: %s", pathStr)

	// Force close any other buffers that might be open
	if err := s.nvimInstance.Command("%bd!"); err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to close all buffers: %w", err)
	}

	if err := s.nvimInstance.Command("edit " + pathStr); err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to open file: %w", err)
	}

	buffer, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to get current buffer: %w", err)
	}

	// Set filetype to Go for .go files
	if strings.HasSuffix(pathStr, ".go") {
		if err = s.nvimInstance.SetBufferOption(buffer, "filetype", "go"); err != nil {
			cleanup()
			return 0, nil, errors.Errorf("failed to set filetype: %w", err)
		}
	}

	// Track the current buffer
	s.currentBuffer = &struct {
		uri    protocol.DocumentURI
		buffer nvim.Buffer
	}{
		uri:    path,
		buffer: buffer,
	}

	// Verify we're in the right buffer
	currentBuf, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to verify current buffer: %w", err)
	}
	if currentBuf != buffer {
		cleanup()
		return 0, nil, errors.Errorf("buffer mismatch after opening file: expected %v, got %v", buffer, currentBuf)
	}

	// attach LSP
	if err := s.attachLSP(t, buffer); err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to attach LSP: %w", err)
	}

	s.t.Logf("Successfully opened file %s with buffer %v", pathStr, buffer)
	return buffer, cleanup, nil
}

func (s *NvimIntegrationTestRunner) Command(cmd string) error {
	return s.nvimInstance.Command(cmd)
}

func (s *NvimIntegrationTestRunner) TmpFilePathOf(path string) protocol.DocumentURI {
	return protocol.URIFromPath(filepath.Join(s.TmpDir, path))
}

func RequireOneRPCResponse[T any](t *testing.T, s *NvimIntegrationTestRunner, method string, fileOpenTime time.Time) (*T, *jrpc2.Error) {
	t.Helper()
	msgs := s.rpcTracker.MessagesSinceLike(fileOpenTime, func(msg protocol.RPCMessage) bool {
		return msg.Method == method && msg.Response != nil
	})
	require.Len(t, msgs, 1, "expected 1 semantic tokens message, got %d", len(msgs))
	if msgs[0].Response.Error() != nil {
		return nil, msgs[0].Response.Error()
	}
	var result T
	require.NoError(t, msgs[0].Response.UnmarshalResult(&result), "failed to unmarshal semantic tokens response")
	return &result, nil
}

func (s *NvimIntegrationTestRunner) MustNvimCommand(t *testing.T, cmd string) {
	t.Helper()
	err := s.nvimInstance.Command(cmd)
	require.NoError(t, err, "failed to run command: %s", cmd)
}

func (s *NvimIntegrationTestRunner) MustExecLua(t *testing.T, cmd string, args ...any) any {
	t.Helper()
	var result any
	err := s.nvimInstance.ExecLua(cmd, &result, args...)
	require.NoError(t, err, "failed to run command: %s", cmd)
	return result
}

func (s *NvimIntegrationTestRunner) MustCall(t *testing.T, name string, args ...any) any {
	t.Helper()
	var result any
	err := s.nvimInstance.Call(name, &result, args...)
	require.NoError(t, err, "failed to call command: %s", name)
	return result
}

// Hover gets hover information at the current cursor position
func (s *NvimIntegrationTestRunner) Hover(t *testing.T, ctx context.Context, request *protocol.HoverParams) (*protocol.Hover, error) {
	t.Helper()
	_, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, request.TextDocument.URI)
	defer cleanup()

	t.Logf("🎯 Moving cursor to position: %v", request.Position)
	// Move cursor to the specified position
	win, err := s.nvimInstance.CurrentWindow()
	require.NoError(t, err, "failed to get current window: %w", err)

	err = s.nvimInstance.SetWindowCursor(win, [2]int{int(request.Position.Line) + 1, int(request.Position.Character)})
	require.NoError(t, err, "failed to set cursor position: %w", err)

	// Trigger hover directly using LSP
	s.MustNvimCommand(t, "lua vim.lsp.buf.hover()")

	time.Sleep(1 * time.Second)

	res, err := RequireOneRPCResponse[protocol.Hover](t, s, "textDocument/hover", fileOpenTime)
	require.Nil(t, err, "expected no error in hover response")
	require.NotNil(t, res, "expected non-nil hover response")

	s.t.Logf("Hover response: %v", res)

	return res, nil

}

func (s *NvimIntegrationTestRunner) GetDiagnostics(t *testing.T, uri protocol.DocumentURI, severity protocol.DiagnosticSeverity) ([]protocol.Diagnostic, []protocol.RPCMessage) {
	t.Helper()

	buf, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, uri)
	defer cleanup()

	s.triggerDiagnosticRefresh(t, buf)

	msgs := s.rpcTracker.MessagesSinceLike(fileOpenTime, func(msg protocol.RPCMessage) bool {
		return msg.Method == "textDocument/diagnostic"
	})

	diags := s.loadNvimDiagnosticsFromBuffer(t, buf, severity)

	protocolDiags := make([]protocol.Diagnostic, 0, len(diags))
	for _, d := range diags {
		protocolDiags = append(protocolDiags, d.ToProtocolDiagnostic())
	}

	return protocolDiags, msgs
}

type NvimDiagnostic struct {
	Bufnr     int            `json:"bufnr"`
	Code      string         `json:"code"`
	Col       int            `json:"col"`
	EndCol    int            `json:"end_col"`
	EndLnum   int            `json:"end_lnum"`
	Lnum      int            `json:"lnum"`
	Message   string         `json:"message"`
	Namespace int            `json:"namespace"`
	Source    string         `json:"source"`
	UserData  map[string]any `json:"user_data"`
}

func (me *NvimDiagnostic) ToProtocolDiagnostic() protocol.Diagnostic {
	return protocol.Diagnostic{
		Message:  me.Message,
		Range:    protocol.Range{Start: protocol.Position{Line: uint32(me.Lnum), Character: uint32(me.Col)}, End: protocol.Position{Line: uint32(me.EndLnum), Character: uint32(me.EndCol)}},
		Severity: protocol.SeverityError,
		Source:   me.Source,
		Code:     me.Code,
		CodeDescription: &protocol.CodeDescription{
			Href: me.UserData["lsp"].(map[string]any)["codeDescription"].(map[string]any)["href"].(string),
		},
		RelatedInformation: nil,
		Tags:               nil,
		Data:               nil,
	}
}

func (s *NvimIntegrationTestRunner) triggerDiagnosticRefresh(t *testing.T, buf nvim.Buffer) {
	t.Helper()
	luaCmd := fmt.Sprintf(`
	local client = vim.lsp.get_active_clients()[1]
	if client then
		-- Request diagnostics from gopls
		client.request('textDocument/diagnostic', {
			textDocument = {
				uri = vim.uri_from_bufnr(%d)
			}
		})
	end
`, buf)
	_ = s.MustExecLua(t, luaCmd)
	time.Sleep(1 * time.Second)
}

func severityToLua(severity protocol.DiagnosticSeverity) string {
	switch severity {
	case protocol.SeverityError:
		return "ERROR"
	case protocol.SeverityWarning:
		return "WARN"
	case protocol.SeverityInformation:
		return "INFO"
	case protocol.SeverityHint:
		return "HINT"
	default:
		return "INFO"
	}
}

func (s *NvimIntegrationTestRunner) loadNvimDiagnosticsFromBuffer(t *testing.T, buf nvim.Buffer, severity protocol.DiagnosticSeverity) []NvimDiagnostic {
	t.Helper()
	// out := s.MustCall(t, "vim.diagnostic.get", buf, map[string]string{"severity": "vim.diagnostic.severity.WARN"})
	l := s.MustExecLua(t, `
		local severity = vim.diagnostic.severity.`+severityToLua(severity)+`
		return vim.diagnostic.get(`+fmt.Sprintf("%d", buf)+`, {severity = severity})
	`)

	require.NotNil(t, l, "expected non-nil diagnostic response")

	by, err := json.Marshal(l)
	require.NoError(t, err, "failed to marshal diagnostic response")

	var diags []NvimDiagnostic
	require.NoError(t, json.Unmarshal(by, &diags), "failed to unmarshal diagnostic response")

	return diags
}

// GetSemanticTokensFull returns semantic tokens for the entire document
func (s *NvimIntegrationTestRunner) GetSemanticTokensFull(t *testing.T, ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	t.Helper()
	_, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, params.TextDocument.URI)
	defer cleanup()

	s.MustNvimCommand(t, `lua vim.lsp.buf_request(0, 'textDocument/semanticTokens/full', { textDocument = vim.lsp.util.make_text_document_params() })`)

	time.Sleep(1 * time.Second)

	res, err := RequireOneRPCResponse[protocol.SemanticTokens](t, s, "textDocument/semanticTokens/full", fileOpenTime)
	require.Nil(t, err, "expected no error in semantic tokens response")
	require.NotNil(t, res, "expected non-nil semantic tokens response")

	return res, nil
}

// GetSemanticTokensRange returns semantic tokens for a specific range
func (s *NvimIntegrationTestRunner) GetSemanticTokensRange(t *testing.T, ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	t.Helper()
	_, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, params.TextDocument.URI)
	defer cleanup()

	// Set the visual selection to the requested range
	startLine := int(params.Range.Start.Line) + 1
	startChar := int(params.Range.Start.Character)
	endLine := int(params.Range.End.Line) + 1
	endChar := int(params.Range.End.Character)

	// Move to start position and enter visual mode
	s.MustNvimCommand(t, fmt.Sprintf("normal! %dG%d|v%dG%d|", startLine, startChar, endLine, endChar))

	// Request semantic tokens directly through LSP
	s.MustNvimCommand(t, `lua vim.lsp.buf_request(0, 'textDocument/semanticTokens/range', { textDocument = vim.lsp.util.make_text_document_params(), range = vim.lsp.util.make_given_range_params().range })`)

	time.Sleep(1 * time.Second)

	res, err := RequireOneRPCResponse[protocol.SemanticTokens](t, s, "textDocument/semanticTokens/range", fileOpenTime)
	require.Nil(t, err, "expected no error in semantic tokens response")
	require.NotNil(t, res, "expected non-nil semantic tokens response")

	return res, nil
}

// SaveAndQuit saves the current buffer and quits Neovim
func (s *NvimIntegrationTestRunner) SaveAndQuit() error {
	if s.currentBuffer != nil {
		outFile := filepath.Join(s.TmpDir, "nvim.out")
		s.t.Logf("Saving current buffer %v to %s", s.currentBuffer.buffer, outFile)

		err := s.nvimInstance.Command("write! " + outFile)
		if err != nil {
			return errors.Errorf("failed to write file: %w", err)
		}
	}

	s.t.Log("Quitting Neovim...")
	err := s.nvimInstance.Command("qa!")
	if err != nil && !strings.Contains(err.Error(), "msgpack/rpc: session closed") {
		return errors.Errorf("failed to quit neovim: %w", err)
	}

	return nil
}

// SaveAndQuitWithOutput saves the current buffer, quits Neovim, and returns the buffer content
func (s *NvimIntegrationTestRunner) SaveAndQuitWithOutput() (string, error) {
	err := s.SaveAndQuit()
	if err != nil {
		return "", errors.Errorf("failed to save and quit: %w", err)
	}

	outFile := filepath.Join(s.TmpDir, "nvim.out")
	content, err := os.ReadFile(outFile)
	if err != nil {
		return "", errors.Errorf("failed to read output file: %w", err)
	}

	return string(content), nil
}

// SaveFile saves the current buffer
func (s *NvimIntegrationTestRunner) SaveFile(buffer nvim.Buffer) error {
	return s.nvimInstance.Command("w")
}

// GetDocumentText returns the current text content of a document
func (s *NvimIntegrationTestRunner) GetDocumentText(t *testing.T, uri protocol.DocumentURI) (string, error) {
	var text string
	buffer, cleanup, _ := s.MustOpenFileWithLock(t, uri)
	defer cleanup()

	lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
	if err != nil {
		return "", errors.Errorf("getting buffer lines: %w", err)
	}
	text = strings.Join(bytesSliceToStringSlice(lines), "\n")
	return text, nil
}

// ApplyEdit applies changes to a document with optional save
func (s *NvimIntegrationTestRunner) ApplyEdit(t *testing.T, uri protocol.DocumentURI, newContent string, save bool) error {
	buffer, cleanup, _ := s.MustOpenFileWithLock(t, uri)
	defer cleanup()

	// Delete all content and insert new content
	if err := s.nvimInstance.Command("normal! ggdG"); err != nil {
		return errors.Errorf("deleting content: %w", err)
	}

	// Insert new content
	if err := s.nvimInstance.Command(fmt.Sprintf("normal! i%s", newContent)); err != nil {
		return errors.Errorf("inserting content: %w", err)
	}

	if save {
		return s.SaveFile(buffer)
	}

	return nil
}

// GetFormattedDocument returns the formatted content of a document
func (s *NvimIntegrationTestRunner) GetFormattedDocument(t *testing.T, ctx context.Context, uri protocol.DocumentURI) (string, error) {
	panic("not implemented")
}

// GetDefinition returns the definition locations for a symbol
func (s *NvimIntegrationTestRunner) GetDefinition(t *testing.T, ctx context.Context, params *protocol.DefinitionParams) ([]*protocol.Location, error) {
	panic("not implemented")
}

// GetReferences returns all references to a symbol
func (s *NvimIntegrationTestRunner) GetReferences(t *testing.T, ctx context.Context, params *protocol.ReferenceParams) ([]*protocol.Location, error) {
	panic("not implemented")
}

// GetDocumentSymbols returns all symbols in a document
func (s *NvimIntegrationTestRunner) GetDocumentSymbols(t *testing.T, ctx context.Context, params *protocol.DocumentSymbolParams) ([]*protocol.DocumentSymbol, error) {
	panic("not implemented")
}

// ApplyRename applies a rename operation to a symbol
func (s *NvimIntegrationTestRunner) ApplyRename(t *testing.T, ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	panic("not implemented")
}

// GetCodeActions returns available code actions for a given range
func (s *NvimIntegrationTestRunner) GetCodeActions(t *testing.T, ctx context.Context, params *protocol.CodeActionParams) ([]*protocol.CodeAction, error) {
	panic("not implemented")
}

// GetCompletion returns completion items at the current position
func (s *NvimIntegrationTestRunner) GetCompletion(t *testing.T, ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	panic("not implemented")
}

// GetSignatureHelp returns signature help for the current position
func (s *NvimIntegrationTestRunner) GetSignatureHelp(t *testing.T, ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	panic("not implemented")
}
