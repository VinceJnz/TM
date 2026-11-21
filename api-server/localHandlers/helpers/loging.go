package helpers

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/observability"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"
)

type HTTPInfo struct {
	RequestID     string          `json:"RequestID"`
	LogTag        string          `json:"LogTag,omitempty"`
	Duration      time.Duration   `json:"Duration"` // how long did it take to
	URL           string          `json:"URL"`
	Host          string          `json:"Host"`
	Cookie        string          `json:"Cookie"`
	Method        string          `json:"Method"` // GET etc.
	RequestURI    string          `json:"RequestURI"`
	Referer       string          `json:"Referer"`
	Protocol      string          `json:"Protocol"`
	RemoteAddress string          `json:"RemoteAddress"`
	Size          int64           `json:"Size"`       // number of bytes of the response sent
	StatusCode    int             `json:"StatusCode"` // response code e.g. 200, 404, etc.
	UserAgent     string          `json:"UserAgent"`
	UserID        int64           `json:"UserID"`
	TLSProtocol   string          `json:"TLSProtocol"`
	Session       *models.Session `json:"Session,omitempty"`
	ResponseBody  string          `json:"ResponseBody,omitempty"`
}

// LogRequest This is a MiddleWare handler that logs all request to the console
func LogRequest(next http.Handler, sessionIDKey appCore.ContextKey, logTag string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := &HTTPInfo{
			RequestID:     observability.RequestID(r.Context()),
			LogTag:        logTag,
			Duration:      0,
			URL:           r.URL.EscapedPath(),
			Host:          r.Host,
			Cookie:        "",
			Method:        r.Method,
			RequestURI:    r.URL.EscapedPath(),
			Referer:       logSafeURL(r.Referer()),
			Protocol:      "",
			RemoteAddress: r.RemoteAddr,
			Size:          0,
			StatusCode:    0,
			UserAgent:     r.UserAgent(), // r.Header.Get("User-Agent"),
			UserID:        0,
			TLSProtocol:   "",
		}
		start := time.Now()

		// Create a response writer that captures the status code and body
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		// Log metadata only; never retain or log response bodies.
		// SECURITY: Session cookie should not be logged for security reasons
		// User identification is available via info.UserID from context
		info.StatusCode = lrw.statusCode
		info.Protocol = r.Proto
		info.Size = lrw.bytesWritten
		if r.TLS != nil {
			info.TLSProtocol = r.TLS.NegotiatedProtocol
		}
		info.Duration = time.Since(start)

		// Try to get session from context (may not exist for public routes)
		if ctxValue := r.Context().Value(sessionIDKey); ctxValue != nil {
			if session, ok := ctxValue.(*models.Session); ok {
				info.UserID = int64(session.UserID)
				//info.Session = session
			}
		}

		infoJSON, err := json.Marshal(info)
		if err != nil {
			log.Println(debugTag+`Handler.LogRequest`, "err =", err)
			return
		}

		log.Printf(debugTag+`LogRequest LogTag: %s, {"HTTPinfo":%v}`, logTag, string(infoJSON))
	})
}

// OAuth codes, callback state and email tokens may appear in query strings.
func logSafeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.String()
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}
func (w *loggingResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type Stats struct {
	Time     time.Time `json:"Time"`
	Requests int64     `json:"Requests"`
	//FailedRequests int64     `json:"FailedRequests"`
}

// LogStats This is a MiddleWare handler that accumulates and logs server stats
// Ulitmately the aim of this is to keep any eye on what is happening with the server
// and take some action if something bad is going on.
// e.g. slow down the login response for some address if it is being attacked.
func (s *Stats) StatsCount(wr http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Requests++
		//Add something to determine errors
		//h.Stats.FailedRequests++
		if s.Time.IsZero() {
			s.Time = time.Now()
		}
		wr.ServeHTTP(w, r)
	})
}

func (s *Stats) StatsLog(logingDuration time.Duration) {
	for {
		time.Sleep(logingDuration)
		statsJSON, err := json.Marshal(s)
		if err != nil {
			log.Println(debugTag+`Handler.StatsLog`, "err =", err)
			return
		}

		log.Printf(debugTag+`{"Server Stats":%v}`, string(statsJSON))
		s.Requests = 0
		//h.Stats.FailedRequests = 0
	}
}

func (s *Stats) StatsRun(logingDuration time.Duration) {
	if logingDuration == 0 {
		logingDuration = time.Duration(1 * time.Minute)
	}
	go s.StatsLog(logingDuration)
}
