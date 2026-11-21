package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestEventsCorrelateWithoutLoggingSecrets(t *testing.T) {
	ctx := JobContext(context.Background())
	var buffer bytes.Buffer
	if err := Write(&buffer, ctx, "checkout_recovery", map[string]any{"attempt_id": "synthetic", "outcome": "deferred", "password": "private-password", "token": "private-token", "cookie": "private-cookie"}); err != nil {
		t.Fatal(err)
	}
	if RequestID(ctx) == "" || strings.Contains(buffer.String(), "private-") {
		t.Fatal(buffer.String())
	}
	var record map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &record); err != nil || record["request_id"] != RequestID(ctx) {
		t.Fatal(record, err)
	}
}
