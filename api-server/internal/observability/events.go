// Package observability emits bounded, structured metadata, never auth payloads.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"time"
)

type requestKey struct{}

func RequestID(ctx context.Context) string { id, _ := ctx.Value(requestKey{}).(string); return id }
func JobContext(ctx context.Context) context.Context {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ctx
	}
	return context.WithValue(ctx, requestKey{}, hex.EncodeToString(value[:]))
}
func Write(output io.Writer, ctx context.Context, event string, fields map[string]any) error {
	record := map[string]any{"time": time.Now().UTC(), "event": event, "request_id": RequestID(ctx)}
	allowed := map[string]bool{"booking_id": true, "attempt_id": true, "refund_id": true, "operation": true, "outcome": true, "resource": true, "duration_ms": true, "status": true, "method": true, "path": true, "user_id": true, "error_kind": true}
	for key, value := range fields {
		if allowed[key] {
			record[key] = value
		}
	}
	return json.NewEncoder(output).Encode(record)
}
func Event(ctx context.Context, event string, fields map[string]any) {
	_ = Write(os.Stderr, ctx, event, fields)
}
