package lsp_test

import (
	"archive/tar"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/archive"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

//go:embed testdata/gen/nvim-lspconfig.tar.gz
var lspConfigTarGz []byte

// testFiles represents a map of file paths to their contents
type testFiles map[string]string

// neovimTestSetup contains all the necessary components for a neovim LSP test
type neovimTestSetup struct {
	nvimInstance  *nvim.Nvim
	serverSpawner *lsp.ServerSpawner
	tmpDir        string
	cleanup       func()
	t             *testing.T
}

func setupNeovimTest(t *testing.T, server *lsp.ServerSpawner, files testFiles) (*neovimTestSetup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	setup := &neovimTestSetup{
		serverSpawner: server,
		t:             t,
	}

	tmpDir, err := os.MkdirTemp("", "nvim-lspconfig-*")
	if err != nil {
		cancel()
		return nil, errors.Errorf("failed to create temp dir: %v", err)
	}

	// Create a Unix domain socket for LSP communication in the temp directory
	socketPath := filepath.Join(tmpDir, "lsp-test.sock")

	// Create cleanup function that will be called when test is done
	cleanup := func() {
		cancel()
		if setup.nvimInstance != nil {
			if err := setup.nvimInstance.Close(); err != nil {
				t.Logf("failed to close neovim: %v", err)
			}
		}
		os.RemoveAll(tmpDir)
		os.Remove(socketPath)
		// Check the Neovim log
		nvimLogPath := filepath.Join(tmpDir, "nvim.log")
		if nvimLog, err := os.ReadFile(nvimLogPath); err == nil {
			lastLines := lastN(strings.Split(string(nvimLog), "\n"), 50)
			t.Logf("Neovim log (last 50 lines):\n%s", strings.Join(lastLines, "\n"))
		}
	}
	setup.cleanup = cleanup

	// Listen on the Unix socket
	t.Log("Starting socket listener...")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		cleanup()
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
		if err := server.Spawn(ctx, conn, conn, zerolog.NewTestWriter(t)); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				serverError <- errors.Errorf("LSP server error: %v", err)
			}
			t.Log("Server stopped with:", err)
		}
	}()

	// Wait for the server to be ready or error
	select {
	case err := <-serverError:
		cleanup()
		return nil, errors.Errorf("LSP server failed to start: %v", err)
	case <-serverStarted:
		t.Log("LSP server ready")
	case <-time.After(5 * time.Second):
		cleanup()
		return nil, errors.Errorf("timeout waiting for LSP server to start")
	}

	configPath, err := setupNeovimConfig(t, tmpDir, socketPath)
	if err != nil {
		cleanup()
		return nil, errors.Errorf("failed to setup LSP config: %v", err)
	}
	setup.tmpDir = tmpDir

	// Create test files
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			cleanup()
			return nil, errors.Errorf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			cleanup()
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
			cleanup()
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
		cleanup()
		return nil, errors.Errorf("failed to create neovim instance: %v", err)
	}
	setup.nvimInstance = nvimInstance

	// Explicitly source our config
	t.Log("Sourcing LSP config...")
	if err := nvimInstance.Command("source " + configPath); err != nil {
		cleanup()
		return nil, errors.Errorf("failed to source config: %v", err)
	}

	return setup, nil
}

// Helper method to wait for LSP to initialize
func (s *neovimTestSetup) waitForLSP() error {
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
	for i := 0; i < 100; i++ {
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

func setupNeovimConfig(t *testing.T, tmpDir string, socketPath string) (string, error) {
	lspConfigDir := filepath.Join(tmpDir, "nvim-lspconfig")
	t.Log("Extracting nvim-lspconfig files...")
	err := archive.ExtractTarGzWithOptions(lspConfigTarGz, lspConfigDir, archive.ExtractOptions{
		StripComponents: 1, // Remove the "nvim-lspconfig-master" prefix
		Filter: func(header *tar.Header) bool {
			return header.Name != "" // Skip empty paths
		},
	})
	if err != nil {
		return "", errors.Errorf("failed to extract nvim-lspconfig: %w", err)
	}

	// Get absolute paths for commands
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		return "", errors.Errorf("failed to get project root: %w", err)
	}
	stdioProxyPath := filepath.Join(projectRoot, "cmd", "stdio-proxy")

	// Create a temporary config.vim
	vimConfig := fmt.Sprintf(`
set verbose=20
let s:lspconfig_path = '%[1]s/nvim-lspconfig'
let &runtimepath = s:lspconfig_path . ',' . $VIMRUNTIME . ',' . s:lspconfig_path . '/after'
set packpath=%[1]s/nvim-lspconfig

" Set up filetype detection
autocmd! BufEnter *.tmpl setlocal filetype=go-template

" Load lspconfig
runtime! plugin/lspconfig.lua

lua <<EOF
local lspconfig = require 'lspconfig'
local configs = require 'lspconfig.configs'

-- Use an on_attach function to only map the following keys
local on_attach = function(client, bufnr)
    print("LSP client attached:", vim.inspect(client))
    print("Buffer:", bufnr)
end

-- Configure the go-template language server
if not configs.go_template then
    configs.go_template = {
        default_config = {
            -- Use stdio-proxy to connect to the LSP server
            cmd = { 'go', 'run', '%[2]s', '%[3]s' },
            filetypes = { 'go-template' },
            root_dir = function(fname)
                return vim.fn.getcwd()
            end,
            settings = {},
            init_options = {}
        },
    }
end

-- Set up logging
vim.lsp.set_log_level("debug")

-- Set up the LSP server
if lspconfig.go_template then
    lspconfig.go_template.setup {
        on_attach = on_attach,
        flags = {
            debounce_text_changes = 150,
            allow_incremental_sync = true,
        }
    }
end

print("LSP setup complete")
EOF`, tmpDir, stdioProxyPath, socketPath)

	configPath := filepath.Join(tmpDir, "config.vim")
	if err := os.WriteFile(configPath, []byte(vimConfig), 0644); err != nil {
		return "", errors.Errorf("failed to write config: %w", err)
	}

	return configPath, nil
}

// Helper methods for neovimTestSetup
func (s *neovimTestSetup) openFile(path string) (nvim.Buffer, error) {
	path = strings.TrimPrefix(path, "file://")

	err := s.nvimInstance.Command("edit " + path)
	if err != nil {
		return 0, errors.Errorf("failed to open file: %w", err)
	}

	buffer, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		return 0, errors.Errorf("failed to get current buffer: %w", err)
	}

	err = s.nvimInstance.SetBufferOption(buffer, "filetype", "go-template")
	if err != nil {
		return 0, errors.Errorf("failed to set filetype: %w", err)
	}

	return buffer, nil
}

