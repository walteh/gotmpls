package nvim

import (
	"archive/tar"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/rs/zerolog"
	nvimlspconfig "github.com/walteh/go-tmpl-typer/gen/git-repo-tarballs/nvim-lspconfig"
	"github.com/walteh/go-tmpl-typer/pkg/archive"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/integration"

	// "github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
	"gitlab.com/tozd/go/errors"
)

// NvimIntegrationTestRunner contains all the necessary components for a neovim LSP test
type NvimIntegrationTestRunner struct {
	nvimInstance   *nvim.Nvim
	serverInstance *protocol.ServerInstance
	TmpDir         string
	t              *testing.T
	currentBuffer  *struct {
		uri    protocol.DocumentURI
		buffer nvim.Buffer
	}
	mu sync.Mutex // Protects file operations
}

var _ integration.IntegrationTestRunner = &NvimIntegrationTestRunner{}

func NewNvimIntegrationTestRunner(t *testing.T, files map[string]string, si *protocol.ServerInstance, config NeovimConfig) (*NvimIntegrationTestRunner, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	tmpDir, err := os.MkdirTemp("", "nvim-test-*")
	if err != nil {
		cancel()
		return nil, errors.Errorf("failed to create temp dir: %v", err)
	}

	setup := &NvimIntegrationTestRunner{
		t:              t,
		serverInstance: si,
		TmpDir:         tmpDir,
	}

	// Create a Unix domain socket for LSP communication in the temp directory
	socketPath := filepath.Join(tmpDir, "lsp-test.sock")

	// Create cleanup function that will be called when test is done
	t.Cleanup(func() {
		cancel()
		if setup.nvimInstance != nil {
			if err := setup.nvimInstance.Close(); err != nil {
				t.Logf("failed to close neovim: %v", err)
			}
		}

		defer func() {
			os.RemoveAll(tmpDir)
			os.Remove(socketPath)
		}()

		// Check the Neovim log
		nvimLogPath := filepath.Join(tmpDir, "nvim.log")
		if nvimLog, err := os.ReadFile(nvimLogPath); err == nil {
			debugNvimLogLines := os.Getenv("DEBUG_NVIM_LOG_LINES")
			var inter int
			if debugNvimLogLines == "" {
				t.Logf("DEBUG_NVIM_LOG_LINES not set, skipping log")
				return
			} else if debugNvimLogLines == "all" {
				t.Logf("DEBUG_NVIM_LOG_LINES set to all, WARNING: this will print a lot of logs")
				inter = math.MaxInt
			} else {
				inter, err = strconv.Atoi(debugNvimLogLines)
				if err != nil {
					t.Logf("could not parse DEBUG_NVIM_LOG_LINES (%s) as a number, using default of 50", debugNvimLogLines)
					inter = 50
				}
			}
			lastLines := lastN(strings.Split(string(nvimLog), "\n"), inter)
			lastWord := "last"
			if inter == math.MaxInt {
				lastWord = "all"
			}
			t.Logf("nvim log (%s %d lines):\n%s", lastWord, len(lastLines), strings.Join(lastLines, "\n"))
		}

	})

	// Listen on the Unix socket
	t.Log("Starting socket listener...")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, errors.Errorf("failed to create listener: %v", err)
	}

	// Start the LSP server in a goroutine
	serverStarted := make(chan struct{})
	serverError := make(chan error, 1)
	go func() {
		defer close(serverStarted)
		defer close(serverError)
		defer listener.Close()

		// Signal that we're ready to accept connections
		serverStarted <- struct{}{}

		// Accept a connection
		t.Log("Waiting for connection...")
		conn, err := listener.Accept()
		if err != nil {
			serverError <- errors.Errorf("failed to accept connection: %v", err)
			return
		}
		defer conn.Close()

		t.Log("Starting server...")

		// server := lsp.NewServer(ctx)
		// opts := jrpc2.ServerOptions{}
		si.ServerOpts.RPCLog = protocol.NewTestLogger(t, map[string]string{
			tmpDir: "/[TEMP_DIR]",
		})
		// si.AddBackgroundCmdFlag(fmt.Sprintf("-logfile=%s/gopls.log", tmpDir))
		zerolog.Ctx(ctx).Info().Msg("Starting server...")

		if err := si.StartAndWait(conn, conn); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				serverError <- errors.Errorf("LSP server error: %v", err)
			}
			t.Log("Server stopped with:", err)
		}
	}()

	// Wait for the server to be ready or error
	select {
	case err := <-serverError:
		return nil, errors.Errorf("LSP server failed to start: %v", err)
	case <-serverStarted:
		t.Log("LSP server ready")
	case <-time.After(5 * time.Second):
		return nil, errors.Errorf("timeout waiting for LSP server to start")
	}

	configPath, err := setupNeovimConfig(t, tmpDir, socketPath, config)
	if err != nil {
		return nil, errors.Errorf("failed to setup LSP config: %v", err)
	}
	setup.TmpDir = tmpDir

	// Create test files
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {

			return nil, errors.Errorf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {

			return nil, errors.Errorf("failed to write file %s: %v", fullPath, err)
		}
	}

	// Start Neovim with the context
	t.Log("Creating neovim instance...")
	cmd := os.Getenv("GO_TMPL_TYPER_NEOVIM_BIN")
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("nvim")
		if err != nil {

			return nil, errors.Errorf("nvim not installed: %v", err)
		}
	}
	t.Logf("Using nvim command: %s", cmd)

	nvimArgs := []string{
		"--clean",
		"-n",
		"--embed",
		"--headless",
		"--noplugin",
		"-u", configPath,
		"-V20" + filepath.Join(tmpDir, "nvim.log"),
	}
	t.Logf("Starting neovim with args: %v", nvimArgs)

	nvimInstance, err := nvim.NewChildProcess(
		nvim.ChildProcessCommand(cmd),
		nvim.ChildProcessArgs(nvimArgs...),
		nvim.ChildProcessContext(ctx),
		nvim.ChildProcessLogf(t.Logf),
	)
	if err != nil {
		return nil, errors.Errorf("failed to create neovim instance: %v", err)
	}
	setup.nvimInstance = nvimInstance

	// Explicitly source our config
	t.Log("Sourcing LSP config...")
	if err := nvimInstance.Command("source " + configPath); err != nil {

		return nil, errors.Errorf("failed to source config: %v", err)
	}

	return setup, nil
}

