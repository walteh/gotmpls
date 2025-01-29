package nvim

import (
	"archive/tar"
	"context"
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

	"github.com/creachadair/jrpc2"
	"github.com/neovim/go-client/nvim"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	nvimlspconfig "github.com/walteh/gotmpls/gen/git-repo-tarballs/nvim-lspconfig"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/targz"
	"gitlab.com/tozd/go/errors"
)

type NvimIntegrationTestRunner struct {
	nvimInstance   *nvim.Nvim
	serverInstance *protocol.ServerDispatcher
	TmpDir         string
	t              *testing.T
	currentBuffer  *struct {
		uri    protocol.DocumentURI
		buffer nvim.Buffer
	}
	mu         sync.Mutex // Protects file operations
	rpcTracker *protocol.RPCTracker
}

func (me *NvimIntegrationTestRunner) PrintNvimLogs(t *testing.T) {
	t.Helper()

	// Check the Neovim log
	nvimLogPath := filepath.Join(me.TmpDir, "nvim.log")
	if nvimLog, err := os.ReadFile(nvimLogPath); err == nil {

		debugNvimLogLines := os.Getenv("DEBUG_NVIM_LOG_LINES")
		var inter int
		if debugNvimLogLines == "" {
			t.Logf("DEBUG_NVIM_LOG_LINES not set, skipping log")
		} else {
			if debugNvimLogLines == "all" {
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

	}

	// do the same for the gopls log
	goplsLogPath := filepath.Join(me.TmpDir, "gopls.log")
	if goplsLog, err := os.ReadFile(goplsLogPath); err == nil {

		debugGoplsLogLines := os.Getenv("DEBUG_GOPLS_LOG_LINES")
		var inter int
		if debugGoplsLogLines == "" {
			t.Logf("DEBUG_GOPLS_LOG_LINES not set, skipping log")
		} else {
			if debugGoplsLogLines == "all" {
				t.Logf("DEBUG_GOPLS_LOG_LINES set to all, WARNING: this will print a lot of logs")
				inter = math.MaxInt

			} else {
				inter, err = strconv.Atoi(debugGoplsLogLines)
				if err != nil {
					t.Logf("could not parse DEBUG_GOPLS_LOG_LINES (%s) as a number, using default of 50", debugGoplsLogLines)
					inter = 50
				}
			}
			lastLines := lastN(strings.Split(string(goplsLog), "\n"), inter)
			lastWord := "last"
			if inter == math.MaxInt {
				lastWord = "all"
			}
			t.Logf("gopls log (%s %d lines):\n%s", lastWord, len(lastLines), strings.Join(lastLines, "\n"))
		}
	}

}

func NewNvimIntegrationTestRunner(t *testing.T, files map[string]string, si *protocol.ServerDispatcher, config NeovimConfig) (*NvimIntegrationTestRunner, error) {
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
		rpcTracker:     protocol.NewRPCTracker(),
	}

	// Set the tracker on the server instance
	si.Instance().SetRPCTracker(setup.rpcTracker)

	// Create a Unix domain socket for LSP communication in the temp directory
	socketPath := filepath.Join(tmpDir, "lsp-test.sock")

	// Create cleanup function that will be called when test is done
	t.Cleanup(func() {
		cancel()
		if setup.nvimInstance != nil {
			if err := setup.nvimInstance.Close(); err != nil && err.Error() != "signal: killed" {
				t.Logf("failed to close neovim: %v", err)
			}
		}

		defer func() {
			os.RemoveAll(tmpDir)
			os.Remove(socketPath)
		}()

		setup.PrintNvimLogs(t)

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

		si.Instance().ServerOpts().RPCLog = protocol.NewTestLogger(t, map[string]string{
			tmpDir: "/[TEMP_DIR]",
		})
		si.Instance().SetRPCTracker(setup.rpcTracker)
		si.Instance().AddArgsToBackgroundCmd("-logfile=" + filepath.Join(tmpDir, "gopls.log"))
		zerolog.Ctx(ctx).Info().Msg("Starting server...")

		if err := si.Instance().StartAndWait(conn, conn); err != nil {
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
	cmd := os.Getenv("GOTMPLS_NEOVIM_BIN")
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

func setupNeovimConfig(t *testing.T, tmpDir string, socketPath string, config NeovimConfig) (string, error) {
	lspConfigDir := filepath.Join(tmpDir, "nvim-lspconfig")
	t.Log("Extracting nvim-lspconfig files...")
	err := targz.ExtractTarGzWithOptions(nvimlspconfig.Data, lspConfigDir, targz.ExtractOptions{
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
autocmd! BufEnter *.tmpl setlocal filetype=gotmpl

" Load lspconfig
runtime! plugin/lspconfig.lua

lua <<EOF
%[4]s

-- Use an on_attach function to only map the following keys
-- local on_attach = function(client, bufnr)
--     print("LSP client attached:", vim.inspect(client))
--     print("Buffer:", bufnr)
--     print("Client capabilities:", vim.inspect(client.server_capabilities))
--     
--     -- Disable semantic tokens
--     -- client.server_capabilities.semanticTokensProvider = nil
-- 
--     -- Set buffer options
--     vim.api.nvim_buf_set_option(bufnr, 'omnifunc', 'v:lua.vim.lsp.omnifunc')
-- end

print("start default config")
%[2]s
print("end default config")

print("start default setup")
%[3]s
print("end default setup")

print("LSP setup complete")
EOF`, lspConfigDir, config.DefaultConfig(socketPath), config.DefaultSetup(), sharedNeovimConfig)

	configPath := filepath.Join(tmpDir, "config.vim")
	if err := os.WriteFile(configPath, []byte(vimConfig), 0644); err != nil {
		return "", errors.Errorf("failed to write config: %w", err)
	}

	return configPath, nil
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
	// t.Skip("this is not working, so ")
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
				// t.Logf("🔍 LSP client info: %v", clientInfo)
			}
		}

		return hasClients
	}

	var success bool
	for i := 0; i < 500; i++ {
		if success = waitForLSP(); success {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !success {
		return errors.Errorf("LSP client failed to attach or initialize")
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
	t.Logf("Opening file: %s", path)
	s.mu.Lock()
	t.Logf("Locked")
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

	// t.Logf("a")

	// Force close any other buffers that might be open
	if err := s.nvimInstance.Command("%bd!"); err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to close all buffers: %w", err)
	}

	// t.Logf("b")

	// // Create a new buffer and set its name
	// createBufCmd := fmt.Sprintf(`
	// 	local buf = vim.api.nvim_create_buf(true, false)
	// 	vim.api.nvim_buf_set_name(buf, '%s')
	// 	vim.api.nvim_set_current_buf(buf)
	// 	return buf
	// `, pathStr)

	// var buffer nvim.Buffer
	// err := s.nvimInstance.ExecLua(createBufCmd, &buffer)
	// if err != nil {
	// 	cleanup()
	// 	return 0, nil, errors.Errorf("failed to create buffer: %w", err)
	// }

	// t.Logf("c")

	// // Read file content and set buffer lines
	// content, err := os.ReadFile(pathStr)
	// if err != nil {
	// 	cleanup()
	// 	return 0, nil, errors.Errorf("failed to read file: %w", err)
	// }

	// lines := strings.Split(string(content), "\n")
	// byteLines := make([][]byte, len(lines))
	// for i, line := range lines {
	// 	byteLines[i] = []byte(line)
	// }
	// if err := s.nvimInstance.SetBufferLines(buffer, 0, -1, true, byteLines); err != nil {
	// 	cleanup()
	// 	return 0, nil, errors.Errorf("failed to set buffer lines: %w", err)
	// }

	err := s.nvimInstance.Command("e " + pathStr)
	if err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to open file: %w", err)
	}

	buffer, err := s.nvimInstance.CurrentBuffer()
	if err != nil {
		cleanup()
		return 0, nil, errors.Errorf("failed to get current buffer: %w", err)
	}

	// // Set filetype to Go for .go files
	// if strings.HasSuffix(pathStr, ".go") {
	// 	if err = s.nvimInstance.SetBufferOption(buffer, "filetype", "go"); err != nil {
	// 		cleanup()
	// 		return 0, nil, errors.Errorf("failed to set filetype: %w", err)
	// 	}
	// }

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
