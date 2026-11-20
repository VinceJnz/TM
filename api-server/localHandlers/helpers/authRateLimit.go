package helpers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Counters live in PostgreSQL, so restarts and multiple API replicas do not
// reset the budget. Never trust caller-controlled forwarding headers for IPs.
func AuthRateLimit(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimRight(r.URL.Path, "/")
			if !strings.Contains(path, "/auth/") || strings.HasSuffix(path, "/status") || strings.HasSuffix(path, "/menuUser") || strings.Contains(path, "/content/") {
				next.ServeHTTP(w, r)
				return
			}
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			// Sending email and creating flows have tighter budgets than verification.
			sending := strings.HasSuffix(path, "/register") || strings.HasSuffix(path, "/login") || strings.HasSuffix(path, "/login-password") || strings.HasSuffix(path, "/requestToken") || strings.HasSuffix(path, "/forgot-password")
			limit := 20
			category := "verify"
			if sending {
				limit = 5
				category = "send"
			}
			keys := []struct {
				value          string
				seconds, limit int
			}{{"ip:" + category + ":" + ip, 60, limit}}
			if sending {
				keys = append(keys, struct {
					value          string
					seconds, limit int
				}{"send-ip:" + ip, 3600, 15})
			}
			if r.Body != nil && r.Method == http.MethodPost {
				data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
				if err != nil {
					WriteError(w, 400, "invalid_request", "invalid request body")
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(data))
				var identity struct {
					Username string `json:"username"`
					Email    string `json:"email"`
				}
				if json.Unmarshal(data, &identity) == nil && sending {
					account := strings.TrimSpace(identity.Email)
					if account == "" {
						account = strings.TrimSpace(identity.Username)
					}
					if account != "" {
						keys = append(keys, struct {
							value          string
							seconds, limit int
						}{"account:" + strings.ToLower(account), 3600, 10})
					}
				}
			}
			for _, key := range keys {
				digest := sha256.Sum256([]byte(key.value + ":" + strconv.Itoa(key.seconds)))
				var attempts int
				err := db.GetContext(r.Context(), &attempts, `WITH expired AS (DELETE FROM st_auth_rate_limits WHERE window_start<CURRENT_TIMESTAMP-INTERVAL '2 hours') INSERT INTO st_auth_rate_limits(key_hash,window_start,attempts) VALUES($1,to_timestamp(floor(extract(epoch FROM CURRENT_TIMESTAMP)/$2::numeric)*$2::numeric),1) ON CONFLICT(key_hash,window_start) DO UPDATE SET attempts=st_auth_rate_limits.attempts+1 RETURNING attempts`, hex.EncodeToString(digest[:]), key.seconds)
				if err != nil {
					WriteError(w, 503, "auth_unavailable", "authentication temporarily unavailable")
					return
				}
				if attempts > key.limit {
					w.Header().Set("Retry-After", strconv.Itoa(key.seconds))
					WriteError(w, 429, "rate_limited", "too many authentication requests")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
