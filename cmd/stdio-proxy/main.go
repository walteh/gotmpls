package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"gitlab.com/tozd/go/errors"
)

// copyData handles copying data between src and dst with proper error handling
func copyData(ctx context.Context, dst io.Writer, src io.Reader, name string) error {
	_, err := io.Copy(dst, src)
	if err != nil && err != io.EOF {
		return errors.Errorf("copying %s: %w", name, err)
	}
	return nil
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return errors.New("invalid number of arguments")
	}

	socketPath := os.Args[1]

	// Setup connection
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return errors.Errorf("connecting to socket %s: %w", socketPath, err)
	}
	defer conn.Close()

	// Create error channel for goroutines
	errChan := make(chan error, 2)

	// Copy from stdin to socket
	go func() {
		if err := copyData(ctx, conn, os.Stdin, "stdin to socket"); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	// Copy from socket to stdout
	go func() {
		if err := copyData(ctx, os.Stdout, conn, "socket to stdout"); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	// Wait for first error or completion
	if err := <-errChan; err != nil {
		return err
	}

	return nil
}

func main() {
	// Setup logging
	ctx := context.Background()

	// Setup signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("received signal, shutting down")
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to run stdio proxy")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
