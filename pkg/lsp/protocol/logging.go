package protocol

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/pkg/debug"
)

type MultiRPCLogger struct {
	mu      sync.Mutex
	loggers []jrpc2.RPCLogger
}

func (m *MultiRPCLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, logger := range m.loggers {
		logger.LogRequest(ctx, req)
	}
}

func (m *MultiRPCLogger) LogResponse(ctx context.Context, resp *jrpc2.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, logger := range m.loggers {
		logger.LogResponse(ctx, resp)
	}
}

func (m *MultiRPCLogger) AddLogger(logger jrpc2.RPCLogger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loggers = append(m.loggers, logger)
}

// LogContext is a key type for storing logger in context
type LogContext struct{}

// LogWritersContext is a key type for storing writers in context
type LogWritersContext struct{}

var (
	logContextKey      = LogContext{}
	logWritersKey      = LogWritersContext{}
	myLoggerId         = xid.New().String()
	defaultWriterMutex sync.RWMutex
	defaultWriters     []io.Writer
)

// ExtendedLogMessageParams extends LogMessageParams with additional fields
type ExtendedLogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
	// Raw          string                 `json:"raw"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
	Time         string                 `json:"time,omitempty"`
	Source       string                 `json:"source,omitempty"`
	IsDependency bool                   `json:"is_dependency,omitempty"`
	// Direction    string                 `json:"direction,omitempty"` // "incoming" or "outgoing"
	// Method       string                 `json:"method,omitempty"`    // JSONRPC method
}

func ApplyServerInstanceToZerolog(ctx context.Context, server Client) context.Context {
	// the server needs to not log to its console, instead it needs to log to the client
	writer := &logWriter{
		server: server,
		ctx:    ctx,
	}

	level := zerolog.Ctx(ctx).GetLevel()

	ctx = zerolog.New(writer).With().
		Str("id", myLoggerId).
		Str("lsp_role", "server").
		Logger().
		Level(level).
		Hook(debug.CustomTimeHook{WithColor: false}).
		Hook(debug.CustomCallerHook{WithColor: false}).
		WithContext(ctx)

	return ctx
}

func ApplyClientsCurrentContextToZerolog(ctx context.Context) context.Context {
	// the client needs to log to its console

	ctx = zerolog.Ctx(ctx).With().
		Str("id", myLoggerId).
		Str("lsp_role", "client").
		Logger().
		Hook(debug.CustomTimeHook{WithColor: false}).
		Hook(debug.CustomCallerHook{WithColor: false}).
		WithContext(ctx)

	return ctx
}

func ApplyRequestToZerolog(ctx context.Context, req *jrpc2.Request) context.Context {
	ctx = zerolog.Ctx(ctx).With().Str("rpc_method", req.Method()).Str("rpc_id", req.ID()).Logger().WithContext(ctx)
	return ctx
}

type logWriter struct {
	server Client
	mu     sync.Mutex
	ctx    context.Context
}

// var _ zerolog.ConsoleWriter = &logWriter{}

// Write implements io.Writer
func (w *logWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Parse the log entry
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		return len(p), nil
	}

	level := ParseMessageTypeFromZerolog(extractField(logEntry, "level", "info"))
	msg := extractField(logEntry, "message", "")
	id := extractField(logEntry, "id", "")
	time := extractField(logEntry, "time", "")
	source := extractField(logEntry, "caller", "")
	// direction := extractField(logEntry, "direction", "")
	// method := extractField(logEntry, "method", "")

	notification := &ExtendedLogMessageParams{
		Type:    level,
		Message: msg,
		// Raw:       string(p),
		Extra:        logEntry,
		Time:         time,
		Source:       source,
		IsDependency: id != myLoggerId,
		// Direction: direction,
		// Method:    method,
	}

	anyNotification := any(notification)

	if w.server != nil {
		err = w.server.Event(w.ctx, &anyNotification)
	}

	return len(p), err
}

// Helper function to extract and delete a field from the log entry
func extractField(entry map[string]interface{}, key, defaultValue string) string {
	if v, ok := entry[key].(string); ok {
		delete(entry, key)
		return v
	}
	return defaultValue
}

// ParseMessageTypeFromZerolog converts zerolog level to LSP MessageType
func ParseMessageTypeFromZerolog(level string) MessageType {
	switch level {
	case "error":
		return Error
	case "warn":
		return Warning
	case "info":
		return Info
	case "debug":
		return Debug
	default:
		return Log
	}
}

// // LSPLogger wraps zerolog.Logger with LSP-specific functionality
// type LSPLogger struct {
// 	logger zerolog.Logger
// 	writer *LSPWriter
// }

// // var _ jrpc2.RPCLogger = &LSPLogger{}

// // LSPWriter implements io.Writer to redirect logs to LSP
// type LSPWriter struct {
// 	mu   sync.Mutex
// 	conn *jsonrpc2.Conn
// 	ctx  context.Context
// }

