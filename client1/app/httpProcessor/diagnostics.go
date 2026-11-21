package httpProcessor

import (
	"log"
	"net/http"
	"strings"
)

func safeRequestID(res *http.Response) string {
	id := res.Header.Get("X-Request-ID")
	if len(id) != 24 || strings.Trim(id, "0123456789abcdef") != "" {
		return "unavailable"
	}
	return id
}

// Diagnostics never include query strings, headers, bodies or decoded values.
func logResponse(req *http.Request, res *http.Response, kind string) {
	log.Printf("http method=%s status=%d request_id=%s category=%s", req.Method, res.StatusCode, safeRequestID(res), kind)
}
