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

	"github.com/neovim/go-client/nvim"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

var _ integration.IntegrationTestRunner = &NvimIntegrationTestRunner{}

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

// Hover gets hover information at the current cursor position
func (s *NvimIntegrationTestRunner) Hover(t *testing.T, ctx context.Context, request *protocol.HoverParams) (*protocol.Hover, []protocol.RPCMessage) {
	t.Helper()
	buf, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, request.TextDocument.URI)
	defer cleanup()

	// Move cursor to the specified position
	win, err := s.nvimInstance.CurrentWindow()
	require.NoError(t, err, "failed to get current window: %w", err)

	t.Logf("🎯 Moving cursor to position [ %v ] in window [ %v ]", request.Position, win)

	// Move cursor to the desired position
	err = s.nvimInstance.SetWindowCursor(win, [2]int{int(request.Position.Line) + 1, int(request.Position.Character)})
	require.NoError(t, err, "failed to set cursor position: %w", err)

	// Log cursor position
	pos, err := s.nvimInstance.WindowCursor(win)
	require.NoError(t, err, "failed to get cursor position: %w", err)
	t.Logf("Cursor is at position: %v", pos)

	// i am not sure how to get a result frfom ths otherwise, its difficult to get the hover content directly from the
	// window that is supposed to be open. so this works for now.
	resp := s.MustExecLua(t, fmt.Sprintf(`-- Save the previous hover handler
local prev_handler = vim.lsp.handlers["textDocument/hover"]

-- Set the custom hover handler
vim.lsp.handlers["textDocument/hover"] = function(err, result, ctx, config)
  if err then
    print("Hover error:", vim.inspect(err))
    _G.last_hover_content = nil
    return
  end
  if not (result and result.contents) then
    print("No hover content available")
    _G.last_hover_content = nil
    return
  end

  -- Save the content to a global variable
  _G.last_hover_content = result
  -- Call the original handler
  prev_handler(err, result, ctx, config)
end

-- Trigger hover
vim.lsp.buf.hover()

-- Wait for hover response
local attempts = 20
while not _G.last_hover_content and attempts > 0 do
  vim.wait(100)
  attempts = attempts - 1
end

-- Reset hover handler %s
vim.lsp.handlers["textDocument/hover"] = prev_handler

-- Return the hover content
return _G.last_hover_content`, buf))

	rpcs := s.rpcTracker.MessagesSinceLike(fileOpenTime, func(msg protocol.RPCMessage) bool {
		return msg.Method == "textDocument/hover"
	})

	by, err := json.Marshal(resp)
	require.NoError(t, err, "failed to marshal hover response")

	var hover protocol.Hover
	require.NoError(t, json.Unmarshal(by, &hover), "failed to unmarshal hover response")

	return &hover, rpcs

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
func (s *NvimIntegrationTestRunner) GetSemanticTokensFull(t *testing.T, ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetSemanticTokensRange returns semantic tokens for a specific range
func (s *NvimIntegrationTestRunner) GetSemanticTokensRange(t *testing.T, ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, []protocol.RPCMessage) {
	panic("not implemented")
}

// SaveAndQuit saves the current buffer and quits Neovim
func (s *NvimIntegrationTestRunner) SaveAndQuit() error {
	if s.currentBuffer != nil {
		outFile := filepath.Join(s.TmpDir, "nvim.out")

		err := s.nvimInstance.Command("write! " + outFile)
		if err != nil {
			return errors.Errorf("failed to write file: %w", err)
		}
	}

	s.t.Log("Quitting Neovim...")
	err := s.nvimInstance.Command("qa!")
	if err != nil && !strings.Contains(err.Error(), "msgpack/rpc: session closed") && !strings.Contains(err.Error(), "signal: killed") {
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
func (s *NvimIntegrationTestRunner) ApplyEdit(t *testing.T, uri protocol.DocumentURI, newContent string, save bool) []protocol.RPCMessage {
	_, cleanup, fileOpenTime := s.MustOpenFileWithLock(t, uri)
	defer cleanup()

	// Delete all content and insert new content
	s.MustNvimCommand(t, "normal! ggdG")

	// Insert new content
	s.MustNvimCommand(t, fmt.Sprintf("normal! i%s", newContent))

	if save {
		s.MustNvimCommand(t, "w")
	}

	rpcs := s.rpcTracker.MessagesSinceLike(fileOpenTime, func(msg protocol.RPCMessage) bool {
		return msg.Method == "textDocument/didChange"
	})

	return rpcs
}

// GetFormattedDocument returns the formatted content of a document
func (s *NvimIntegrationTestRunner) GetFormattedDocument(t *testing.T, ctx context.Context, uri protocol.DocumentURI) (string, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetDefinition returns the definition locations for a symbol
func (s *NvimIntegrationTestRunner) GetDefinition(t *testing.T, ctx context.Context, params *protocol.DefinitionParams) ([]*protocol.Location, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetReferences returns all references to a symbol
func (s *NvimIntegrationTestRunner) GetReferences(t *testing.T, ctx context.Context, params *protocol.ReferenceParams) ([]*protocol.Location, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetDocumentSymbols returns all symbols in a document
func (s *NvimIntegrationTestRunner) GetDocumentSymbols(t *testing.T, ctx context.Context, params *protocol.DocumentSymbolParams) ([]*protocol.DocumentSymbol, []protocol.RPCMessage) {
	panic("not implemented")
}

// ApplyRename applies a rename operation to a symbol
func (s *NvimIntegrationTestRunner) ApplyRename(t *testing.T, ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetCodeActions returns available code actions for a given range
func (s *NvimIntegrationTestRunner) GetCodeActions(t *testing.T, ctx context.Context, params *protocol.CodeActionParams) ([]*protocol.CodeAction, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetCompletion returns completion items at the current position
func (s *NvimIntegrationTestRunner) GetCompletion(t *testing.T, ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, []protocol.RPCMessage) {
	panic("not implemented")
}

// GetSignatureHelp returns signature help for the current position
func (s *NvimIntegrationTestRunner) GetSignatureHelp(t *testing.T, ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, []protocol.RPCMessage) {
	panic("not implemented")
}
