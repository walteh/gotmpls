package protocol_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/walteh/gotmpls/gen/mockery"
	"github.com/walteh/gotmpls/pkg/lsp/protocol"
)

func TestInitializationHandshake(t *testing.T) {
	t.Parallel()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create pipes for bidirectional communication
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	// Create buffered channels for synchronization
	serverReady := make(chan struct{})

	// Create mock server
	mockServer := mockery.NewMockServer_protocol(t)

	// Create channels for synchronization
	initializeStarted := make(chan struct{})
	initializeCompleted := make(chan struct{})
	initializedReceived := make(chan struct{})
	startTime := time.Now() // Track timing from the beginning

	// Set up expectations
	mockServer.EXPECT().
		Initialize(mock.Anything, mock.MatchedBy(func(params *protocol.ParamInitialize) bool {
			t.Logf("🔍 Initialize called with params: %+v", params)
			t.Logf("🕒 Time since start: %v", time.Since(startTime))
			return true // Accept any params for now to debug
		})).
		Run(func(ctx context.Context, params *protocol.ParamInitialize) {
			t.Log("⚡ Initialize handler running")
			t.Logf("🕒 Time in handler: %v", time.Since(startTime))
			close(initializeStarted)
			// Check if context is cancelled
			if ctx.Err() != nil {
				t.Logf("❌ Initialize context error: %v", ctx.Err())
				return
			}
			t.Logf("📋 Initialize params: ProcessID=%d, RootURI=%s", params.ProcessID, params.RootURI)
			t.Log("✅ Initialize handler completed")
			close(initializeCompleted)
		}).
		Return(&protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				TextDocumentSync: &protocol.Or_ServerCapabilities_textDocumentSync{
					Value: protocol.Incremental,
				},
			},
		}, nil).
		Once()

	mockServer.EXPECT().
		Initialized(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, params *protocol.InitializedParams) {
			t.Log("📬 Initialized notification received")
			close(initializedReceived)
		}).
		Return(nil).
		Once()

	mockServer.EXPECT().
		Shutdown(mock.Anything).
		Run(func(ctx context.Context) {
			t.Log("🛑 Shutdown received")
			if ctx.Err() != nil {
				t.Logf("❌ Shutdown context error: %v", ctx.Err())
				return
			}
		}).
		Return(nil).
		Once()

	mockServer.EXPECT().
		Exit(mock.Anything).
		Run(func(ctx context.Context) {
			t.Log("🚪 Exit notification received")
		}).
		Return(nil).
		Once()

	// Create RPC tracker with enhanced logging
	rpcTracker := protocol.NewRPCTracker()

	// Create server instance with RPC logging
	serverOpts := &jrpc2.ServerOptions{
		RPCLog:      &testRPCLogger{t: t},
		Concurrency: 1,    // Ensure single request handling
		AllowPush:   true, // Allow server-initiated messages
	}
	t.Log("🔧 Creating server instance")
	serverCtx := protocol.ContextWithRPCTracker(ctx, rpcTracker)
	serverInstance := protocol.NewServerInstance(serverCtx, mockServer, serverOpts)
	serverInstance.Instance().SetRPCTracker(rpcTracker)

	// Start server in background
	serverDone := make(chan error, 1)
	go func() {
		t.Log("🚀 Starting server...")
		t.Logf("🕒 Server start time: %v", time.Since(startTime))
		// Signal server is ready to accept connections
		close(serverReady)
		err := serverInstance.Instance().StartAndWait(serverReader, serverWriter)
		if err != nil && err != context.DeadlineExceeded && err != io.ErrClosedPipe {
			t.Logf("❌ Server error: %v", err)
		}
		t.Log("🏁 Server finished")
		t.Logf("🕒 Server finish time: %v", time.Since(startTime))
		serverDone <- err
		close(serverDone)
	}()

	// Wait for server to be ready
	<-serverReady
	t.Log("🔌 Server ready")
	t.Logf("🕒 Server ready time: %v", time.Since(startTime))

	// Create client channel with options
	t.Log("📱 Creating client channels")
	clientChans := channel.LSP(clientReader, clientWriter)
	clientOpts := &jrpc2.ClientOptions{
		OnNotify: func(req *jrpc2.Request) {
			t.Logf("📨 Client received notification: %s", req.Method())
		},
	}
	t.Log("🔌 Creating client")
	client := jrpc2.NewClient(clientChans, clientOpts)
	defer func() {
		t.Log("🔌 Closing client...")
		t.Logf("🕒 Client close time: %v", time.Since(startTime))
		if err := client.Close(); err != nil && err != io.ErrClosedPipe {
			t.Logf("❌ Client close error: %v", err)
		}
	}()

	// Send initialize request
	t.Log("📤 Sending initialize request")
	t.Logf("🕒 Initialize request start time: %v", time.Since(startTime))
	var initResult protocol.InitializeResult
	err := client.CallResult(ctx, "initialize", &protocol.ParamInitialize{
		XInitializeParams: protocol.XInitializeParams{
			ProcessID: 1,
			RootURI:   protocol.DocumentURI("file:///workspace"),
			Capabilities: protocol.ClientCapabilities{
				TextDocument: protocol.TextDocumentClientCapabilities{},
			},
		},
	}, &initResult)

	// Wait for initialize to start and complete with timeout
	select {
	case <-initializeStarted:
		t.Log("🎯 Initialize handler started")
	case <-time.After(2 * time.Second):
		t.Error("❌ Initialize handler never started")
	}

	select {
	case <-initializeCompleted:
		t.Log("🎯 Initialize handler completed")
	case <-time.After(2 * time.Second):
		t.Error("❌ Initialize handler never completed")
	}

	t.Logf("⏱️ Initialize request took %v", time.Since(startTime))
	require.NoError(t, err, "initialize request should succeed")
	require.NotNil(t, initResult.Capabilities.TextDocumentSync, "server should return text document sync capability")

	// Wait a bit for the server to be ready for notifications
	time.Sleep(100 * time.Millisecond)

	// Send initialized notification
	t.Log("📤 Sending initialized notification")
	err = client.Notify(ctx, "initialized", &protocol.InitializedParams{})
	require.NoError(t, err, "initialized notification should succeed")

	// Wait for initialized notification with timeout
	select {
	case <-initializedReceived:
		t.Log("✅ Server received initialized notification")
	case <-time.After(2 * time.Second):
		t.Error("❌ Server never received initialized notification")
	}

	// Wait a bit before shutdown
	time.Sleep(100 * time.Millisecond)

	// Clean shutdown
	t.Log("📤 Sending shutdown request")
	_, err = client.Call(ctx, "shutdown", nil)
	require.NoError(t, err, "shutdown request should succeed")

	t.Log("📤 Sending exit notification")
	err = client.Notify(ctx, "exit", nil)
	require.NoError(t, err, "exit notification should succeed")

	// Close pipes to ensure clean shutdown
	t.Log("Closing pipes...")
	clientWriter.Close()
	serverWriter.Close()

	// Wait for server to finish with timeout
	select {
	case err := <-serverDone:
		if err != nil && err != context.DeadlineExceeded && err != io.ErrClosedPipe {
			t.Errorf("Server shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server shutdown timed out")
	}

	// Log RPC messages for debugging
	t.Log("RPC Messages:")
	for _, msg := range rpcTracker.GetMessages() {
		if msg.Request != nil {
			t.Logf("  -> %s: %s", msg.Method, msg.Request.ParamString())
		}
		if msg.Response != nil {
			t.Logf("  <- %s: %s", msg.Method, msg.Response.ResultString())
		}
	}
}

