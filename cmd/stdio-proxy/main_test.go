package main

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/tozd/go/errors"
)

func TestStdioProxy(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	testErr := make(chan error, 1)

	go func() {
		defer close(done)
		// Create a temporary directory for our test
		tmpDir, err := os.MkdirTemp("", "stdio-proxy-test-*")
		if err != nil {
			testErr <- err
			return
		}
		defer os.RemoveAll(tmpDir)

		prevStdin := os.Stdin
		prevStdout := os.Stdout
		prevOsArgs := os.Args
		defer func() {
			os.Stdin = prevStdin
			os.Stdout = prevStdout
			os.Args = prevOsArgs
		}()

		// Create pipes for stdin/stdout
		stdinR, stdinW, err := os.Pipe()
		if err != nil {
			testErr <- err
			return
		}
		stdoutR, stdoutW, err := os.Pipe()
		if err != nil {
			testErr <- err
			return
		}

		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Create a test socket
		socketPath := filepath.Join(tmpDir, "test.sock")
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			testErr <- err
			return
		}

		// Accept connections in a goroutine
		connChan := make(chan net.Conn)
		acceptErr := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				acceptErr <- err
				return
			}
			connChan <- conn
			close(connChan)
		}()

		os.Args = []string{"stdio-proxy", socketPath}

		// Start the proxy process in a goroutine
		proxyDone := make(chan struct{})
		go func() {
			main()
			close(proxyDone)
		}()

		// Wait for the connection
		var serverConn net.Conn
		select {
		case serverConn = <-connChan:
			if serverConn == nil {
				testErr <- errors.New("server connection is nil")
				return
			}
		case err := <-acceptErr:
			testErr <- err
			return
		case <-time.After(2 * time.Second):
			testErr <- errors.New("timeout waiting for connection")
			return
		}

		// Test stdin -> socket
		testMsg := "Hello, socket!"
		_, err = io.WriteString(stdinW, testMsg+"\n")
		if err != nil {
			testErr <- err
			return
		}

		buf := make([]byte, 1024)
		n, err := serverConn.Read(buf)
		if err != nil {
			testErr <- err
			return
		}
		if string(buf[:n]) != testMsg+"\n" {
			testErr <- errors.Errorf("expected %q, got %q", testMsg+"\n", string(buf[:n]))
			return
		}

		// Test socket -> stdout
		responseMsg := "Hello, stdio!"
		_, err = serverConn.Write([]byte(responseMsg + "\n"))
		if err != nil {
			testErr <- err
			return
		}

		outBuf := &bytes.Buffer{}
		_, err = io.CopyN(outBuf, stdoutR, int64(len(responseMsg)+1))
		if err != nil {
			testErr <- err
			return
		}
		if outBuf.String() != responseMsg+"\n" {
			testErr <- errors.Errorf("expected %q, got %q", responseMsg+"\n", outBuf.String())
			return
		}

		// Clean shutdown
		stdinW.Close()
		serverConn.Close()
		stdoutW.Close()
		listener.Close()

		// Wait for proxy to finish
		select {
		case <-proxyDone:
			// Success
		case <-time.After(time.Second):
			testErr <- errors.New("timeout waiting for proxy to finish")
			return
		}
	}()

	// Add overall test timeout and collect errors
	select {
	case <-done:
		// Check if there were any errors
		select {
		case err := <-testErr:
			t.Fatal(err)
		default:
			// Test completed successfully
		}
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out after 2 seconds")
	}
}
