package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSRedirectAndGracefulShutdown(t *testing.T) {
	fixture := httptest.NewTLSServer(http.NotFoundHandler())
	cert := fixture.TLS.Certificates[0]
	fixture.Close()
	plain, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	secure, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		plain.Close()
		t.Fatal(err)
	}
	g := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "", "", "https://trusted.example.test:8443")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- g.ServeListeners(ctx, plain, secure, cert) }()
	client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	defer client.CloseIdleConnections()
	r, err := http.NewRequest("GET", "http://"+plain.Addr().String()+"/api/v9/test?x=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "attacker.example.test"
	response, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 308 || response.Header.Get("Location") != "https://trusted.example.test:8443/api/v9/test?x=1" {
		t.Fatal("untrusted redirect or API served over HTTP")
	}
	response, err = client.Get("https://" + secure.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatal("HTTPS application unavailable")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
}
