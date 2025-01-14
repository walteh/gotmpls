package protocol

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

// type serverDispatcher struct {
// 	sender *jsonrpc2.Conn
// }

// // NewServerDispatcher creates a new dispatcher that will send requests to the client
// func NewServerDispatcher(sender *jsonrpc2.Conn) Server {
// 	return &serverDispatcher{sender: sender}
// }

// type clientDispatcher struct {
// 	sender *jsonrpc2.Conn
// }

// // NewClientDispatcher creates a new dispatcher that will send requests to the server
// func NewClientDispatcher(sender *jsonrpc2.Conn) Client {
// 	return &clientDispatcher{sender: sender}
// }

// func UnmarshalJSON(data *json.RawMessage, v any) error {
// 	if data == nil {
// 		return nil
// 	}
// 	return json.Unmarshal([]byte(*data), v)
// }

func reply_fwd(ctx context.Context, conn *jsonrpc2.Conn, id *jsonrpc2.Request, result any, err error) error {
	if err != nil {
		return conn.ReplyWithError(ctx, id.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: err.Error(),
		})
	}
	return conn.Reply(ctx, id.ID, result)
}

// func sendParseError(ctx context.Context, conn *jsonrpc2.Conn, id jsonrpc2.Request, err error) error {
// 	return conn.ReplyWithError(ctx, id.ID, &jsonrpc2.Error{
// 		Code:    jsonrpc2.CodeInternalError,
// 		Message: err.Error(),
// 	})
// }

// func recoverHandlerPanic(method string) {
// 	if r := recover(); r != nil {
// 		log.Printf("panic in %s: %v", method, r)
// 	}
// }
