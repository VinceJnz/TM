package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
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
	for _, name := range []string{"index.html", "main.wasm", "wasm_exec.js"} {
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
