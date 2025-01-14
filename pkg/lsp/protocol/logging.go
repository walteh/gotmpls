package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/walteh/go-tmpl-typer/pkg/debug"
)

// LogContext is a key type for storing logger in context
type LogContext struct{}

var (
	logContextKey = LogContext{}
	myLoggerId    = xid.New().String()
)

// ExtendedLogMessageParams extends LogMessageParams with additional fields
type ExtendedLogMessageParams struct {
	Type         MessageType            `json:"type"`    // We don't embed to have full control
	Message      string                 `json:"message"` // of our extended format
	Raw          string                 `json:"raw"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
	Time         string                 `json:"time,omitempty"`
	Source       string                 `json:"source,omitempty"`
	IsDependency bool                   `json:"isDependency,omitempty"`
}

// LSPLogger wraps zerolog.Logger with LSP-specific functionality
type LSPLogger struct {
	logger zerolog.Logger
	writer *LSPWriter
}

// LSPWriter implements io.Writer to redirect logs to LSP
type LSPWriter struct {
	mu      sync.Mutex
	conn    *jsonrpc2.Conn
	ctx     context.Context
	writers []io.Writer // Additional writers for testing/debugging
}

// NewLSPWriter creates a new LSPWriter instance
func NewLSPWriter(ctx context.Context, conn *jsonrpc2.Conn) *LSPWriter {
	return &LSPWriter{
		conn:    conn,
		ctx:     ctx,
		writers: make([]io.Writer, 0),
	}
}

// AddWriter adds an additional writer for logs (useful for testing)
func (w *LSPWriter) AddWriter(writer io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writers = append(w.writers, writer)
}

// RemoveWriter removes a writer from the list
func (w *LSPWriter) RemoveWriter(writer io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, wr := range w.writers {
		if wr == writer {
			w.writers = append(w.writers[:i], w.writers[i+1:]...)
			break
		}
	}
}

// Write implements io.Writer
func (w *LSPWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Write to additional writers first
	for _, writer := range w.writers {
		if _, err := writer.Write(p); err != nil {
			// Just log the error, don't fail the whole write
			fmt.Printf("Error writing to additional writer: %v\n", err)
		}
	}

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

	// Extract fields
	msg := extractField(logEntry, "message", "")
	id := extractField(logEntry, "id", "")
	time := extractField(logEntry, "time", "")
	source := extractField(logEntry, "caller", "")

	// Create notification
	notification := ExtendedLogMessageParams{
		Type:    level,
		Message: msg,
		Raw:     string(p),
		Extra:   logEntry,
		Time:    time,
		Source:  source,
	}

	if id != myLoggerId {
		notification.Type = Dependency
	}

	// Send LSP notification
	err = w.conn.Notify(w.ctx, "window/logMessage", notification)
	return len(p), err
}

// NewLogger creates a new LSPLogger with LSP integration
func NewLogger(ctx context.Context, conn *jsonrpc2.Conn, extraWriters ...io.Writer) *LSPLogger {
	lspWriter := NewLSPWriter(ctx, conn)
	for _, w := range extraWriters {
		lspWriter.AddWriter(w)
	}

	logger := zerolog.New(lspWriter).With().
		Logger().
		Hook(debug.CustomTimeHook{WithColor: false}).
		Hook(debug.CustomCallerHook{WithColor: false})

	return &LSPLogger{
		logger: logger,
		writer: lspWriter,
	}
}

// WithContext returns a new context with the logger attached
func (l *LSPLogger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, logContextKey, l)
}

// FromContext retrieves the logger from context
func FromContext(ctx context.Context) *LSPLogger {
	if l, ok := ctx.Value(logContextKey).(*LSPLogger); ok {
		return l
	}
	return nil
}

// AddWriter adds a writer to the underlying LSPWriter
func (l *LSPLogger) AddWriter(w io.Writer) {
	l.writer.AddWriter(w)
}

// RemoveWriter removes a writer from the underlying LSPWriter
func (l *LSPLogger) RemoveWriter(w io.Writer) {
	l.writer.RemoveWriter(w)
}

// Helper functions for logging with proper context and caller info
func (l *LSPLogger) Debug() *zerolog.Event {
	return l.logger.Debug().Str("id", myLoggerId).CallerSkipFrame(1)
}

func (l *LSPLogger) Info() *zerolog.Event {
	return l.logger.Info().Str("id", myLoggerId).CallerSkipFrame(1)
}

func (l *LSPLogger) Warn() *zerolog.Event {
	return l.logger.Warn().Str("id", myLoggerId).CallerSkipFrame(1)
}

func (l *LSPLogger) Error() *zerolog.Event {
	return l.logger.Error().Str("id", myLoggerId).CallerSkipFrame(1)
}

// Convenience methods for formatted logging
func (l *LSPLogger) Debugf(format string, args ...interface{}) {
	l.Debug().Msgf(format, args...)
}

func (l *LSPLogger) Infof(format string, args ...interface{}) {
	l.Info().Msgf(format, args...)
}

func (l *LSPLogger) Warnf(format string, args ...interface{}) {
	l.Warn().Msgf(format, args...)
}

func (l *LSPLogger) Errorf(format string, args ...interface{}) {
	l.Error().Msgf(format, args...)
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
