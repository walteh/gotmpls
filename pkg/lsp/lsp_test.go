package lsp_test

import (
	"archive/tar"
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
	nvimInstance *nvim.Nvim
	server       *lsp.Server
	tmpDir       string
	cleanup      func()
	t            *testing.T
}

func setupNeovimTest(t *testing.T, server *lsp.Server, files testFiles) (*neovimTestSetup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	setup := &neovimTestSetup{
		server: server,
		t:      t,
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
		if err := server.Start(ctx, conn, conn); err != nil {
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
		var b bool
		err := s.nvimInstance.Eval(`luaeval('vim.lsp.buf_get_clients() ~= nil and #vim.lsp.buf_get_clients() > 0')`, &b)
		if err != nil {
			s.t.Logf("Error checking LSP clients: %v", err)
			return false
		}
		s.t.Logf("LSP clients count: %v", b)

		if b {
			// Get LSP client info
			var clientInfo string
			err = s.nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_active_clients())')`, &clientInfo)
			if err == nil {
				s.t.Logf("LSP client info: %v", clientInfo)
			}

			// Get LSP logs
			var logs string
			err = s.nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_log_lines())')`, &logs)
			if err == nil {
				s.t.Logf("LSP logs: %v", logs)
			}
		}

		return b
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

func TestNeovimBasic(t *testing.T) {
	server := lsp.NewServer(
		parser.NewDefaultTemplateParser(),
		types.NewDefaultValidator(),
		ast.NewDefaultPackageAnalyzer(),
		diagnostic.NewDefaultGenerator(),
		true,
	)

	files := testFiles{
		"test.tmpl": "{{ .Value }}",
	}

	setup, err := setupNeovimTest(t, server, files)
	require.NoError(t, err)
	defer setup.cleanup()

	// Open and edit the test file
	testFile := filepath.Join(setup.tmpDir, "test.tmpl")
	t.Log("Opening test file...")
	err = setup.nvimInstance.Command("edit " + testFile)
	require.NoError(t, err)

	// Set filetype
	t.Log("Setting filetype...")
	err = setup.nvimInstance.Command("setfiletype go-template")
	require.NoError(t, err)

	// Wait for LSP to initialize
	err = setup.waitForLSP()
	require.NoError(t, err)

	// Notify LSP server about the file
	t.Log("Notifying LSP server about file...")
	err = setup.nvimInstance.Command("lua vim.lsp.buf_attach_client(0, 1)")
	require.NoError(t, err)

	// Send file contents to LSP server
	t.Log("Sending file contents to LSP server...")
	err = setup.nvimInstance.Command(`lua vim.lsp.buf_notify(0, 'textDocument/didOpen', {
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
	err = setup.nvimInstance.Command("normal! gg0")
	require.NoError(t, err)

	// Request hover using request_sync
	t.Log("Requesting hover...")
	err = setup.nvimInstance.Command(`lua
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
	err = setup.nvimInstance.Command("write! " + testFile + ".out")
	require.NoError(t, err)

	t.Log("Quitting neovim...")
	err = setup.nvimInstance.Command("quit!")
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