func TestCustomLSPBuffering(t *testing.T) {
	t.Parallel()

	t.Run("large_message_handling", func(t *testing.T) {
		t.Parallel()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create pipe for testing
		serverReader, clientWriter := io.Pipe()
		clientReader, serverWriter := io.Pipe()

		// Create LSP channel with custom buffering
		serverChans := channel.LSP(serverReader, serverWriter)
		clientChans := channel.LSP(clientReader, clientWriter)

		// Create large test data (7MB to test 8MB buffer)
		type LargeMessage struct {
			Data []byte `json:"data"`
		}
		largeData := make([]byte, 7*1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		testMsg := LargeMessage{Data: largeData}

		// Start server
		serverOpts := &jrpc2.ServerOptions{
			RPCLog: &testRPCLogger{t: t},
		}
		server := jrpc2.NewServer(handler.Map{
			"test": handler.New(func(ctx context.Context, params *LargeMessage) (*LargeMessage, error) {
				return params, nil
			}),
		}, serverOpts)
		go func() {
			err := server.Start(serverChans).Wait()
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("Server error: %v", err)
			}
		}()

		// Create client
		clientOpts := &jrpc2.ClientOptions{
			OnNotify: func(req *jrpc2.Request) {
				t.Logf("Client received notification: %s", req.Method())
			},
		}
		client := jrpc2.NewClient(clientChans, clientOpts)
		defer func() {
			t.Log("Closing client...")
			if err := client.Close(); err != nil {
				t.Logf("Client close error: %v", err)
			}
		}()

		// Send large message
		var response LargeMessage
		err := client.CallResult(ctx, "test", testMsg, &response)
		require.NoError(t, err, "large message should be handled")
		require.Equal(t, testMsg.Data, response.Data, "response should match sent data")
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		t.Parallel()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create pipe for testing
		serverReader, clientWriter := io.Pipe()
		clientReader, serverWriter := io.Pipe()

		// Create LSP channel with custom buffering
		serverChans := channel.LSP(serverReader, serverWriter)
		clientChans := channel.LSP(clientReader, clientWriter)

		// Define message type
		type EchoMessage struct {
			Message string `json:"message"`
		}

		// Start server
		serverOpts := &jrpc2.ServerOptions{
			RPCLog: &testRPCLogger{t: t},
		}
		server := jrpc2.NewServer(handler.Map{
			"echo": handler.New(func(ctx context.Context, params *EchoMessage) (*EchoMessage, error) {
				time.Sleep(10 * time.Millisecond) // Simulate processing
				return params, nil
			}),
		}, serverOpts)
		go func() {
			err := server.Start(serverChans).Wait()
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("Server error: %v", err)
			}
		}()

		// Create client
		clientOpts := &jrpc2.ClientOptions{
			OnNotify: func(req *jrpc2.Request) {
				t.Logf("Client received notification: %s", req.Method())
			},
		}
		client := jrpc2.NewClient(clientChans, clientOpts)
		defer func() {
			t.Log("Closing client...")
			if err := client.Close(); err != nil {
				t.Logf("Client close error: %v", err)
			}
		}()

		// Send concurrent requests
		const numRequests = 10
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				defer wg.Done()
				msg := &EchoMessage{Message: fmt.Sprintf("test-%d", id)}
				var response EchoMessage
				err := client.CallResult(ctx, "echo", msg, &response)
				require.NoError(t, err, "concurrent request should succeed")
				require.Equal(t, msg.Message, response.Message, "response should match request")
			}(i)
		}

		wg.Wait()
	})
}

