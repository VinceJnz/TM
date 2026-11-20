package helpers

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// TrustedProxy accepts forwarding information only from configured network peers.
// Walk right to left so an untrusted client cannot prepend its own identity.
func TrustedProxy(csv string) (func(http.Handler) http.Handler, error) {
	var prefixes []netip.Prefix
	for _, entry := range strings.Split(csv, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry")
		}
		prefixes = append(prefixes, prefix)
	}
	trusted := func(ip netip.Addr) bool {
		for _, prefix := range prefixes {
			if prefix.Contains(ip.Unmap()) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			peer, parseErr := netip.ParseAddr(host)
			if err != nil || parseErr != nil || !trusted(peer) {
				next.ServeHTTP(w, r)
				return
			}
			raw := r.Header.Get("X-Forwarded-For")
			hops := strings.Split(raw, ",")
			if raw == "" || len(raw) > 4096 || len(hops) > 20 {
				next.ServeHTTP(w, r)
				return
			}
			addresses := make([]netip.Addr, len(hops))
			for i, hop := range hops {
				ip, err := netip.ParseAddr(strings.TrimSpace(hop))
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				addresses[i] = ip.Unmap()
			}
			client := peer.Unmap()
			for i := len(addresses) - 1; i >= 0 && trusted(client); i-- {
				client = addresses[i]
			}
			copy := r.Clone(r.Context())
			copy.RemoteAddr = net.JoinHostPort(client.String(), "0")
			next.ServeHTTP(w, copy)
		})
	}, nil
}