// Helper method to wait for LSP to initialize
func (s *NvimIntegrationTestRunner) WaitForLSP() error {
	waitForLSP := func() bool {
		var hasClients bool
		err := s.nvimInstance.Eval(`luaeval('vim.lsp.get_active_clients() ~= nil and #vim.lsp.get_active_clients() > 0')`, &hasClients)
		if err != nil {
			s.t.Logf("Error checking LSP clients: %v", err)
			return false
		}

		s.t.Logf("LSP clients count: %v", hasClients)

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

func setupNeovimConfig(t *testing.T, tmpDir string, socketPath string, config NeovimConfig) (string, error) {
	lspConfigDir := filepath.Join(tmpDir, "nvim-lspconfig")
	t.Log("Extracting nvim-lspconfig files...")
	err := archive.ExtractTarGzWithOptions(nvimlspconfig.Data, lspConfigDir, archive.ExtractOptions{
		StripComponents: 1, // Remove the "nvim-lspconfig-master" prefix
		Filter: func(header *tar.Header) bool {
			return header.Name != "" // Skip empty paths
		},
	})
	if err != nil {
		return "", errors.Errorf("failed to extract nvim-lspconfig: %w", err)
	}

	// list the files in the nvim-lspconfig dir
	files, err := os.ReadDir(lspConfigDir)
	if err != nil {
		return "", errors.Errorf("failed to read nvim-lspconfig dir: %w", err)
	}
	t.Logf("Files in nvim-lspconfig dir: %v", files)

	vimConfig := fmt.Sprintf(`
set verbose=20
let s:lspconfig_path = '%[1]s'
let &runtimepath = s:lspconfig_path . ',' . $VIMRUNTIME . ',' . s:lspconfig_path . '/after'
set packpath=%[1]s

" Set up filetype detection
autocmd! BufEnter *.tmpl setlocal filetype=go-template

" Load lspconfig
runtime! plugin/lspconfig.lua

lua <<EOF

-- Enable debug logging
vim.lsp.set_log_level("debug")

local lspconfig = require 'lspconfig'
local configs = require 'lspconfig.configs'
local util = require 'lspconfig.util'
local async = require 'lspconfig.async'
-- Print loaded configs for debugging
print("Available LSP configs:", vim.inspect(configs))

-- Configure capabilities
local capabilities = vim.lsp.protocol.make_client_capabilities()
capabilities.textDocument.hover = {
    dynamicRegistration = true,
    contentFormat = { "plaintext", "markdown" }
}
-- Disable semantic tokens
-- capabilities.textDocument.semanticTokens = nil

-- Use an on_attach function to only map the following keys
local on_attach = function(client, bufnr)
    print("LSP client attached:", vim.inspect(client))
    print("Buffer:", bufnr)
    print("Client capabilities:", vim.inspect(client.server_capabilities))
    
    -- Disable semantic tokens
    -- client.server_capabilities.semanticTokensProvider = nil

    -- Set buffer options
    vim.api.nvim_buf_set_option(bufnr, 'omnifunc', 'v:lua.vim.lsp.omnifunc')
end

print("start default config")
%[2]s
print("end default config")

print("start default setup")
%[3]s
print("end default setup")

print("LSP setup complete")
EOF`, lspConfigDir, config.DefaultConfig(socketPath), config.DefaultSetup())

	fmt.Printf("vimConfig: %s", vimConfig)

	configPath := filepath.Join(tmpDir, "config.vim")
	if err := os.WriteFile(configPath, []byte(vimConfig), 0644); err != nil {
		return "", errors.Errorf("failed to write config: %w", err)
	}

	return configPath, nil
}

// Helper methods for neovimTestSetup
func (s *NvimIntegrationTestRunner) OpenFile(path protocol.DocumentURI) (nvim.Buffer, error) {
	// If there's a file already open, close it first
	if s.currentBuffer != nil {
		s.t.Logf("Closing previously open file: %s", s.currentBuffer.uri)
		if err := s.nvimInstance.Command("bd!"); err != nil {
			return 0, errors.Errorf("failed to close previous buffer: %w", err)
		}
		s.currentBuffer = nil
	}

	pathStr := strings.TrimPrefix(string(path.Path()), "file://")
	s.t.Logf("Opening file: %s", pathStr)

	// Force close any other buffers that might be open
	if err := s.nvimInstance.Command("%bd!"); err != nil {
		return 0, errors.Errorf("failed to close all buffers: %w", err)
	}

	if err := s.nvimInstance.Command("edit " + pathStr); err != nil {
		return 0, errors.Errorf("failed to open file: %w", err)
	}

	buffer, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		return 0, errors.Errorf("failed to get current buffer: %w", err)
	}

	// Set filetype to Go for .go files
	if strings.HasSuffix(pathStr, ".go") {
		if err = s.nvimInstance.SetBufferOption(buffer, "filetype", "go"); err != nil {
			return 0, errors.Errorf("failed to set filetype: %w", err)
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
		return 0, errors.Errorf("failed to verify current buffer: %w", err)
	}
	if currentBuf != buffer {
		return 0, errors.Errorf("buffer mismatch after opening file: expected %v, got %v", buffer, currentBuf)
	}

	s.t.Logf("Successfully opened file %s with buffer %v", pathStr, buffer)
	return buffer, nil
}

func (s *NvimIntegrationTestRunner) attachLSP(buf nvim.Buffer) error {
	s.t.Log("Waiting for LSP to initialize...")
	if err := s.WaitForLSP(); err != nil {
		return errors.Errorf("failed to wait for LSP: %w", err)
	}

	// Attach LSP client using Lua - this will automatically send didOpen
	if err := s.nvimInstance.Eval(`luaeval('vim.lsp.buf_attach_client(0, 1)')`, nil); err != nil {
		return errors.Errorf("failed to attach LSP client: %w", err)
	}

	// Wait for LSP server to process the file
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *NvimIntegrationTestRunner) Command(cmd string) error {
	return s.nvimInstance.Command(cmd)
}

func (s *NvimIntegrationTestRunner) TmpFilePathOf(path string) protocol.DocumentURI {

	return protocol.URIFromPath(filepath.Join(s.TmpDir, path))
}

// setupLSPInterceptors sets up handlers to intercept LSP messages
func (s *NvimIntegrationTestRunner) setupLSPInterceptors(t *testing.T) error {
	interceptCmd := `
		-- Store original handlers
		if not _G.original_handlers then
			_G.original_handlers = {}
			_G.intercepted_messages = {}
		end

		-- Helper to intercept messages
		local function intercept_message(method, result)
			if not _G.intercepted_messages[method] then
				_G.intercepted_messages[method] = {}
			end
			table.insert(_G.intercepted_messages[method], vim.json.encode(result))
		end

		-- Intercept specific message types
		local methods_to_intercept = {
			"textDocument/hover",
			"textDocument/publishDiagnostics",
			"textDocument/diagnostic",
				-- Add more methods as needed
			}

		for _, method in ipairs(methods_to_intercept) do
			if not _G.original_handlers[method] then
				_G.original_handlers[method] = vim.lsp.handlers[method]
				vim.lsp.handlers[method] = function(err, result, ctx, config)
					-- Store the intercepted message
					if result then
						print("intercepted message", method, result)
						intercept_message(method, result)
					end
					-- Call original handler
					if _G.original_handlers[method] then
						_G.original_handlers[method](err, result, ctx, config)
					end
				end
			end
		end
	`
	return s.nvimInstance.ExecLua(interceptCmd, nil)
}

// getInterceptedMessages retrieves messages of a specific type
func (s *NvimIntegrationTestRunner) getInterceptedMessages(method string) ([]string, error) {
	var messages []string
	getMessagesCmd := fmt.Sprintf(`
		if _G.intercepted_messages and _G.intercepted_messages["%s"] then
			return _G.intercepted_messages["%s"]
		end
		return {}
	`, method, method)

	err := s.nvimInstance.ExecLua(getMessagesCmd, &messages)
	if err != nil {
		return nil, errors.Errorf("failed to get intercepted messages: %w", err)
	}
	return messages, nil
}

func getUnmarshaledInterceptedMessages[T any](s *NvimIntegrationTestRunner, method string) ([]T, error) {
	var res []T
	messages, err := s.getInterceptedMessages(method)
	if err != nil {
		return nil, errors.Errorf("failed to get intercepted messages: %w", err)
	}
	for _, msg := range messages {
		var m T
		if err := json.Unmarshal([]byte(msg), &m); err != nil {
			return nil, errors.Errorf("failed to unmarshal message: %w", err)
		}
		res = append(res, m)
	}
	return res, nil
}

func getUnmarshaledIntercepedMessage[T any](s *NvimIntegrationTestRunner, method string) (*T, error) {
	messages, err := getUnmarshaledInterceptedMessages[T](s, method)
	if err != nil {
		return nil, errors.Errorf("failed to get intercepted messages: %w", err)
	}
	return &messages[len(messages)-1], nil
}

func getUnmarshaledIntercepedMessageWithTimeout[T any](s *NvimIntegrationTestRunner, method string, timeout time.Duration) (*T, error) {
	messages, err := getUnmarshaledIntercepedMessagesWithTimeout[T](s, method, timeout)
	if err != nil {
		return new(T), errors.Errorf("failed to get intercepted messages: %w", err)
	}
	return &messages[len(messages)-1], nil
}

func getUnmarshaledIntercepedMessagesWithTimeout[T any](s *NvimIntegrationTestRunner, method string, timeout time.Duration) ([]T, error) {
	start := time.Now()
	for time.Since(start) < timeout {
		messages, err := getUnmarshaledInterceptedMessages[T](s, method)
		if err != nil {
			return nil, errors.Errorf("failed to get intercepted messages: %w", err)
		}
		if len(messages) > 0 {
			return messages, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, errors.Errorf("timeout waiting for %s message", method)
}

// clearInterceptedMessages clears stored messages
func (s *NvimIntegrationTestRunner) clearInterceptedMessages() error {
	clearCmd := `_G.intercepted_messages = {}`
	return s.nvimInstance.ExecLua(clearCmd, nil)
}

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

func withFileResult[T any](s *NvimIntegrationTestRunner, uri protocol.DocumentURI, operation func(buffer nvim.Buffer) (T, error)) (T, error) {
	var res T
	err := s.withFile(uri, func(buffer nvim.Buffer) error {
		var err error
		res, err = operation(buffer)
		return err
	})
	return res, err
}

func withFileResultsMethodTimeout[T any](s *NvimIntegrationTestRunner, uri protocol.DocumentURI, method string, timeout time.Duration, operation func(buffer nvim.Buffer) error) ([]T, error) {
	return withFileResult(s, uri, func(buffer nvim.Buffer) ([]T, error) {
		if err := operation(buffer); err != nil {
			return nil, err
		}
		return getUnmarshaledIntercepedMessagesWithTimeout[T](s, method, timeout)
	})
}

func withFileResultMethodTimeout[T any](s *NvimIntegrationTestRunner, uri protocol.DocumentURI, method string, timeout time.Duration, operation func(buffer nvim.Buffer) error) (*T, error) {
	return withFileResult(s, uri, func(buffer nvim.Buffer) (*T, error) {
		if err := operation(buffer); err != nil {
			return nil, err
		}
		return getUnmarshaledIntercepedMessageWithTimeout[T](s, method, timeout)
	})
}

// withFile ensures atomic and consistent file handling for LSP operations
func (s *NvimIntegrationTestRunner) withFile(uri protocol.DocumentURI, operation func(buffer nvim.Buffer) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get buffer for URI
	buffers, err := s.nvimInstance.Buffers()
	if err != nil {
		return errors.Errorf("failed to get buffers: %w", err)
	}

	var buffer nvim.Buffer
	for _, b := range buffers {
		name, err := s.nvimInstance.BufferName(b)
		if err != nil {
			continue
		}
		if strings.HasSuffix(name, string(uri)) {
			buffer = b
			break
		}
	}

	// If buffer not found, try to open the file
	if buffer == 0 {
		buffer, err = s.OpenFile(uri)
		if err != nil {
			return errors.Errorf("failed to open file: %w", err)
		}

		// Attach LSP
		if err := s.attachLSP(buffer); err != nil {
			return errors.Errorf("failed to attach LSP: %w", err)
		}

		// Set up interceptors if not already set
		if err := s.setupLSPInterceptors(s.t); err != nil {
			return errors.Errorf("failed to setup LSP interceptors: %w", err)
		}

		// Clear any previous messages
		if err := s.clearInterceptedMessages(); err != nil {
			return errors.Errorf("failed to clear intercepted messages: %w", err)
		}
	}

	// Execute the function with the buffer
	return operation(buffer)
}

func (s *NvimIntegrationTestRunner) Hover(t *testing.T, ctx context.Context, request *protocol.HoverParams) (*protocol.Hover, error) {
	return withFileResultMethodTimeout[protocol.Hover](s, request.TextDocument.URI, "textDocument/hover", time.Second, func(buffer nvim.Buffer) error {
		t.Logf("🎯 Moving cursor to position: %v", request.Position)
		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(request.Position.Line) + 1, int(request.Position.Character)})
		if err != nil {
			return errors.Errorf("failed to set cursor position: %w", err)
		}

		// Trigger hover directly using LSP
		err = s.nvimInstance.Command("lua vim.lsp.buf.hover()")
		if err != nil {
			return errors.Errorf("failed to trigger hover: %w", err)
		}
		return nil
	})
}

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

// GetDiagnostics returns the current diagnostics for a given file URI
func (s *NvimIntegrationTestRunner) GetDiagnostics(t *testing.T, uri protocol.DocumentURI, timeout time.Duration) (*protocol.FullDocumentDiagnosticReport, error) {
	return withFileResultMethodTimeout[protocol.FullDocumentDiagnosticReport](s, uri, "textDocument/diagnostic", timeout, func(buffer nvim.Buffer) error {
		t.Log("🔄 Getting diagnostics (timeout:", timeout, ")")

		// Get diagnostics directly from gopls
		luaCmd := `
			local bufnr = vim.api.nvim_get_current_buf()
			local client = vim.lsp.get_active_clients()[1]
			if client then
				-- Request diagnostics from gopls
				client.request('textDocument/diagnostic', {
					textDocument = {
						uri = vim.uri_from_bufnr(bufnr)
					}
				})
			end
		`
		if err := s.nvimInstance.ExecLua(luaCmd, nil); err != nil {
			return errors.Errorf("failed to get diagnostics: %w", err)
		}

		return nil
	})
}

func (s *NvimIntegrationTestRunner) SaveFile(buffer nvim.Buffer) error {
	// Save the file
	err := s.nvimInstance.Command("w")
	if err != nil {
		return errors.Errorf("failed to save file: %w", err)
	}

	// Get current buffer text
	lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
	if err != nil {
		return errors.Errorf("failed to get buffer lines: %w", err)
	}
	text := strings.Join(bytesSliceToStringSlice(lines), "\n")

	// Send file contents to LSP server using Lua
	bufPath, err := s.nvimInstance.BufferName(buffer)
	if err != nil {
		return errors.Errorf("failed to get buffer name: %w", err)
	}

	s.t.Logf("Sending didChange notification for %s with text: %s", bufPath, text)

	// Notify LSP server about the change
	notifyCmd := fmt.Sprintf(`luaeval('vim.lsp.buf_notify(0, "textDocument/didChange", {
		textDocument = {
			uri = "file://%s",
			version = 2
		},
		contentChanges = {
			{
				text = [[%s]]
			}
		}
	})')`, bufPath, text)

	err = s.nvimInstance.Eval(notifyCmd, nil)
	if err != nil {
		return errors.Errorf("failed to notify LSP: %w", err)
	}

	// Also send a didSave notification
	saveCmd := fmt.Sprintf(`luaeval('vim.lsp.buf_notify(0, "textDocument/didSave", {
		textDocument = {
			uri = "file://%s"
		},
		text = [[%s]]
	})')`, bufPath, text)

	err = s.nvimInstance.Eval(saveCmd, nil)
	if err != nil {
		return errors.Errorf("failed to notify LSP of save: %w", err)
	}

	return nil
}

// GetDocumentText returns the current text content of a document
func (s *NvimIntegrationTestRunner) GetDocumentText(t *testing.T, uri protocol.DocumentURI) (string, error) {
	var text string
	err := s.withFile(uri, func(buffer nvim.Buffer) error {
		lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
		if err != nil {
			return errors.Errorf("getting buffer lines: %w", err)
		}
		text = strings.Join(bytesSliceToStringSlice(lines), "\n")
		return nil
	})
	if err != nil {
		return "", err
	}
	return text, nil
}

// GetFormattedDocument returns the formatted content of a document
func (s *NvimIntegrationTestRunner) GetFormattedDocument(t *testing.T, ctx context.Context, uri protocol.DocumentURI) (string, error) {
	var formatted string
	err := s.withFile(uri, func(buffer nvim.Buffer) error {
		// Set up interceptors if not already set
		if err := s.setupLSPInterceptors(t); err != nil {
			return errors.Errorf("failed to set up LSP interceptors: %w", err)
		}

		// Clear any previous messages
		if err := s.clearInterceptedMessages(); err != nil {
			return errors.Errorf("failed to clear intercepted messages: %w", err)
		}

		// Format the document
		formatCmd := `
			if vim.lsp.buf.format then
				vim.lsp.buf.format({async = false})
			end
		`
		if err := s.nvimInstance.ExecLua(formatCmd, nil); err != nil {
			return errors.Errorf("formatting document: %w", err)
		}

		// Get the formatted text
		lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
		if err != nil {
			return errors.Errorf("getting buffer lines: %w", err)
		}
		formatted = strings.Join(bytesSliceToStringSlice(lines), "\n")
		return nil
	})
	if err != nil {
		return "", err
	}
	return formatted, nil
}

// GetDefinition returns the definition locations for a symbol
func (s *NvimIntegrationTestRunner) GetDefinition(t *testing.T, ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) ([]protocol.Location, error) {
		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return nil, errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Position.Line) + 1, int(params.Position.Character)})
		if err != nil {
			return nil, errors.Errorf("failed to set cursor position: %w", err)
		}

		// Trigger definition lookup
		err = s.nvimInstance.Command("lua vim.lsp.buf.definition()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger definition lookup: %w", err)
		}

		return getUnmarshaledIntercepedMessagesWithTimeout[protocol.Location](s, "textDocument/definition", time.Second)
	})
}

