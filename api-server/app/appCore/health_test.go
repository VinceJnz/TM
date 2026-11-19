package appCore

import (
	"api-server/v2/internal/testdb"
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadinessAndCleanupCancellation(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	c := &Config{Db: db}
	ctx, cancel := context.WithCancel(context.Background())
	c.startTokenCleanupJob(ctx)
	w := httptest.NewRecorder()
	c.Readiness(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 200 {
		t.Fatal("database not ready")
	}
	cancel()
	select {
	case <-c.cleanupDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup job did not stop")
	}
	db.Close()
	w = httptest.NewRecorder()
	c.Readiness(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 {
		t.Fatal("closed database reported ready")
	}
}
