package httpserver

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWasmNegotiatesGzipAndRevalidatesSelectedRepresentation(t *testing.T) {
	dir := t.TempDir()
	wasm := []byte("\x00asm\x01\x00\x00\x00")
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(wasm); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"main.wasm": wasm, "main.wasm.gz": compressed.Bytes()} {
		if err := os.WriteFile(dir+"/"+name, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	h := Static(dir)
	request := func(method, encoding, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/main.wasm", nil)
		r.Header.Set("Accept-Encoding", encoding)
		r.Header.Set("If-None-Match", etag)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatal("response did not vary by encoding")
		}
		return w
	}
	identity := request("GET", "gzip;q=0, *;q=1", "")
	zipped := request("GET", "br, gzip", "")
	identityTag, gzipTag := identity.Header().Get("ETag"), zipped.Header().Get("ETag")
	if identity.Code != 200 || identity.Header().Get("Content-Encoding") != "" || !bytes.Equal(identity.Body.Bytes(), wasm) {
		t.Fatal("identity response was not preserved")
	}
	if zipped.Code != 200 || zipped.Header().Get("Content-Encoding") != "gzip" ||
		zipped.Header().Get("Content-Type") != "application/wasm" || gzipTag == "" || gzipTag == identityTag {
		t.Fatal("gzip response lacks its encoding, MIME type, or distinct validator")
	}
	reader, err := gzip.NewReader(zipped.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(decoded, wasm) {
		t.Fatal("gzip response did not decode to the original WASM")
	}
	if w := request("GET", "gzip", gzipTag); w.Code != 304 || w.Body.Len() != 0 {
		t.Fatal("gzip validator did not revalidate")
	}
	if w := request("GET", "gzip", identityTag); w.Code != 200 {
		t.Fatal("identity validator incorrectly matched gzip")
	}
	if w := request("HEAD", "gzip", ""); w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
		t.Fatal("gzip HEAD did not return headers without the body")
	}
}

func TestAssetsRevalidateByContentAndConfigIsUncached(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/main.wasm", []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	h := Static(dir)
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest("GET", "/main.wasm", nil))
	etag := first.Header().Get("ETag")
	if etag == "" || first.Code != 200 {
		t.Fatal("artifact has no content validator")
	}
	r := httptest.NewRequest("GET", "/main.wasm", nil)
	r.Header.Set("If-None-Match", etag)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 304 {
		t.Fatal("unchanged artifact was not revalidated")
	}
	if err := os.WriteFile(dir+"/main.wasm", []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	Static(dir).ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("ETag") == etag {
		t.Fatal("new deployment reused stale artifact")
	}
	w = httptest.NewRecorder()
	ClientConfig("/custom/v9")(w, httptest.NewRequest("GET", "/client-config.js", nil))
	if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), "/custom/v9") {
		t.Fatal("runtime API prefix unavailable")
	}
}
