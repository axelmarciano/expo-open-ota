package helpers

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
)

// ClientIP resolves the caller's address for IP-based access restrictions.
//
// By default it is the TCP peer (RemoteAddr): unforgeable, but behind a load
// balancer every request carries the balancer's address. When
// TRUST_PROXY_HEADERS=true the rightmost X-Forwarded-For entry wins instead:
// that is the value appended by the proxy directly in front of the server,
// the only entry of the list a client cannot forge. Only enable it when the
// server is reachable exclusively through that proxy.
//
// Returns the zero Addr when nothing parses; callers with an IP allowlist
// must treat an unresolvable address as a mismatch, not as a pass.
// The env var is read with os.Getenv rather than config.GetEnv because the
// config package imports helpers (import cycle); TRUST_PROXY_HEADERS has no
// default value, so the behavior is identical.
func ClientIP(r *http.Request) netip.Addr {
	if os.Getenv("TRUST_PROXY_HEADERS") == "true" {
		forwardedFor := r.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			entries := strings.Split(forwardedFor, ",")
			candidate := strings.TrimSpace(entries[len(entries)-1])
			if addr, err := netip.ParseAddr(candidate); err == nil {
				return addr.Unmap()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}