// ApplyEdit applies changes to a document with optional save
func (s *NvimIntegrationTestRunner) ApplyEdit(t *testing.T, uri protocol.DocumentURI, newContent string, save bool) error {
	return s.withFile(uri, func(buffer nvim.Buffer) error {
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

		// If not saving, just notify LSP about the change
		lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
		if err != nil {
			return errors.Errorf("getting buffer lines: %w", err)
		}
		text := strings.Join(bytesSliceToStringSlice(lines), "\n")

		bufPath, err := s.nvimInstance.BufferName(buffer)
		if err != nil {
			return errors.Errorf("getting buffer name: %w", err)
		}

		notifyCmd := fmt.Sprintf(`luaeval('vim.lsp.buf_notify(0, "textDocument/didChange", {
			textDocument = {
				uri = "file://%s",
				version = 2
			},
			contentChanges = {
				{
					text = [[%s]]
				}
			}
		})')`, bufPath, text)

		return s.nvimInstance.Eval(notifyCmd, nil)
	})
}

// GetReferences returns all references to a symbol
func (s *NvimIntegrationTestRunner) GetReferences(t *testing.T, ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) ([]protocol.Location, error) {
		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return nil, errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Position.Line) + 1, int(params.Position.Character)})
		if err != nil {
			return nil, errors.Errorf("failed to set cursor position: %w", err)
		}

		// Trigger references lookup
		err = s.nvimInstance.Command("lua vim.lsp.buf.references()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger references lookup: %w", err)
		}

		return getUnmarshaledIntercepedMessagesWithTimeout[protocol.Location](s, "textDocument/references", time.Second)
	})
}