func (s *neovimTestSetup) attachLSP(buffer nvim.Buffer) error {
	err := s.waitForLSP()
	if err != nil {
		return errors.Errorf("failed to wait for LSP: %w", err)
	}

	// Attach LSP client using Lua
	err = s.nvimInstance.Eval(`luaeval('vim.lsp.buf_attach_client(0, 1)')`, nil)
	if err != nil {
		return errors.Errorf("failed to attach LSP client: %w", err)
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

	notifyCmd := fmt.Sprintf(`luaeval('vim.lsp.buf_notify(0, "textDocument/didOpen", {
		textDocument = {
			uri = "file://%s",
			languageId = "go-template",
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

func (s *neovimTestSetup) requestHover(t *testing.T, ctx context.Context, request *lsp.HoverParams) (*lsp.Hover, error) {
	// Check for go.mod file
	goModPath := filepath.Join(s.tmpDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, nil // Return nil if go.mod doesn't exist
	}

	// Check if go.mod is valid
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, nil // Return nil if can't read go.mod
	}
	if !strings.HasPrefix(string(goModContent), "module ") {
		return nil, nil // Return nil if go.mod is invalid
	}

	buffer, err := s.openFile(request.TextDocument.URI)
	if err != nil {
		return nil, errors.Errorf("failed to open file: %w", err)
	}

	err = s.attachLSP(buffer)
	if err != nil {
		return nil, errors.Errorf("failed to attach LSP: %w", err)
	}

	// Get current buffer text
	lines, err := s.nvimInstance.BufferLines(buffer, 0, -1, true)
	if err != nil {
		return nil, errors.Errorf("failed to get buffer lines: %w", err)
	}
	text := strings.Join(bytesSliceToStringSlice(lines), "\n")
	t.Logf("Buffer content:\n%s", text)

	// Move cursor to the specified position
	win, err := s.nvimInstance.CurrentWindow()
	if err != nil {
		return nil, errors.Errorf("failed to get current window: %w", err)
	}

	err = s.nvimInstance.SetWindowCursor(win, [2]int{request.Position.Line + 1, request.Position.Character})
	if err != nil {
		return nil, errors.Errorf("failed to set cursor position: %w", err)
	}

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

	t.Logf("Hover result: %v", hoverResult)

	if hoverResult == nil {
		return nil, nil // this is the case where there is no hover result, which is a valid case
	}

	t.Logf("Hover result string: %v", *hoverResult)

	var hover lsp.Hover
	err = json.Unmarshal([]byte(*hoverResult), &hover)
	if err != nil {
		return nil, errors.Errorf("unmarshalling hover: %w", err)
	}
	return &hover, nil
}

func (s *neovimTestSetup) saveAndQuit() error {
	outFile := filepath.Join(s.tmpDir, "nvim.out")
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

func (s *neovimTestSetup) saveAndQuitWithOutput() (string, error) {
	err := s.saveAndQuit()
	if err != nil {
		return "", errors.Errorf("failed to save and quit: %w", err)
	}

	outFile := filepath.Join(s.tmpDir, "nvim.out")
	content, err := os.ReadFile(outFile)
	if err != nil {
		return "", errors.Errorf("failed to read output file: %w", err)
	}

	return string(content), nil
}

func TestNeovimBasic(t *testing.T) {
	ctx := context.Background()

	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	files := testFiles{
		"test.tmpl": "{{- /*gotype: test.Items*/ -}}\n{{ .Value }}",
		"go.mod":    "module test",
		"test.go": `
package test
type Items struct {
	Value string
}`,
	}

	setup, err := setupNeovimTest(t, server, files)
	require.NoError(t, err)
	defer setup.cleanup()

	// // Open test file and set up LSP
	testFile := filepath.Join(setup.tmpDir, "test.tmpl")

	// Test hover at the start of the file
	hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
		Position:     lsp.Position{Line: 1, Character: 6},
	})
	require.NoError(t, err)
	require.NotNil(t, hoverResult)
	// Save and quit
	output, err := setup.saveAndQuitWithOutput()
	require.NoError(t, err)

	// Verify the output
	require.NotEmpty(t, output, "neovim output should not be empty")

	require.Equal(t, "**Variable**: Items.Value\n**Type**: string", hoverResult.Contents.Value)

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
