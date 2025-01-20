package protocol

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/creachadair/jrpc2"
)

type rpcTrackerContextKey struct{}

// RPCMessage represents a single RPC message
type RPCMessage struct {
	Method   string
	Request  *jrpc2.Request  // Original request if available
	Response *jrpc2.Response // Response if available
	Time     time.Time       // When the message was tracked
}

var _ jrpc2.RPCLogger = (*RPCTracker)(nil)

func (t *RPCTracker) LogRequest(ctx context.Context, req *jrpc2.Request) {
	t.AddKnownMethod(req.ID(), req.Method())
	t.Track(RPCMessage{
		Method:  req.Method(),
		Request: req,
		Time:    time.Now(),
	})
}

func (t *RPCTracker) LogResponse(ctx context.Context, resp *jrpc2.Response) {
	t.Track(RPCMessage{
		Method:   t.GetKnownMethod(resp.ID()),
		Response: resp,
		Time:     time.Now(),
	})
}

func (t *RPCTracker) MessagesSince(since time.Time) []RPCMessage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	copy := append([]RPCMessage{}, t.messages...)
	return slices.DeleteFunc(copy, func(msg RPCMessage) bool {
		return msg.Time.Before(since)
	})
}

func (t *RPCTracker) MessagesSinceLike(since time.Time, predicate func(RPCMessage) bool) []RPCMessage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	copy := append([]RPCMessage{}, t.messages...)
	return slices.DeleteFunc(copy, func(msg RPCMessage) bool {
		return msg.Time.Before(since) || !predicate(msg)
	})
}

func (t *RPCTracker) WaitOnWit(since time.Time, predicate func(RPCMessage) bool) []RPCMessage {
	return t.MessagesSinceLike(since, predicate)
}

// RPCTracker tracks incoming and outgoing RPC messages for testing purposes
type RPCTracker struct {
	mu sync.RWMutex

	messages     []RPCMessage
	subs         map[chan<- RPCMessage]struct{}
	knownMethods map[string]string
}

func (t *RPCTracker) AddKnownMethod(method string, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.knownMethods[method] = name
}

func (t *RPCTracker) GetKnownMethod(method string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.knownMethods[method]
}

// NewRPCTracker creates a new RPCTracker
func NewRPCTracker() *RPCTracker {
	return &RPCTracker{
		messages:     make([]RPCMessage, 0),
		subs:         make(map[chan<- RPCMessage]struct{}),
		knownMethods: make(map[string]string),
	}
}

// Subscribe creates a new subscription for messages
// The returned function should be called to unsubscribe
func (t *RPCTracker) Subscribe(bufSize int) (<-chan RPCMessage, func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan RPCMessage, bufSize)
	t.subs[ch] = struct{}{}

	return ch, func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.subs, ch)
		close(ch)
	}
}

// Track adds a message to the tracker and notifies subscribers
func (t *RPCTracker) Track(msg RPCMessage) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	msg.Time = time.Now()
	t.messages = append(t.messages, msg)

	// Notify subscribers
	for ch := range t.subs {
		select {
		case ch <- msg:
		default:
			// Skip if channel is full
		}
	}
}

// GetMessages returns all tracked messages
func (t *RPCTracker) GetMessages() []RPCMessage {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]RPCMessage{}, t.messages...)
}

// Clear clears all tracked messages
func (t *RPCTracker) Clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages = nil
}

// WaitForMessage waits for a message matching the given predicate
// Returns nil if timeout is reached
func (t *RPCTracker) WaitForMessages(since time.Time, count int, timeout time.Duration, predicate func(RPCMessage) bool) ([]RPCMessage, bool) {

	// First check existing messages
	result := t.MessagesSinceLike(since, predicate)

	if len(result) >= count {
		return result, true
	}

	ch, unsub := t.Subscribe(0)
	defer unsub()

	// Then wait for new messages
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case msg := <-ch:
			if predicate(msg) {
				result = append(result, msg)
			}
			if len(result) >= count {
				return result, true
			}
		case <-timer.C:
			return result, len(result) == count
		}
	}
}

// Helper functions for context
func GetRPCTrackerFromContext(ctx context.Context) *RPCTracker {
	if tracker, ok := ctx.Value(rpcTrackerContextKey{}).(*RPCTracker); ok {
		return tracker
	}
	return nil
}

func ContextWithRPCTracker(ctx context.Context, tracker *RPCTracker) context.Context {
	return context.WithValue(ctx, rpcTrackerContextKey{}, tracker)
}

// TrackRPC is a helper function to track RPC messages from context
// func TrackRPC(ctx context.Context, dir MessageDir, method string, req *jrpc2.Request, resp *jrpc2.Response, params, result interface{}, err error) {
// 	if tracker := GetRPCTrackerFromContext(ctx); tracker != nil {
// 		tracker.Track(RPCMessage{
// 			Method:    method,
// 			Request:   req,
// 			Response:  resp,
// 			Params:    params,
// 			Result:    result,
// 			Error:     err,
// 			Direction: dir,
// 		})
// 	}
// }
