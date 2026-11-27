package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

func ClientConfig(apiBase string) http.HandlerFunc {
	data, _ := json.Marshal(map[string]string{"apiBase": apiBase})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte("window.TM_CONFIG=" + string(data) + ";"))
	}
}

// Hash immutable image artifacts once. Revalidate the WASM/runtime pair across
// deployments without timestamp URLs or retaining entire binaries in memory.
func Static(dir string) http.Handler {
	etags := map[string]string{}
	for _, name := range []string{"index.html", "main.wasm", "main.wasm.gz", "wasm_exec.js"} {
		file, err := os.Open(dir + "/" + name)
		if err != nil {
			continue
		}
		hash := sha256.New()
		_, err = io.Copy(hash, file)
		file.Close()
		if err == nil {
			etags["/"+name] = "\"" + hex.EncodeToString(hash.Sum(nil)) + "\""
		}
	}
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Clean(r.URL.Path)
		if name == "/" {
			name = "/index.html"
		}
		w.Header().Set("Cache-Control", "no-cache")
		if name == "/main.wasm" && etags["/main.wasm.gz"] != "" {
			w.Header().Add("Vary", "Accept-Encoding")
			if acceptsGzip(r.Header.Values("Accept-Encoding")) {
				compressed, err := os.Open(dir + "/main.wasm.gz")
				if err == nil {
					defer compressed.Close()
					if info, err := compressed.Stat(); err == nil {
						w.Header().Set("Content-Type", "application/wasm")
						w.Header().Set("Content-Encoding", "gzip")
						w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
						w.Header().Set("ETag", etags["/main.wasm.gz"])
						http.ServeContent(w, r, "main.wasm", info.ModTime(), compressed)
						return
					}
				}
			}
		}
		if etag := etags[name]; etag != "" {
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}

func acceptsGzip(values []string) bool {
	wildcard := false
	for _, entry := range strings.Split(strings.Join(values, ","), ",") {
		coding, parameters, _ := strings.Cut(entry, ";")
		coding = strings.ToLower(strings.TrimSpace(coding))
		if coding != "gzip" && coding != "*" {
			continue
		}
		accepted := true
		for _, parameter := range strings.Split(parameters, ";") {
			key, value, _ := strings.Cut(parameter, "=")
			if strings.EqualFold(strings.TrimSpace(key), "q") {
				quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				accepted = err == nil && quality > 0 && quality <= 1
				break
			}
		}
		if coding == "gzip" {
			return accepted
		}
		wildcard = accepted
	}
	return wildcard
}