// GetDocumentSymbols returns all symbols in a document
func (s *NvimIntegrationTestRunner) GetDocumentSymbols(t *testing.T, ctx context.Context, params *protocol.DocumentSymbolParams) ([]protocol.DocumentSymbol, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) ([]protocol.DocumentSymbol, error) {
		// Trigger document symbols request
		err := s.nvimInstance.Command("lua vim.lsp.buf.document_symbol()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger document symbols: %w", err)
		}

		return getUnmarshaledIntercepedMessagesWithTimeout[protocol.DocumentSymbol](s, "textDocument/documentSymbol", time.Second)
	})
}

// ApplyRename applies a rename operation to a symbol
func (s *NvimIntegrationTestRunner) ApplyRename(t *testing.T, ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	var edit *protocol.WorkspaceEdit
	err := s.withFile(params.TextDocument.URI, func(buffer nvim.Buffer) error {
		// Set up interceptors if not already set
		if err := s.setupLSPInterceptors(t); err != nil {
			return errors.Errorf("failed to set up LSP interceptors: %w", err)
		}

		// Clear any previous messages
		if err := s.clearInterceptedMessages(); err != nil {
			return errors.Errorf("failed to clear intercepted messages: %w", err)
		}

		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Position.Line) + 1, int(params.Position.Character)})
		if err != nil {
			return errors.Errorf("failed to set cursor position: %w", err)
		}

		// Set up the rename command with the new name
		renameCmd := fmt.Sprintf(`
			local params = vim.lsp.util.make_position_params()
			params.newName = "%s"
			vim.lsp.buf.rename(params.newName)
		`, params.NewName)

		err = s.nvimInstance.ExecLua(renameCmd, nil)
		if err != nil {
			return errors.Errorf("failed to trigger rename: %w", err)
		}

		// Wait for and get rename response
		start := time.Now()
		for time.Since(start) < time.Second {
			messages, err := s.getInterceptedMessages("textDocument/rename")
			if err != nil {
				return errors.Errorf("failed to get rename messages: %w", err)
			}

			if len(messages) > 0 {
				// Use the most recent message
				lastMessage := messages[len(messages)-1]
				t.Logf("Received rename message: %s", lastMessage)

				err = json.Unmarshal([]byte(lastMessage), &edit)
				if err != nil {
					return errors.Errorf("failed to unmarshal rename response: %w", err)
				}
				return nil
			}

			time.Sleep(50 * time.Millisecond)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return edit, nil
}

