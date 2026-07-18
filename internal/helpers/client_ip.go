package helpers

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

// ClientIP resolves the caller's address for IP-based access restrictions.
//
// By default it is the TCP peer (RemoteAddr): unforgeable, but behind a load
// balancer every request carries the balancer's address. When
// TRUST_PROXY_HEADERS=true the client is read from X-Forwarded-For instead.
//
// X-Forwarded-For is a comma-separated list to which every proxy appends the
// address it received the connection from, so the entries a client can forge
// always sit to the LEFT of the one your outermost trusted proxy added. We
// therefore count trusted proxies from the RIGHT: with N proxies in front of
// the server, the real client is the entry N positions from the end.
// TRUST_PROXY_DEPTH sets N (default 1, the single load-balancer case). Counting
// from the right keeps the choice unspoofable, because extra entries a client
// injects only grow the left of the list and never shift the Nth-from-right
// entry.
//
// If the header carries fewer entries than TRUST_PROXY_DEPTH, the request did
// not traverse the whole trusted chain, so we fall back to RemoteAddr (the
// immediate peer), which a client allowlist will reject. Returns the zero Addr
// when nothing parses; callers with an IP allowlist must treat an unresolvable
// address as a mismatch, not as a pass.
//
// The env vars are read with os.Getenv rather than config.GetEnv because the
// config package imports helpers (import cycle); neither var has a config
// default, so the behavior is identical.
func ClientIP(r *http.Request) netip.Addr {
	if os.Getenv("TRUST_PROXY_HEADERS") == "true" {
		forwardedFor := r.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			entries := strings.Split(forwardedFor, ",")
			if idx := len(entries) - trustedProxyDepth(); idx >= 0 {
				candidate := strings.TrimSpace(entries[idx])
				if addr, err := netip.ParseAddr(candidate); err == nil {
					return addr.Unmap()
				}
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

// trustedProxyDepth is the number of trusted proxies in front of the server,
// from TRUST_PROXY_DEPTH. It defaults to 1 and is never below 1: a value of 0
// or less, or an unparseable one, would mean "trust the rightmost entry, which
// a client can influence in a multi-proxy chain", so it is clamped up to 1.
func trustedProxyDepth() int {
	if raw := os.Getenv("TRUST_PROXY_DEPTH"); raw != "" {
		if depth, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && depth >= 1 {
			return depth
		}
	}
	return 1
}
