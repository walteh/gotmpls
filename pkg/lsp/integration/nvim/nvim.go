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
	pathStr := strings.TrimPrefix(string(path.Path()), "file://")

	s.t.Logf("Opening file: %s", pathStr)

	err := s.nvimInstance.Command("edit " + pathStr)
	if err != nil {
		return 0, errors.Errorf("failed to open file: %w", err)
	}

	buffer, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		return 0, errors.Errorf("failed to get current buffer: %w", err)
	}

	// Set filetype to Go for .go files
	if strings.HasSuffix(pathStr, ".go") {
		err = s.nvimInstance.SetBufferOption(buffer, "filetype", "go")
		if err != nil {
			return 0, errors.Errorf("failed to set filetype: %w", err)
		}
	}

	return buffer, nil
}

func (s *NvimIntegrationTestRunner) AttachLSP(buf nvim.Buffer) error {
	s.t.Log("Waiting for LSP to initialize...")
	err := s.WaitForLSP()
	if err != nil {
		return errors.Errorf("failed to wait for LSP: %w", err)
	}

	// Get buffer info for logging
	bufPath, err := s.nvimInstance.BufferName(buf)
	if err != nil {
		return errors.Errorf("getting buffer name: %w", err)
	}

	s.t.Logf("Attaching LSP to buffer %s", bufPath)

	// Attach LSP client using Lua
	err = s.nvimInstance.Eval(`luaeval('vim.lsp.buf_attach_client(0, 1)')`, nil)
	if err != nil {
		return errors.Errorf("failed to attach LSP client: %w", err)
	}

	// Get current buffer text
	lines, err := s.nvimInstance.BufferLines(buf, 0, -1, true)
	if err != nil {
		return errors.Errorf("failed to get buffer lines: %w", err)
	}
	text := strings.Join(bytesSliceToStringSlice(lines), "\n")

	s.t.Logf("Sending initial didOpen notification for %s", bufPath)

	// Send file contents to LSP server using Lua
	notifyCmd := fmt.Sprintf(`luaeval('vim.lsp.buf_notify(0, "textDocument/didOpen", {
		textDocument = {
			uri = "file://%s",
			languageId = "go",
			version = 1,
			text = [[%s]]
		}
	})')`, bufPath, text)

	err = s.nvimInstance.Eval(notifyCmd, nil)
	if err != nil {
		return errors.Errorf("failed to notify LSP: %w", err)
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

func (s *NvimIntegrationTestRunner) Hover(t *testing.T, ctx context.Context, request *protocol.HoverParams) (*protocol.Hover, error) {
	///////////////////////////

	// TODO: eliminate this, since it doesnt work with larget packages

	// //Check for go.mod file
	// goModPath := filepath.Join(s.TmpDir, "go.mod")
	// if _, err := os.Stat(goModPath); os.IsNotExist(err) {
	// 	return nil, nil // Return nil if go.mod doesn't exist
	// }

	// //Check if go.mod is valid
	// goModContent, err := os.ReadFile(goModPath)
	// if err != nil {
	// 	return nil, nil // Return nil if can't read go.mod
	// }
	// if !strings.HasPrefix(string(goModContent), "module ") {
	// 	return nil, nil // Return nil if go.mod is invalid
	// }

	///////////////////////////

	buffer, err := s.OpenFile(request.TextDocument.URI)
	if err != nil {
		return nil, errors.Errorf("failed to open file: %w", err)
	}

	err = s.AttachLSP(buffer)
	if err != nil {
		return nil, errors.Errorf("failed to attach LSP: %w", err)
	}

	// Get current buffer text
	// lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
	// if err != nil {
	// 	return nil, errors.Errorf("failed to get buffer lines: %w", err)
	// }
	// text := strings.Join(bytesSliceToStringSlice(lines), "\n")
	// t.Logf("Buffer content:\n%s", text)

	// Move cursor to the specified position
	win, err := s.nvimInstance.CurrentWindow()
	if err != nil {
		return nil, errors.Errorf("failed to get current window: %w", err)
	}

	err = s.nvimInstance.SetWindowCursor(win, [2]int{int(request.Position.Line) + 1, int(request.Position.Character)})
	if err != nil {
		return nil, errors.Errorf("failed to set cursor position: %w", err)
	}

	// Request hover using Lua
	// var hoverResult *string
	// hoverCmd := `
	// 	-- Enable debug logging
	// 	vim.lsp.set_log_level("debug")

	// 	-- Print current buffer and cursor info
	// 	print("Current buffer:", vim.api.nvim_get_current_buf())
	// 	print("Cursor position:", vim.inspect(vim.api.nvim_win_get_cursor(0)))

	// 	-- Get active clients
	// 	local clients = vim.lsp.get_active_clients()
	// 	print("Active clients:", vim.inspect(clients))

	// 	local function waitForLspClient()
	// 		local count = 0
	// 		while count < 50 do  -- 5 second timeout
	// 			local clients = vim.lsp.get_active_clients()
	// 			if #clients > 0 then
	// 				print("LSP client ready")
	// 				return clients[1]
	// 			end
	// 			print("Waiting for LSP client...")
	// 			vim.cmd('sleep 100m')  -- 100ms sleep
	// 			count = count + 1
	// 		end
	// 		return nil
	// 	end

	// 	-- Store the original hover handler
	// 	local orig_handler = vim.lsp.handlers['textDocument/hover']
	// 	local result = nil
	// 	local done = false

	// 	-- Set up a custom handler to capture the result
	// 	vim.lsp.handlers['textDocument/hover'] = function(err, method, result_)
	// 		result = result_
	// 		done = true
	// 		return orig_handler(err, method, result_)
	// 	end

	// 	-- Wait for LSP client
	// 	local client = waitForLspClient()
	// 	if not client then
	// 		print("No LSP client found after timeout")
	// 		return vim.json.encode({error = "LSP not ready"})
	// 	end

	// 	-- Make the hover request
	// 	vim.lsp.buf.hover()

	// 	-- Wait for result
	// 	local count = 0
	// 	while not done and count < 50 do  -- 5 second timeout
	// 		vim.cmd('sleep 100m')  -- 100ms sleep
	// 		count = count + 1
	// 	end

	// 	-- Restore the original handler
	// 	vim.lsp.handlers['textDocument/hover'] = orig_handler

	// 	if result then
	// 		return vim.json.encode(result)
	// 	else
	// 		print("No hover result")
	// 		return vim.json.encode({error = "No hover result"})
	// 	end
	// `

	bufPath, err := s.nvimInstance.BufferName(buffer)
	if err != nil {
		return nil, errors.Errorf("failed to get buffer name: %w", err)
	}

	// Request hover using Lua
	var hoverResult *string
	hoverCmd := fmt.Sprintf(`
		local result = vim.lsp.buf_request_sync(0, "textDocument/hover", {
			textDocument = { uri = "file://%s" },
			position = { line = %d, character = %d }
		}, 1000)
		if result and result[1] and result[1].result then
			return vim.json.encode(result[1].result)
		end
		return nil
	`, bufPath, request.Position.Line, request.Position.Character)

	err = s.nvimInstance.ExecLua(hoverCmd, &hoverResult)
	if err != nil {
		return nil, errors.Errorf("failed to request hover: %w", err)
	}

	// t.Logf("Hover result: %v", hoverResult)

	if hoverResult == nil {
		return nil, nil // this is the case where there is no hover result, which is a valid case
	}

	// t.Logf("Hover result string: %v", *hoverResult)

	var hover protocol.Hover
	err = json.Unmarshal([]byte(*hoverResult), &hover)
	if err != nil {
		return nil, errors.Errorf("unmarshalling hover: %w", err)
	}
	return &hover, nil
}

func (s *NvimIntegrationTestRunner) SaveAndQuit() error {
	outFile := filepath.Join(s.TmpDir, "nvim.out")
	err := s.nvimInstance.Command("write! " + outFile)
	if err != nil {
		return errors.Errorf("failed to write file: %w", err)
	}

	err = s.nvimInstance.Command("quit!")
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

// GetDiagnostics returns the current diagnostics for a given file URI
func (s *NvimIntegrationTestRunner) GetDiagnostics(uri protocol.DocumentURI) []protocol.Diagnostic {
	// Wait a bit to ensure diagnostics have been processed
	time.Sleep(500 * time.Millisecond)

	// Open the file to make it the current buffer
	_, err := s.OpenFile(uri)
	if err != nil {
		s.t.Fatalf("failed to get buffer for diagnostics: %v", err)
	}

	// Get diagnostics using LSP API
	var diagnosticsResult *string
	getDiagnosticsCmd := `
		local diagnostics = vim.diagnostic.get(0)
		if diagnostics and #diagnostics > 0 then
			return vim.json.encode(diagnostics)
		end
		return nil
	`

	err = s.nvimInstance.ExecLua(getDiagnosticsCmd, &diagnosticsResult)
	if err != nil {
		s.t.Fatalf("failed to get diagnostics: %v", err)
	}

	if diagnosticsResult == nil {
		return nil
	}

	var diagnostics []protocol.Diagnostic
	err = json.Unmarshal([]byte(*diagnosticsResult), &diagnostics)
	if err != nil {
		s.t.Fatalf("failed to unmarshal diagnostics: %v", err)
	}

	return diagnostics
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

// CheckDiagnostics waits for and verifies expected diagnostics
func (s *NvimIntegrationTestRunner) CheckDiagnostics(t *testing.T, uri protocol.DocumentURI, expectedDiagnostics []protocol.Diagnostic, timeout time.Duration) error {
	start := time.Now()
	var lastDiags []protocol.Diagnostic

	s.t.Logf("Waiting for diagnostics (timeout: %v)", timeout)
	s.t.Logf("Expected diagnostics: %+v", expectedDiagnostics)

	for time.Since(start) < timeout {
		diags := s.GetDiagnostics(uri)
		lastDiags = diags
		s.t.Logf("Current diagnostics: %+v", diags)

		// Check if diagnostics match
		if len(diags) == len(expectedDiagnostics) {
			match := true
			for i, expected := range expectedDiagnostics {
				actual := diags[i]
				if expected.Message != actual.Message ||
					expected.Range.Start.Line != actual.Range.Start.Line ||
					expected.Range.Start.Character != actual.Range.Start.Character ||
					expected.Range.End.Line != actual.Range.End.Line ||
					expected.Range.End.Character != actual.Range.End.Character ||
					expected.Severity != actual.Severity {
					match = false
					s.t.Logf("Diagnostic mismatch at index %d:\nExpected: %+v\nActual: %+v", i, expected, actual)
					break
				}
			}
			if match {
				s.t.Log("Diagnostics matched successfully")
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	// If we get here, the timeout was reached
	return errors.Errorf("timeout waiting for expected diagnostics.\nExpected: %+v\nGot: %+v", expectedDiagnostics, lastDiags)
}

// ApplyEditWithSave applies an edit to a file and saves it
func (s *NvimIntegrationTestRunner) ApplyEditWithSave(t *testing.T, uri protocol.DocumentURI, newContent string) error {
	buf, err := s.OpenFile(uri)
	if err != nil {
		return errors.Errorf("opening file: %w", err)
	}

	// Delete all content and insert new content
	err = s.nvimInstance.Command("normal! ggdG")
	if err != nil {
		return errors.Errorf("deleting content: %w", err)
	}

	// Insert new content
	err = s.nvimInstance.Command(fmt.Sprintf("normal! i%s", newContent))
	if err != nil {
		return errors.Errorf("inserting content: %w", err)
	}

	// Save the file
	err = s.SaveFile(buf)
	if err != nil {
		return errors.Errorf("saving file: %w", err)
	}

	return nil
}

// ApplyEditWithoutSave applies an edit to a file without saving it
func (s *NvimIntegrationTestRunner) ApplyEditWithoutSave(t *testing.T, uri protocol.DocumentURI, newContent string) error {
	buf, err := s.OpenFile(uri)
	if err != nil {
		return errors.Errorf("opening file: %w", err)
	}

	// Delete all content and insert new content
	err = s.nvimInstance.Command("normal! ggdG")
	if err != nil {
		return errors.Errorf("deleting content: %w", err)
	}

	// Insert new content
	err = s.nvimInstance.Command(fmt.Sprintf("normal! i%s", newContent))
	if err != nil {
		return errors.Errorf("inserting content: %w", err)
	}

	// Notify LSP about the change without saving
	lines, err := s.nvimInstance.BufferLines(buf, 0, -1, true)
	if err != nil {
		return errors.Errorf("getting buffer lines: %w", err)
	}
	text := strings.Join(bytesSliceToStringSlice(lines), "\n")

	bufPath, err := s.nvimInstance.BufferName(buf)
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

	err = s.nvimInstance.Eval(notifyCmd, nil)
	if err != nil {
		return errors.Errorf("failed to notify LSP: %w", err)
	}

	return nil
}
