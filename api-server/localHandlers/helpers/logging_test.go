package helpers

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggingKeepsOnlyBoundedMetadata(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	h := LogRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.WriteHeader(500)
		w.Write([]byte(strings.Repeat("private-response", 100000)))
	}), "test", "test")
	r := httptest.NewRequest("GET", "/callback?token=private-query", nil)
	r.Header.Set("Referer", "https://example.test/?code=private-code")
	r.AddCookie(&http.Cookie{Name: "session", Value: "private-cookie"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if len(output.String()) > 2048 {
		t.Fatal("response body was retained in logs")
	}
	for _, secret := range []string{"private-query", "private-code", "private-cookie", "private-response"} {
		if strings.Contains(output.String(), secret) {
			t.Fatal("sensitive response/request data was logged")
		}
	}
	if !strings.Contains(output.String(), `"StatusCode":201`) || !strings.Contains(output.String(), `"Size":1600000`) {
		t.Fatal("incorrect status or response byte count:", output.String())
	}
}

func TestImplicitSuccessStatus(t *testing.T) {
	w := &loggingResponseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: 200}
	if _, err := w.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if w.statusCode != 200 || w.bytesWritten != 2 {
		t.Fatal("implicit status/count incorrect")
	}
}
