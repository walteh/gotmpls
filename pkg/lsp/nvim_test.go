package lsp_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/stretchr/testify/require"
)

// func TestNeovimHover(t *testing.T) {
// 	if os.Getenv("NEOVIM_TEST") == "" {
// 		t.Skip("skipping neovim test; NEOVIM_TEST not set")
// 	}

// 	t.Log("Setting up LSP config...")
// 	// Setup temporary config and socket
// 	tmpDir, configPath, socket, proxyScript := setupLSPConfig(t)
// 	t.Logf("Using tmpDir: %s", tmpDir)
// 	t.Logf("Using configPath: %s", configPath)
// 	t.Logf("Using socket: %s", socket)
// 	t.Logf("Using proxy script: %s", proxyScript)

// 	// Create a debug log file for the proxy script
// 	proxyLogPath := filepath.Join(tmpDir, "proxy.log")
// 	t.Logf("Proxy log will be written to: %s", proxyLogPath)

// 	// Start the LSP server first
// 	serverStarted := make(chan struct{})
// 	serverError := make(chan error, 1)
// 	go func() {
// 		defer close(serverStarted)
// 		defer close(serverError)

// 		// Create and start the LSP server first
// 		t.Log("Creating LSP server...")
// 		server := lsp.NewServer(
// 			parser.NewDefaultTemplateParser(),
// 			types.NewDefaultValidator(),
// 			ast.NewDefaultPackageAnalyzer(),
// 			diagnostic.NewDefaultGenerator(),
// 			true,
// 		)

// 		// Listen on the Unix socket
// 		t.Log("Starting socket listener...")
// 		listener, err := net.Listen("unix", socket)
// 		if err != nil {
// 			serverError <- fmt.Errorf("failed to listen on socket: %v", err)
// 			return
// 		}
// 		defer listener.Close()

// 		// Signal that we're ready to accept connections
// 		serverStarted <- struct{}{}

// 		// Accept a connection
// 		t.Log("Waiting for connection...")
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			serverError <- fmt.Errorf("failed to accept connection: %v", err)
// 			return
// 		}
// 		defer conn.Close()

// 		t.Log("Starting server...")
// 		if err := server.Start(context.Background(), conn, conn); err != nil {
// 			if err != io.EOF {
// 				serverError <- fmt.Errorf("LSP server error: %v", err)
// 			}
// 			t.Log("Server stopped with:", err)
// 		}
// 	}()

// 	// Wait for the server to be ready or error
// 	select {
// 	case err := <-serverError:
// 		t.Fatalf("LSP server failed to start: %v", err)
// 	case <-serverStarted:
// 		t.Log("LSP server ready")
// 	case <-time.After(5 * time.Second):
// 		t.Fatal("timeout waiting for LSP server to start")
// 	}

// 	// Create a context with timeout for the entire test
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	// Start Neovim with the context
// 	t.Log("Creating neovim instance...")
// 	cmd := os.Getenv("GO_TMPL_TYPER_NEOVIM_BIN")
// 	if cmd == "" {
// 		var err error
// 		cmd, err = exec.LookPath("nvim")
// 		if err != nil {
// 			t.Skip("nvim not installed; skipping LSP tests")
// 		}
// 	}
// 	t.Logf("Using nvim command: %s", cmd)

// 	nvimInstance, err := nvim.NewChildProcess(
// 		nvim.ChildProcessCommand(cmd),
// 		nvim.ChildProcessArgs("--clean", "-n", "--embed", "--headless", "--noplugin", "-u", configPath, "-V20"+filepath.Join(tmpDir, "nvim.log")),
// 		nvim.ChildProcessContext(ctx),
// 		nvim.ChildProcessLogf(t.Logf),
// 	)
// 	require.NoError(t, err)
// 	defer func() {
// 		t.Log("Closing neovim instance...")
// 		err := nvimInstance.Close()
// 		if err != nil {
// 			t.Logf("failed to close neovim: %v", err)
// 		}

// 		// Check the proxy log
// 		if proxyLog, err := os.ReadFile(proxyLogPath); err == nil {
// 			t.Logf("Proxy script log:\n%s", string(proxyLog))
// 		} else {
// 			t.Logf("Failed to read proxy log: %v", err)
// 		}

// 		// Check the Neovim log
// 		nvimLogPath := filepath.Join(tmpDir, "nvim.log")
// 		if nvimLog, err := os.ReadFile(nvimLogPath); err == nil {
// 			lastLines := lastN(strings.Split(string(nvimLog), "\n"), 50)
// 			t.Logf("Neovim log (last 50 lines):\n%s", strings.Join(lastLines, "\n"))
// 		} else {
// 			t.Logf("Failed to read Neovim log: %v", err)
// 		}
// 	}()

