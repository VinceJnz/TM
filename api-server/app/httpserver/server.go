package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Group struct {
	HTTP, HTTPS *http.Server
	draining    atomic.Bool
}

func New(handler http.Handler, httpAddr, httpsAddr, httpsOrigin string) *Group {
	g := &Group{}
	guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.draining.Load() && r.URL.Path == "/readyz" {
			http.Error(w, "draining", 503)
			return
		}
		handler.ServeHTTP(w, r)
	})
	redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			guard.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, strings.TrimRight(httpsOrigin, "/")+r.URL.RequestURI(), http.StatusPermanentRedirect)
	})
	server := func(addr string, h http.Handler) *http.Server {
		return &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	}
	g.HTTP = server(httpAddr, redirect)
	g.HTTPS = server(httpsAddr, guard)
	return g
}

func (g *Group) Serve(ctx context.Context, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return errors.New("TLS certificate/key unavailable")
	}
	plain, err := net.Listen("tcp", g.HTTP.Addr)
	if err != nil {
		return err
	}
	defer plain.Close()
	secure, err := net.Listen("tcp", g.HTTPS.Addr)
	if err != nil {
		return err
	}
	defer secure.Close()
	return g.ServeListeners(ctx, plain, secure, cert)
}

func (g *Group) ServeListeners(ctx context.Context, plain, secure net.Listener, cert tls.Certificate) error {
	errorsCh := make(chan error, 2)
	go func() { errorsCh <- g.HTTP.Serve(plain) }()
	go func() {
		errorsCh <- g.HTTPS.Serve(tls.NewListener(secure, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}))
	}()
	var result error
	select {
	case <-ctx.Done():
	case result = <-errorsCh:
	}
	g.draining.Store(true)
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Shutdown both listeners before closing shared database/services.
	done := make(chan error, 1)
	go func() { done <- g.HTTP.Shutdown(shutdown) }()
	secureErr := g.HTTPS.Shutdown(shutdown)
	plainErr := <-done
	if secureErr != nil || plainErr != nil {
		g.HTTP.Close()
		g.HTTPS.Close()
		if result == nil {
			result = errors.New("server shutdown timed out")
		}
	}
	if errors.Is(result, http.ErrServerClosed) {
		return nil
	}
	return result
}