func TestSimpleRequestResponse(t *testing.T) {
	t.Parallel()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create pipes for bidirectional communication
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	// Create mock server
	mockServer := mockery.NewMockServer_protocol(t)

	// Track when notification is received
	notificationReceived := make(chan struct{})
	serverStarted := make(chan struct{})
	serverReady := make(chan struct{})
	serverDone := make(chan error, 1)

	// Set up expectations
	mockServer.EXPECT().
		Initialize(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, params *protocol.ParamInitialize) {
			t.Log("🔍 Initialize handler called")
			if ctx.Err() != nil {
				t.Logf("❌ Initialize context error: %v", ctx.Err())
				return
			}
		}).
		Return(&protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{},
		}, nil).
		Once()

	mockServer.EXPECT().
		Initialized(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, params *protocol.InitializedParams) {
			t.Log("✅ Notification received")
			if ctx.Err() != nil {
				t.Logf("❌ Initialized context error: %v", ctx.Err())
				return
			}
			close(notificationReceived)
		}).
		Return(nil).
		Once()

	mockServer.EXPECT().
		Shutdown(mock.Anything).
		Run(func(ctx context.Context) {
			t.Log("🛑 Shutdown received")
			if ctx.Err() != nil {
				t.Logf("❌ Shutdown context error: %v", ctx.Err())
				return
			}
		}).
		Return(nil).
		Once()

	mockServer.EXPECT().
		Exit(mock.Anything).
		Run(func(ctx context.Context) {
			t.Log("🚪 Exit notification received")
		}).
		Return(nil).
		Once()

	// Create and start server with logging
	serverOpts := &jrpc2.ServerOptions{
		AllowPush: true,
		RPCLog:    &testRPCLogger{t: t},
	}
	serverInstance := protocol.NewServerInstance(ctx, mockServer, serverOpts)

	go func() {
		t.Log("🚀 Starting server...")
		close(serverStarted)
		err := serverInstance.Instance().StartAndWait(serverReader, serverWriter)
		if err != nil && err != io.ErrClosedPipe {
			t.Logf("❌ Server error: %v", err)
		}
		t.Log("🏁 Server finished")
		serverDone <- err
		close(serverDone)
	}()

	// Wait for server to start
	<-serverStarted
	t.Log("🔌 Server started")
	close(serverReady)

	// Create client with logging
	t.Log("📱 Creating client channels")
	clientChans := channel.LSP(clientReader, clientWriter)
	clientOpts := &jrpc2.ClientOptions{
		OnNotify: func(req *jrpc2.Request) {
			t.Logf("📨 Client received notification: %s", req.Method())
		},
	}
	t.Log("🔌 Creating client")
	client := jrpc2.NewClient(clientChans, clientOpts)
	defer func() {
		t.Log("🔌 Closing client...")
		if err := client.Close(); err != nil && err != io.ErrClosedPipe {
			t.Logf("❌ Client close error: %v", err)
		}
	}()

	// Wait for server to be ready
	<-serverReady
	t.Log("🔌 Server ready")

	// Send initialize request
	t.Log("📤 Sending initialize request")
	var initResult protocol.InitializeResult
	err := client.CallResult(ctx, "initialize", &protocol.ParamInitialize{}, &initResult)
	require.NoError(t, err, "initialize request should succeed")
	t.Log("✅ Initialize request succeeded")

	// Send initialized notification
	t.Log("📤 Sending initialized notification")
	err = client.Notify(ctx, "initialized", &protocol.InitializedParams{})
	require.NoError(t, err, "initialized notification should succeed")
	t.Log("✅ Initialized notification sent")

	// Wait for notification with timeout
	select {
	case <-notificationReceived:
		t.Log("✅ Test passed - notification was received")
	case <-time.After(5 * time.Second):
		t.Error("❌ Test failed - notification was not received")
	}

	// Clean shutdown
	t.Log("📤 Sending shutdown request")
	_, err = client.Call(ctx, "shutdown", nil)
	require.NoError(t, err, "shutdown request should succeed")

	t.Log("📤 Sending exit notification")
	err = client.Notify(ctx, "exit", nil)
	require.NoError(t, err, "exit notification should succeed")

	// Close pipes
	t.Log("🔌 Closing pipes...")
	clientWriter.Close()
	serverWriter.Close()

	// Wait for server to finish
	select {
	case err := <-serverDone:
		if err != nil && err != io.ErrClosedPipe {
			t.Errorf("❌ Server error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("❌ Server shutdown timed out")
	}
}

type testRPCLogger struct {
	t *testing.T
}

func (l *testRPCLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	l.t.Logf("Server received request: method=%s, params=%s", req.Method(), req.ParamString())
}

func (l *testRPCLogger) LogResponse(ctx context.Context, rsp *jrpc2.Response) {
	l.t.Logf("Server sent response: id=%s, result=%s", rsp.ID(), rsp.ResultString())
}