// 	// Explicitly source our config
// 	t.Log("Sourcing LSP config...")
// 	err = nvimInstance.Command("source " + configPath)
// 	require.NoError(t, err)

// 	// Wait for LSP to initialize
// 	t.Log("Waiting for LSP to initialize...")
// 	waitForLSP := func() bool {
// 		var b bool
// 		err := nvimInstance.Eval(`luaeval('vim.lsp.buf_get_clients() ~= nil and #vim.lsp.buf_get_clients() > 0')`, &b)
// 		if err != nil {
// 			t.Logf("Error checking LSP clients: %v", err)
// 			return false
// 		}
// 		t.Logf("LSP clients count: %v", b)

// 		if b {
// 			// Get LSP client info
// 			var clientInfo string
// 			err = nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_active_clients())')`, &clientInfo)
// 			if err == nil {
// 				t.Logf("LSP client info: %v", clientInfo)
// 			}

// 			// Get LSP logs
// 			var logs string
// 			err = nvimInstance.Eval(`luaeval('vim.inspect(vim.lsp.get_log_lines())')`, &logs)
// 			if err == nil {
// 				t.Logf("LSP logs: %v", logs)
// 			}
// 		}

// 		return b
// 	}
// 	require.Eventually(t, waitForLSP, 10*time.Second, 100*time.Millisecond, "LSP client failed to attach")

// 	t.Log("Enabling LSP logging...")
// 	err = nvimInstance.Command("lua vim.lsp.set_log_level('debug')")
// 	require.NoError(t, err)
// 	err = nvimInstance.Command("lua require('vim.lsp.log').set_format_func(vim.inspect)")
// 	require.NoError(t, err)

// 	t.Log("Getting current working directory...")
// 	cwd, err := os.Getwd()
// 	require.NoError(t, err)

// 	t.Log("Getting runtimepath...")
// 	var runtimepath string
// 	err = nvimInstance.Eval(`&runtimepath`, &runtimepath)
// 	require.NoError(t, err)
// 	t.Logf("runtimepath: %v", strings.Split(runtimepath, ","))

// 	t.Log("Opening test file...")
// 	testFile := filepath.Join(cwd, "testdata", "hover.tmpl")
// 	err = nvimInstance.Command("edit " + testFile)
// 	require.NoError(t, err)

// 	t.Log("Getting buffer number...")
// 	var bufnr int
// 	err = nvimInstance.Eval("bufnr('%')", &bufnr)
// 	require.NoError(t, err)

// 	t.Log("Getting buffer lines...")
// 	lines, err := nvimInstance.BufferLines(nvim.Buffer(bufnr), 0, -1, true)
// 	require.NoError(t, err)
// 	t.Logf("lines: %d", len(lines))

// 	t.Log("Moving cursor...")
// 	err = nvimInstance.SetWindowCursor(0, [2]int{1, 31})
// 	require.NoError(t, err)

// 	// Create a channel to signal completion
// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)

// 		t.Log("Triggering hover...")
// 		err = nvimInstance.Command("lua vim.lsp.buf.hover()")
// 		if err != nil {
// 			t.Errorf("failed to trigger hover: %v", err)
// 			return
// 		}

// 		t.Log("Waiting for hover response...")
// 		time.Sleep(1 * time.Second)

// 		t.Log("Checking for hover window...")
// 		var floatWinVisible bool
// 		err = nvimInstance.Eval("len(nvim_list_wins()) > 1", &floatWinVisible)
// 		if err != nil {
// 			t.Errorf("failed to check float window: %v", err)
// 			return
// 		}
// 		if !floatWinVisible {
// 			t.Error("hover window should be visible")
// 			return
// 		}

// 		t.Log("Getting hover window content...")
// 		var floatWinContent [][]byte
// 		err = nvimInstance.Eval(`getbufline(winbufnr(v:lua.vim.lsp.util.get_floating_windows()[1]), 1, '$')`, &floatWinContent)
// 		if err != nil {
// 			t.Errorf("failed to get float window content: %v", err)
// 			return
// 		}
// 		if len(floatWinContent) == 0 {
// 			t.Error("hover content should not be empty")
// 			return
// 		}

// 		for _, line := range floatWinContent {
// 			t.Logf("L%03d K     line %q", len(line), string(line))
// 		}

// 		t.Log("Closing hover window...")
// 		err = nvimInstance.Command("wincmd p")
// 		if err != nil {
// 			t.Errorf("failed to close hover window: %v", err)
// 			return
// 		}
// 	}()