// GetCodeActions returns available code actions for a given range
func (s *NvimIntegrationTestRunner) GetCodeActions(t *testing.T, ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) ([]protocol.CodeAction, error) {

		// Move cursor to the start of the range
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return nil, errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Range.Start.Line) + 1, int(params.Range.Start.Character)})
		if err != nil {
			return nil, errors.Errorf("failed to set cursor position: %w", err)
		}

		// Set visual selection to the range if it's not empty
		if params.Range.Start != params.Range.End {
			visualCmd := fmt.Sprintf("normal! v%dG%d|", params.Range.End.Line+1, params.Range.End.Character+1)
			err = s.nvimInstance.Command(visualCmd)
			if err != nil {
				return nil, errors.Errorf("failed to set visual selection: %w", err)
			}
		}

		// Trigger code actions
		err = s.nvimInstance.Command("lua vim.lsp.buf.code_action()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger code actions: %w", err)
		}

		return getUnmarshaledIntercepedMessagesWithTimeout[protocol.CodeAction](s, "textDocument/codeAction", time.Second)
	})

}

// GetCompletion returns completion items at the current position
func (s *NvimIntegrationTestRunner) GetCompletion(t *testing.T, ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) (*protocol.CompletionList, error) {
		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return nil, errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Position.Line) + 1, int(params.Position.Character)})
		if err != nil {
			return nil, errors.Errorf("failed to set cursor position: %w", err)
		}

		// Trigger completion
		err = s.nvimInstance.Command("lua vim.lsp.buf.completion()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger completion: %w", err)
		}

		return getUnmarshaledIntercepedMessageWithTimeout[protocol.CompletionList](s, "textDocument/completion", time.Second)
	})
}

