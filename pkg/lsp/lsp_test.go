package lsp_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
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
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
)

//go:embed testdata/gen/nvim-lspconfig.tar.gz
var lspConfigTarGz []byte

func setupLSPConfig(t *testing.T) (string, string) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "nvim-lspconfig-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Extract nvim-lspconfig from embedded tar.gz
	gzr, err := gzip.NewReader(bytes.NewReader(lspConfigTarGz))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)

	t.Log("Extracting nvim-lspconfig files...")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// Remove the "nvim-lspconfig-master" prefix from the path
		targetPath := strings.TrimPrefix(header.Name, "nvim-lspconfig-master/")
		if targetPath == "" {
			continue
		}

		target := filepath.Join(tmpDir, "nvim-lspconfig", targetPath)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0755)
			require.NoError(t, err)
		case tar.TypeReg:
			dir := filepath.Dir(target)
			err = os.MkdirAll(dir, 0755)
			require.NoError(t, err)

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			require.NoError(t, err)
			_, err = io.Copy(f, tr)
			require.NoError(t, err)
			f.Close()
		}
	}

	// Get absolute paths for commands
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
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
            cmd = { 'go', 'run', '%[2]s', '/tmp/lsp-test.sock' },
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
EOF`, tmpDir, stdioProxyPath)

	configPath := filepath.Join(tmpDir, "config.vim")
	err = os.WriteFile(configPath, []byte(vimConfig), 0644)
	require.NoError(t, err)

	return tmpDir, configPath
}

func TestNeovimBasic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a Unix domain socket for LSP communication
	socketPath := "/tmp/lsp-test.sock"
	os.Remove(socketPath) // Clean up any existing socket

	// Create and start the LSP server
	t.Log("Creating LSP server...")
	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	// Listen on the Unix socket
	t.Log("Starting socket listener...")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	// Start the LSP server in a goroutine
	serverStarted := make(chan struct{})
	serverError := make(chan error, 1)
	go func() {
		defer close(serverStarted)
		defer close(serverError)

		// Signal that we're ready to accept connections
		serverStarted <- struct{}{}

		// Accept a connection
		t.Log("Waiting for connection...")
		conn, err := listener.Accept()
		if err != nil {
			serverError <- fmt.Errorf("failed to accept connection: %v", err)
			return
		}
		defer conn.Close()

		t.Log("Starting server...")
		if err := server.Start(ctx, conn, conn); err != nil {
			if err != io.EOF {
				serverError <- fmt.Errorf("LSP server error: %v", err)
			}
			t.Log("Server stopped with:", err)
		}
	}()

	// Wait for the server to be ready or error
	select {
	case err := <-serverError:
		t.Fatalf("LSP server failed to start: %v", err)
	case <-serverStarted:
		t.Log("LSP server ready")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for LSP server to start")
	}

	tmpDir, configPath := setupLSPConfig(t)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.tmpl")
	err = os.WriteFile(testFile, []byte("{{ .Value }}"), 0644)
	require.NoError(t, err)

	// Start Neovim with the context
	t.Log("Creating neovim instance...")
	cmd := os.Getenv("GO_TMPL_TYPER_NEOVIM_BIN")
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("nvim")
		if err != nil {
			t.Skip("nvim not installed; skipping LSP tests")
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
	require.NoError(t, err)
	defer func() {
		t.Log("Closing neovim instance...")
		err := nvimInstance.Close()
		if err != nil {
			t.Logf("failed to close neovim: %v", err)
		}

		// Check the Neovim log
		nvimLogPath := filepath.Join(tmpDir, "nvim.log")
		if nvimLog, err := os.ReadFile(nvimLogPath); err == nil {
			lastLines := lastN(strings.Split(string(nvimLog), "\n"), 50)
			t.Logf("Neovim log (last 50 lines):\n%s", strings.Join(lastLines, "\n"))
		} else {
			t.Logf("Failed to read Neovim log: %v", err)
		}
	}()

	// Explicitly source our config
	t.Log("Sourcing LSP config...")
	err = nvimInstance.Command("source " + configPath)
	require.NoError(t, err)

	// Open and edit the test file
	t.Log("Opening test file...")
	err = nvimInstance.Command("edit " + testFile)
	require.NoError(t, err)

	// Set filetype
	t.Log("Setting filetype...")
	err = nvimInstance.Command("setfiletype go-template")
	require.NoError(t, err)

	// Wait for LSP to initialize
	t.Log("Waiting for LSP to initialize...")
	waitForLSP := func() bool {
		var b bool
		err := nvimInstance.Eval(`luaeval('vim.lsp.buf_get_clients() ~= nil and #vim.lsp.buf_get_clients() > 0')`, &b)
		if err != nil {
			t.Logf("Error checking LSP clients: %v", err)
			return false
		}
		t.Logf("LSP clients count: %v", b)

		if b {
			// Get LSP client info
			var clientInfo string
			err = nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_active_clients())')`, &clientInfo)
			if err == nil {
				t.Logf("LSP client info: %v", clientInfo)
			}

			// Get LSP logs
			var logs string
			err = nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_log_lines())')`, &logs)
			if err == nil {
				t.Logf("LSP logs: %v", logs)
			}
		}

		return b
	}
	require.Eventually(t, waitForLSP, 10*time.Second, 100*time.Millisecond, "LSP client failed to attach")

	// Notify LSP server about the file
	t.Log("Notifying LSP server about file...")
	err = nvimInstance.Command("lua vim.lsp.buf_attach_client(0, 1)")
	require.NoError(t, err)

	// Send file contents to LSP server
	t.Log("Sending file contents to LSP server...")
	err = nvimInstance.Command(`lua vim.lsp.buf_notify(0, 'textDocument/didOpen', {
		textDocument = {
			uri = vim.uri_from_fname(vim.fn.expand('%:p')),
			languageId = 'go-template',
			version = 1,
			text = vim.fn.join(vim.fn.getline(1, '$'), '\n')
		}
	})`)
	require.NoError(t, err)

	// Wait for LSP server to process the file
	time.Sleep(100 * time.Millisecond)

	// Test hover functionality
	t.Log("Testing hover...")
	err = nvimInstance.Command("normal! gg0")
	require.NoError(t, err)

	// Request hover using request_sync
	t.Log("Requesting hover...")
	err = nvimInstance.Command(`lua
		local params = vim.lsp.util.make_position_params()
		local result = vim.lsp.buf_request_sync(0, 'textDocument/hover', params, 1000)
		if result and result[1] then
			local hover = result[1].result
			if hover then
				print("Hover result:", vim.inspect(hover))
			else
				print("No hover result")
			end
		else
			print("No response from server")
		end
	`)
	require.NoError(t, err)

	// Wait for hover response
	time.Sleep(100 * time.Millisecond)

	// Write the file and quit
	t.Log("Writing file...")
	err = nvimInstance.Command("write! " + testFile + ".out")
	require.NoError(t, err)

	t.Log("Quitting neovim...")
	err = nvimInstance.Command("quit!")
	if err != nil && !strings.Contains(err.Error(), "msgpack/rpc: session closed") {
		t.Errorf("unexpected error quitting neovim: %v", err)
	}

	// Check if the output file was created
	outFile := testFile + ".out"
	_, err = os.Stat(outFile)
	require.NoError(t, err, "Output file should exist")

	// Read the output file
	content, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "{{ .Value }}\n", string(content), "File content should match")
}
