package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeAcceptsOnlyReadyAndRespectsCancellation(t *testing.T) {
	for _, status := range []int{200, 503} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		err := probe(context.Background(), server.Client(), server.URL)
		server.Close()
		if (err == nil) != (status == 200) {
			t.Fatal(status, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if probe(ctx, http.DefaultClient, "http://127.0.0.1:1") == nil {
		t.Fatal("cancelled probe succeeded")
	}
}
