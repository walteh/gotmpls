package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <socket-path>\n", os.Args[0])
		os.Exit(1)
	}

	socketPath := os.Args[1]

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to socket %s: %v\n", socketPath, err)
		os.Exit(1)
	}
	defer conn.Close()

	// Create channels to signal when copying is done
	done := make(chan struct{}, 2)

	// Copy from stdin to socket
	go func() {
		io.Copy(conn, os.Stdin)
		done <- struct{}{}
	}()

	// Copy from socket to stdout
	go func() {
		io.Copy(os.Stdout, conn)
		done <- struct{}{}
	}()

	// Wait for either copy operation to finish
	<-done
}