// GetSignatureHelp returns signature help for the current position
func (s *NvimIntegrationTestRunner) GetSignatureHelp(t *testing.T, ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return withFileResult(s, params.TextDocument.URI, func(buffer nvim.Buffer) (*protocol.SignatureHelp, error) {
		// Move cursor to the specified position
		win, err := s.nvimInstance.CurrentWindow()
		if err != nil {
			return nil, errors.Errorf("failed to get current window: %w", err)
		}

		err = s.nvimInstance.SetWindowCursor(win, [2]int{int(params.Position.Line) + 1, int(params.Position.Character)})
		if err != nil {
			return nil, errors.Errorf("failed to set cursor position: %w", err)
		}

		// Trigger signature help
		err = s.nvimInstance.Command("lua vim.lsp.buf.signature_help()")
		if err != nil {
			return nil, errors.Errorf("failed to trigger signature help: %w", err)
		}

		return getUnmarshaledIntercepedMessageWithTimeout[protocol.SignatureHelp](s, "textDocument/signatureHelp", time.Second)
	})
}

// withFileDiagnostics is used specifically for diagnostic operations
func (s *NvimIntegrationTestRunner) withFileDiagnostics(t *testing.T, uri protocol.DocumentURI, fn func() ([]protocol.Diagnostic, error)) ([]protocol.Diagnostic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get buffer for URI
	buffers, err := s.nvimInstance.Buffers()
	if err != nil {
		return nil, errors.Errorf("failed to get buffers: %w", err)
	}

	var buffer nvim.Buffer
	for _, b := range buffers {
		name, err := s.nvimInstance.BufferName(b)
		if err != nil {
			continue
		}
		if strings.HasSuffix(name, string(uri)) {
			buffer = b
			break
		}
	}

	if buffer == 0 {
		return nil, errors.Errorf("buffer not found for URI %s", uri)
	}

	// Execute the function with the buffer
	return fn()
}
