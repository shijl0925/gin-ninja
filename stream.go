package ninja

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

const (
	streamKindSSE       = "sse"
	streamKindWebSocket = "websocket"
)

type streamConfig struct {
	kind string
}

// SSEEvent represents a single server-sent event message.
type SSEEvent struct {
	ID    string
	Event string
	Data  any
	Retry time.Duration
}

// SSEStream writes compliant server-sent event frames.
type SSEStream struct {
	c    *gin.Context
	sent bool
}

// Send emits one SSE frame and flushes it to the client.
func (s *SSEStream) Send(event SSEEvent) error {
	if s == nil || s.c == nil {
		return InternalError()
	}
	writer := s.c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return InternalError()
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(writer, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	if event.Event != "" {
		if _, err := fmt.Fprintf(writer, "event: %s\n", event.Event); err != nil {
			return err
		}
	}
	if event.Retry > 0 {
		if _, err := fmt.Fprintf(writer, "retry: %d\n", event.Retry.Milliseconds()); err != nil {
			return err
		}
	}
	for _, line := range strings.Split(sseData(event.Data), "\n") {
		if _, err := fmt.Fprintf(writer, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}
	flusher.Flush()
	s.sent = true
	return nil
}

// WebSocketConn is a small convenience wrapper over x/net/websocket.
type WebSocketConn struct {
	*websocket.Conn
}

func (c *WebSocketConn) SendJSON(v any) error {
	return websocket.JSON.Send(c.Conn, v)
}

func (c *WebSocketConn) ReceiveJSON(v any) error {
	return websocket.JSON.Receive(c.Conn, v)
}

func (c *WebSocketConn) SendText(value string) error {
	return websocket.Message.Send(c.Conn, value)
}

func (c *WebSocketConn) ReceiveText() (string, error) {
	var value string
	err := websocket.Message.Receive(c.Conn, &value)
	return value, err
}

// SSE registers a GET endpoint that streams server-sent events.
func SSE[TIn any](r *Router, path string, handler func(*Context, *TIn, *SSEStream) error, opts ...OperationOption) {
	op := newSSEOperation(path, handler, r.tags)
	op.security = cloneSecurityRequirements(r.security)
	op.tagDescriptions = cloneStringMap(r.tagDescriptions)
	for _, opt := range opts {
		opt(op)
	}
	op.finalize()
	r.operations = append(r.operations, op)
}

// WebSocket registers a GET endpoint that upgrades the connection to WebSocket.
func WebSocket[TIn any](r *Router, path string, handler func(*Context, *TIn, *WebSocketConn) error, opts ...OperationOption) {
	op := newWebSocketOperation(path, handler, r.tags)
	op.security = cloneSecurityRequirements(r.security)
	op.tagDescriptions = cloneStringMap(r.tagDescriptions)
	for _, opt := range opts {
		opt(op)
	}
	op.finalize()
	r.operations = append(r.operations, op)
}

func newSSEOperation[TIn any](path string, handler func(*Context, *TIn, *SSEStream) error, defaultTags []string) *operation {
	var zeroIn TIn
	inputType := reflect.TypeOf(zeroIn)

	op := &operation{
		method:        http.MethodGet,
		path:          path,
		inputType:     inputType,
		tags:          append([]string(nil), defaultTags...),
		successStatus: http.StatusOK,
		stream:        &streamConfig{kind: streamKindSSE},
	}

	op.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)
		input := new(TIn)
		if err := bindInput(c, http.MethodGet, input); err != nil {
			writeError(c, err)
			return
		}

		c.Header("Content-Type", "text/event-stream")
		if c.Writer.Header().Get("Cache-Control") == "" {
			c.Header("Cache-Control", "no-cache")
		}
		c.Header("Connection", "keep-alive")

		stream := &SSEStream{c: c}
		if err := handler(ctx, input, stream); err != nil && !stream.sent && !c.Writer.Written() {
			writeError(c, err)
		}
	}

	return op
}

func newWebSocketOperation[TIn any](path string, handler func(*Context, *TIn, *WebSocketConn) error, defaultTags []string) *operation {
	var zeroIn TIn
	inputType := reflect.TypeOf(zeroIn)

	op := &operation{
		method:        http.MethodGet,
		path:          path,
		inputType:     inputType,
		tags:          append([]string(nil), defaultTags...),
		successStatus: http.StatusSwitchingProtocols,
		stream:        &streamConfig{kind: streamKindWebSocket},
	}

	op.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)
		input := new(TIn)
		if err := bindInput(c, http.MethodGet, input); err != nil {
			writeError(c, err)
			return
		}

		websocket.Handler(func(conn *websocket.Conn) {
			defer conn.Close()
			if err := handler(ctx, input, &WebSocketConn{Conn: conn}); err != nil {
				// Record the failure for Gin middleware/logging without echoing raw
				// internal error details back to the WebSocket client.
				_ = ctx.Error(err)
			}
		}).ServeHTTP(c.Writer, c.Request)
	}

	return op
}

func sseData(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	case time.Duration:
		return strconv.FormatInt(v.Milliseconds(), 10)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}
