package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/creachadair/jrpc2"
	"github.com/rs/zerolog"
)

type rpcTestLogger struct {
	logger          zerolog.TestingLog
	rewrites        map[string]string
	enableTelemetry bool
	enableRequests  bool
	enableResponses bool
	isHuman         bool
}

func DebugAll() bool {
	return os.Getenv("DEBUG_LSP_ALL") == "1" || os.Getenv("DEBUG") == "1"
}

func DebugIsHuman() bool {
	return os.Getenv("HUMAN") == "1"
}

func NewTestLogger(t zerolog.TestingLog, rewrites map[string]string) jrpc2.RPCLogger {
	if rewrites == nil {
		rewrites = make(map[string]string)
	}

	lgr := &rpcTestLogger{logger: t, rewrites: rewrites,
		isHuman:         DebugIsHuman(),
		enableTelemetry: DebugAll(),
		enableRequests:  os.Getenv("DEBUG_LSP_BIG_REQUESTS") == "1",
		enableResponses: os.Getenv("DEBUG_LSP_BIG_RESPONSES") == "1",
	}

	for k, v := range rewrites {
		lgr.logger.Logf("FYI: '%s' will be rewritten to '%s' in logs for this test", k, v)
	}

	if !lgr.enableRequests {
		lgr.logger.Logf("FYI: big client request logs will be suppressed. Set DEBUG_LSP_BIG_REQUESTS=1 to see them")
	}

	if !lgr.enableResponses {
		lgr.logger.Logf("FYI: big server response logs will be suppressed. Set DEBUG_LSP_BIG_RESPONSES=1 to see them")
	}

	if !lgr.enableTelemetry {
		lgr.logger.Logf("FYI: internal logs (ex. a log made with zerolog.Ctx(ctx)) will be suppressed. Set DEBUG=1 to see them")
	}

	if !lgr.isHuman {
		lgr.logger.Logf("FYI: json logs will not be formatted for humans - set HUMAN=1 to see more readable json")
	} else {
		lgr.logger.Logf("FYI: json logs will be formatted for humans - warning: this might overflow ai context windows: set HUMAN=0 to make them more compact.")

	}

	if !DebugAll() {
		lgr.logger.Logf("FYI: set DEBUG_LSP_ALL=1 or DEBUG=1 to see all logs")
	}

	return lgr
}

var _ jrpc2.RPCLogger = &rpcTestLogger{}

type fancyRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type CallbackRPCLogger interface {
	LogCallbackRequestRaw(ctx context.Context, method string, params any)
	LogCallbackRequest(ctx context.Context, req *jrpc2.Request)
	LogCallbackResponse(ctx context.Context, res *jrpc2.Response)
}

func (l *rpcTestLogger) namedResponseLog(ctx context.Context, name string, res *jrpc2.Response) {

	lenRes := len(res.ResultString())
	var v any
	if lenRes > maxResultLength && !l.enableResponses {
		v = fmt.Sprintf("suppressed %d chars: set DEBUG_LSP_BIG_RESPONSES=1 to see", lenRes)
	} else {
		err := res.UnmarshalResult(&v)
		if err != nil {
			l.logger.Logf("lsp server response:%s", l.formatJSON(res))
			return
		}
	}

	parsed := fancyResponse{
		ID:     res.ID(),
		Result: v,
		Error:  res.Error(),
	}

	if parsed.ID == "" {
		parsed.ID = "notification"
	}

	l.logger.Logf("%s response:%s", name, l.formatJSON(parsed))
}

func (l *rpcTestLogger) namedRequestLog(ctx context.Context, name string, req *jrpc2.Request) {
	if req.Method() == "telemetry/event" && !l.enableTelemetry {
		return
	}

	lenReq := len(req.ParamString())
	var v any
	if lenReq > maxResultLength && !l.enableRequests {
		v = fmt.Sprintf("suppressed %d chars: set DEBUG_LSP_BIG_REQUESTS=1 to see", lenReq)
	} else {
		err := req.UnmarshalParams(&v)
		if err != nil {
			l.logger.Logf("lsp client request:%s", l.formatJSON(req))
			return
		}
	}
	parsed := fancyRequest{
		ID:     req.ID(),
		Method: req.Method(),
		Params: v,
	}

	if parsed.ID == "" {
		parsed.ID = "notification"
	}

	l.logger.Logf("lsp %s request:%s", name, l.formatJSON(parsed))
}

func (l *rpcTestLogger) formatJSON(s any) string {
	prefix := " "
	suffix := ""
	if l.isHuman {
		prefix = "\n\n"
		suffix = "\n\n"
	}
	buf := bytes.NewBuffer(nil)
	marshaller := json.NewEncoder(buf)
	if l.isHuman {
		marshaller.SetIndent("", "\t")
	}
	err := marshaller.Encode(s)
	if err != nil {
		return prefix + fmt.Sprintf("%+v", s) + suffix
	}

	str := buf.String()

	for k, v := range l.rewrites {
		str = strings.ReplaceAll(str, k, v)
	}

	return prefix + str + suffix
}

type fancyResponse struct {
	ID     string `json:"id"`
	Result any    `json:"result"`
	Error  any    `json:"error"`
}

const maxResultLength = 240

func (l *rpcTestLogger) LogResponse(ctx context.Context, res *jrpc2.Response) {
	l.namedResponseLog(ctx, "server", res)
}

func (l *rpcTestLogger) LogCallbackResponse(ctx context.Context, res *jrpc2.Response) {
	l.namedResponseLog(ctx, "client (callback)", res)
}

func (l *rpcTestLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	l.namedRequestLog(ctx, "client", req)
}

func (l *rpcTestLogger) LogCallbackRequest(ctx context.Context, req *jrpc2.Request) {
	l.namedRequestLog(ctx, "server (callback)", req)
}

func (l *rpcTestLogger) LogCallbackRequestRaw(ctx context.Context, method string, params any) {
	raw, err := json.Marshal(params)
	if err != nil {
		l.logger.Logf("failed to marshal params: %v", err)
		return
	}
	parsed := &jrpc2.ParsedRequest{
		ID:     "unknown",
		Method: method,
		Params: raw,
		Error:  nil,
	}

	l.namedRequestLog(ctx, "server (callback)", parsed.ToRequest())
}