// 	select {
// 	case <-done:
// 		t.Log("Test completed successfully")
// 	case <-time.After(10 * time.Second):
// 		t.Fatal("test timed out")
// 	}
// }

func checkNested(t *testing.T) bool {
	if os.Getenv("NVIM") != "" {
		t.Skip("detected running from neovim; skipping to avoid hanging")
		return true
	}

	return false
}

func testFile(t *testing.T, client *nvim.Nvim, file string) {
	require := require.New(t)

	err := client.Command(`edit ` + file)
	require.NoError(err)

	testBuf, err := client.CurrentBuffer()
	require.NoError(err)

	window, err := client.CurrentWindow()
	require.NoError(err)

	// Wait for LSP client to attach
	waitForLSP := func() bool {
		var b bool
		err := client.Eval(`luaeval('#vim.lsp.buf_get_clients() > 0')`, &b)
		return err == nil && b
	}
	require.Eventually(waitForLSP, 5*time.Second, 10*time.Millisecond)

	lineCount, err := client.BufferLineCount(testBuf)
	require.NoError(err)

	t.Logf("lines: %d", lineCount)

	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		lspLogs, err := os.ReadFile("go-tmpl-typer-lsp.log")
		if err == nil {
			t.Logf("language server logs:\n\n%s", string(lspLogs))
		}
	})

	for testLine := 1; testLine <= lineCount; testLine++ {
		mode, err := client.Mode()
		require.NoError(err)

		if mode.Mode != "n" {
			// Reset back to normal mode
			err = client.FeedKeys("\x1b", "t", true)
			require.NoError(err)
		}

		err = client.SetWindowCursor(window, [2]int{testLine, 0})
		require.NoError(err)

		lineb, err := client.CurrentLine()
		require.NoError(err)
		line := string(lineb)

		segs := strings.Split(line, "; test: ")
		if len(segs) < 2 {
			continue
		}

		eq := strings.Split(segs[1], " => ")

		codes := strings.TrimSpace(eq[0])
		keys, err := client.ReplaceTermcodes(codes, true, true, true)
		require.NoError(err)

		err = client.FeedKeys(keys, "t", true)
		require.NoError(err)

		targetPos := strings.Index(eq[1], "┃")
		target := strings.ReplaceAll(eq[1], "┃", "")
		target = strings.ReplaceAll(target, "\\t", "\t")

		// Wait for hover info
		waitForHover := func() bool {
			line, err := client.CurrentLine()
			require.NoError(err)

			pos, err := client.WindowCursor(window)
			require.NoError(err)

			idx := strings.Index(string(line), target)
			if idx == -1 {
				t.Logf("L%03d %s\tline %q does not contain %q", testLine, codes, string(line), target)
				return false
			}

			col := targetPos + idx // account for leading whitespace

			if pos[1] != col {
				t.Logf("L%03d %s\tline %q: at %d, need %d", testLine, codes, string(line), pos[1], col)
				return false
			}

			t.Logf("L%03d %s\tmatched: %s", testLine, codes, eq[1])

			return true
		}
		require.Eventually(waitForHover, 5*time.Second, 10*time.Millisecond)

		// Go back to initial test buffer
		err = client.SetCurrentBuffer(testBuf)
		require.NoError(err)
	}
}

func sandboxNvim(t *testing.T) *nvim.Nvim {
	require := require.New(t)

	ctx := context.Background()

	cmd := os.Getenv("GO_TMPL_TYPER_NEOVIM_BIN")
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("nvim")
		if err != nil {
			t.Skip("nvim not installed; skipping LSP tests")
		}
	}

	client, err := nvim.NewChildProcess(
		nvim.ChildProcessCommand(cmd),
		nvim.ChildProcessArgs("--clean", "-n", "--embed", "--headless", "--noplugin", "-V10nvim.log"),
		nvim.ChildProcessContext(ctx),
		nvim.ChildProcessLogf(t.Logf),
	)
	require.NoError(err)

	t.Cleanup(func() {
		err := client.Close()
		if err != nil {
			t.Logf("failed to close neovim: %s", err)
		}

		if t.Failed() {
			nvimLogs, err := os.ReadFile("nvim.log")
			if err == nil {
				for _, line := range lastN(strings.Split(string(nvimLogs), "\n"), 20) {
					t.Logf("neovim: %s", line)
				}
			}
		}
	})

	err = client.Command(`source testdata/config.vim`)
	require.NoError(err)

	paths, err := client.RuntimePaths()
	require.NoError(err)

	t.Logf("runtimepath: %v", paths)

	return client
}

func lastN[T any](vals []T, n int) []T {
	if len(vals) <= n {
		return vals
	}

	return vals[len(vals)-n:]
}
