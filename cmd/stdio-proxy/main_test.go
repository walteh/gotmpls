package main

import (
	"bytes"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdioProxy(t *testing.T) {
	// Create a temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "stdio-proxy-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Build the stdio-proxy binary
	binaryPath := filepath.Join(tmpDir, "stdio-proxy")
	buildCmd := exec.Command("go", "build", "-o", binaryPath)
	buildCmd.Dir = "." // Build in the current directory
	require.NoError(t, buildCmd.Run(), "Failed to build stdio-proxy")

	// Create a test socket
	socketPath := filepath.Join(tmpDir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	// Accept connections in a goroutine
	connChan := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		require.NoError(t, err)
		connChan <- conn
	}()

	// Start the proxy process
	proxyCmd := exec.Command(binaryPath, socketPath)
	stdin, err := proxyCmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := proxyCmd.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, proxyCmd.Start())
	defer proxyCmd.Process.Kill()

	// Wait for the connection
	var serverConn net.Conn
	select {
	case serverConn = <-connChan:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for connection")
	}
	defer serverConn.Close()

	// Test stdin -> socket
	testMsg := "Hello, socket!"
	_, err = io.WriteString(stdin, testMsg+"\n")
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := serverConn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, testMsg+"\n", string(buf[:n]))

	// Test socket -> stdout
	responseMsg := "Hello, stdio!"
	_, err = serverConn.Write([]byte(responseMsg + "\n"))
	require.NoError(t, err)

	outBuf := &bytes.Buffer{}
	_, err = io.CopyN(outBuf, stdout, int64(len(responseMsg)+1))
	require.NoError(t, err)
	require.Equal(t, responseMsg+"\n", outBuf.String())
}