// // GetContextWriters retrieves writers from context
// func GetContextWriters(ctx context.Context) []io.Writer {
// 	if writers, ok := ctx.Value(logWritersKey).([]io.Writer); ok && len(writers) > 0 {
// 		return writers
// 	}
// 	defaultWriterMutex.RLock()
// 	defer defaultWriterMutex.RUnlock()
// 	return defaultWriters
// }

// // AddDefaultWriter adds a writer to the default set
// func AddDefaultWriter(w io.Writer) {
// 	defaultWriterMutex.Lock()
// 	defer defaultWriterMutex.Unlock()
// 	defaultWriters = append(defaultWriters, w)
// }

// // RemoveDefaultWriter removes a writer from the default set
// func RemoveDefaultWriter(w io.Writer) {
// 	defaultWriterMutex.Lock()
// 	defer defaultWriterMutex.Unlock()
// 	for i, writer := range defaultWriters {
// 		if writer == w {
// 			defaultWriters = append(defaultWriters[:i], defaultWriters[i+1:]...)
// 			break
// 		}
// 	}
// }

// // WithWriter adds a writer to the context
// func WithWriter(ctx context.Context, w io.Writer) context.Context {
// 	existing := GetContextWriters(ctx)
// 	return context.WithValue(ctx, logWritersKey, append(existing, w))
// }

// // RemoveWriter removes a writer from the context
// func RemoveWriter(ctx context.Context, w io.Writer) context.Context {
// 	existing := GetContextWriters(ctx)
// 	var filtered []io.Writer
// 	for _, writer := range existing {
// 		if writer != w {
// 			filtered = append(filtered, writer)
// 		}
// 	}
// 	return context.WithValue(ctx, logWritersKey, filtered)
// }

// // NewLSPWriter creates a new LSPWriter instance
// func NewLSPWriter(ctx context.Context, conn *jsonrpc2.Conn) *LSPWriter {
// 	return &LSPWriter{
// 		conn: conn,
// 		ctx:  ctx,
// 	}
// }

// // NewLogger creates a new LSPLogger with LSP integration
// func NewLogger(ctx context.Context, conn *jsonrpc2.Conn) *LSPLogger {
// 	lspWriter := NewLSPWriter(ctx, conn)

// 	logger := zerolog.New(lspWriter).With().
// 		Logger().
// 		Hook(debug.CustomTimeHook{WithColor: false}).
// 		Hook(debug.CustomCallerHook{WithColor: false})

// 	return &LSPLogger{
// 		logger: logger,
// 		writer: lspWriter,
// 	}
// }

// // WithContext returns a new context with the logger attached
// func (l *LSPLogger) WithContext(ctx context.Context) context.Context {
// 	return context.WithValue(ctx, logContextKey, l)
// }

// // FromContext retrieves the logger from context
// func FromContext(ctx context.Context) *LSPLogger {
// 	if l, ok := ctx.Value(logContextKey).(*LSPLogger); ok {
// 		return l
// 	}
// 	return nil
// }

// // Helper functions for logging with proper context and caller info
// func (l *LSPLogger) Debug() *zerolog.Event {
// 	return l.logger.Debug().Str("id", myLoggerId).CallerSkipFrame(1)
// }

// func (l *LSPLogger) Info() *zerolog.Event {
// 	return l.logger.Info().Str("id", myLoggerId).CallerSkipFrame(1)
// }

// func (l *LSPLogger) Warn() *zerolog.Event {
// 	return l.logger.Warn().Str("id", myLoggerId).CallerSkipFrame(1)
// }

// func (l *LSPLogger) Error() *zerolog.Event {
// 	return l.logger.Error().Str("id", myLoggerId).CallerSkipFrame(1)
// }

// // LogClientRequest logs incoming client requests
// func (l *LSPLogger) LogClientRequest(method string, params interface{}) {
// 	l.Info().
// 		Str("direction", "incoming").
// 		Str("method", method).
// 		Interface("params", params).
// 		Msg("Client request received")
// }

// // LogServerResponse logs outgoing server responses
// func (l *LSPLogger) LogServerResponse(method string, result interface{}) {
// 	l.Info().
// 		Str("direction", "outgoing").
// 		Str("method", method).
// 		Interface("result", result).
// 		Msg("Server response sent")
// }

// // Convenience methods for formatted logging
// func (l *LSPLogger) Debugf(format string, args ...interface{}) {
// 	l.Debug().Msgf(format, args...)
// }

// func (l *LSPLogger) Infof(format string, args ...interface{}) {
// 	l.Info().Msgf(format, args...)
// }

// func (l *LSPLogger) Warnf(format string, args ...interface{}) {
// 	l.Warn().Msgf(format, args...)
// }

// func (l *LSPLogger) Errorf(format string, args ...interface{}) {
// 	l.Error().Msgf(format, args...)
// }
