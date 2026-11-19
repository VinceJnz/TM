package appCore

import (
	"api-server/v2/migrations"
	"context"
	"net/http"
	"time"
)

func (c *Config) Liveness(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
func (c *Config) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if c.Db == nil || c.Db.PingContext(ctx) != nil || migrations.Check(ctx, c.Db) != nil {
		http.Error(w, "database unavailable", 503)
		return
	}
	w.WriteHeader(http.StatusOK)
}
