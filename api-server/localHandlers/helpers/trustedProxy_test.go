package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedProxyRejectsSpoofingAndUsesNearestUntrustedHop(t *testing.T) {
	middleware, err := TrustedProxy("10.0.0.0/8,::1/128")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ peer, header, want string }{
		{"192.0.2.1:12", "1.2.3.4", "192.0.2.1:12"},
		{"10.0.0.1:12", "1.2.3.4,192.0.2.1,10.0.0.2", "192.0.2.1:0"},
		{"10.0.0.1:12", "not-an-ip", "10.0.0.1:12"},
		{"[::1]:12", "2001:db8::1", "[2001:db8::1]:0"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.peer
		r.Header.Set("X-Forwarded-For", tc.header)
		middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if r.RemoteAddr != tc.want {
				t.Fatal(r.RemoteAddr, tc.want)
			}
		})).ServeHTTP(httptest.NewRecorder(), r)
	}
	if _, err = TrustedProxy("*"); err == nil {
		t.Fatal("invalid proxy range accepted")
	}
}
