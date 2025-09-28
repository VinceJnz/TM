package helpers

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type HTTPInfo struct {
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
}

// LogRequest This is a MiddleWare handler that logs all request to the console
func LogRequest(next http.Handler, sessionIDKey appCore.ContextKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := &HTTPInfo{
			Duration:      0,
			URL:           r.URL.String(),
			Host:          r.Host,
			Cookie:        "",
			Method:        r.Method,
			RequestURI:    r.RequestURI, // "",
			Referer:       r.Referer(),  // r.Header.Get("Referer"),
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
		lrw := &loggingResponseWriter{ResponseWriter: w, body: bytes.NewBuffer(nil)}
		next.ServeHTTP(lrw, r)

		// Log the response status code and body
		//log.Printf(debugTag+`LogRequest()1 Response status: %d`, lrw.statusCode)
		if lrw.statusCode != 0 && lrw.statusCode != 200 {
			log.Printf(debugTag+`LogRequest()2 Response body: %s, status: %d`, lrw.body.String(), lrw.statusCode)
		}

		cookie, err := r.Cookie("session")
		if err == nil {
			info.Cookie = cookie.Value
			//token, err := h.srvc.Session.FindSessionToken(info.Cookie)
			//if err == nil {
			//	info.UserID = token.UserID
			//}
		}
		info.StatusCode = lrw.statusCode
		info.Protocol = r.Proto
		info.Size = r.ContentLength
		if r.TLS != nil {
			info.TLSProtocol = r.TLS.NegotiatedProtocol
		}
		info.Duration = time.Since(start)

		infoJSON, err := json.Marshal(info)
		if err != nil {
			log.Println(debugTag+`Handler.LogRequest()5`, "err =", err)
			return
		}

		log.Printf(debugTag+`LogRequest {"HTTPinfo":%v}`, string(infoJSON))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

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
			log.Println(debugTag+`Handler.StatsLog()1`, "err =", err)
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
