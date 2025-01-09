package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/debug"
)

// var _ zerolog.ConsoleWriter = &LSPWriter{}

// LSPWriter implements io.Writer to redirect logs to LSP
type LSPWriter struct {
	mu   sync.Mutex
	conn *jsonrpc2.Conn
	ctx  context.Context
}

func (s *Server) ApplyLSPWriter(ctx context.Context, conn *jsonrpc2.Conn) context.Context {
	lspWriter := NewLSPWriter(ctx, conn)

	ctx = zerolog.New(lspWriter).With().
		Logger().
		Hook(debug.CustomTimeHook{WithColor: false}).
		Hook(debug.CustomCallerHook{WithColor: false}).
		WithContext(ctx)

	return ctx
}

var myLoggerId = xid.New().String()

func (s *Server) debugf(ctx context.Context, format string, args ...interface{}) {
	if !s.debug {
		return
	}

	msg := fmt.Sprintf(format, args...)

	zerolog.Ctx(ctx).Debug().
		Str("id", myLoggerId).
		CallerSkipFrame(1).
		// Str("component", "lsp").
		// Bool("debug", s.debug).
		Msg(msg)

}

func NewLSPWriter(ctx context.Context, conn *jsonrpc2.Conn) *LSPWriter {
	return &LSPWriter{
		conn: conn,
		ctx:  ctx,
	}
}

func (w *LSPWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Parse the log entry to extract level and message
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		return len(p), nil // Skip malformed entries
	}

	// Convert zerolog level to LSP message type
	var level MessageType = Unknown
	if l, ok := logEntry["level"].(string); ok {
		level = ParseMessageTypeFromZerolog(l)

	}

	// Format the message
	msg := ""
	if m, ok := logEntry["message"].(string); ok {
		msg = m
		delete(logEntry, "message")
	}

	id := ""
	if i, ok := logEntry["id"].(string); ok {
		id = i
		delete(logEntry, "id") // just for checking to see if rthis is the right logger, no need to send to user
	}

	time := ""
	if t, ok := logEntry["time"].(string); ok {
		time = t
		delete(logEntry, "time")
	}

	source := ""
	if s, ok := logEntry["caller"].(string); ok {
		source = s
		delete(logEntry, "caller")
	}

	var notification LogMessageParams
	if id == myLoggerId {
		delete(logEntry, "level")
		notification = LogMessageParams{
			Type:    level,
			Message: msg,
			Raw:     string(p),
			Extra:   logEntry,
			Time:    time,
			Source:  source,
		}
	} else {
		notification = LogMessageParams{
			Type:    Dependency,
			Message: msg,
			Raw:     string(p),
			Extra:   logEntry,
			Time:    time,
			Source:  source,
		}
	}

	// Create LSP notification
	err = w.conn.Notify(w.ctx, "window/logMessage", notification)
	return len(p), err
}

// // SetupGlobalLogger configures zerolog to route all output through the LSP connection
// func SetupGlobalLogger(ctx context.Context, conn *jsonrpc2.Conn, debug bool) context.Context {
// 	// Create LSP writer
// 	lspWriter := NewLSPWriter(ctx, conn)

// 	// Create the logger
// 	logger := zerolog.New(lspWriter).With().
// 		Str("component", "lsp-server").
// 		Bool("debug", debug).
// 		Timestamp().
// 		Caller().
// 		Logger()

// 	// Set log level based on debug flag
// 	if debug {
// 		zerolog.SetGlobalLevel(zerolog.DebugLevel)
// 	} else {
// 		zerolog.SetGlobalLevel(zerolog.InfoLevel)
// 	}

// 	// Log takeover message
// 	logger.Info().Msg("LSP server taking over logging output")

// 	return logger.WithContext(ctx)
// }
