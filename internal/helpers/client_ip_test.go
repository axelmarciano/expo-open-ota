package helpers

import (
	"net/http"
	"net/netip"
	"testing"
)

func requestWith(remoteAddr, xff string) *http.Request {
	r := &http.Request{
		RemoteAddr: remoteAddr,
		Header:     http.Header{},
	}
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name         string
		trustHeaders string
		depth        string
		remoteAddr   string
		xff          string
		want         string // expected address, or "" for the zero (unresolvable) Addr
	}{
		{
			name:       "no proxy trust falls back to RemoteAddr and ignores XFF",
			remoteAddr: "203.0.113.9:54321",
			xff:        "1.1.1.1",
			want:       "203.0.113.9",
		},
		{
			name:       "RemoteAddr without a port still parses",
			remoteAddr: "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "unparseable RemoteAddr yields the zero Addr",
			remoteAddr: "not-an-ip",
			want:       "",
		},
		{
			name:         "single proxy: rightmost XFF entry is the client",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "203.0.113.7",
			want:         "203.0.113.7",
		},
		{
			name:         "single proxy: client-forged left entry is ignored",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "1.1.1.1, 203.0.113.7",
			want:         "203.0.113.7",
		},
		{
			name:         "two proxies: client is two hops from the right",
			trustHeaders: "true",
			depth:        "2",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "203.0.113.7, 10.0.0.5",
			want:         "203.0.113.7",
		},
		{
			name:         "two proxies: client-forged entries do not shift the result",
			trustHeaders: "true",
			depth:        "2",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "1.1.1.1, 203.0.113.7, 10.0.0.5",
			want:         "203.0.113.7",
		},
		{
			name:         "depth larger than the chain falls back to RemoteAddr",
			trustHeaders: "true",
			depth:        "2",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "203.0.113.7",
			want:         "10.0.0.1",
		},
		{
			name:         "trust enabled but no XFF header falls back to RemoteAddr",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			want:         "10.0.0.1",
		},
		{
			name:         "IPv4-mapped IPv6 entry is unmapped",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "::ffff:203.0.113.7",
			want:         "203.0.113.7",
		},
		{
			name:         "IPv6 client entry is returned as-is",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "2001:db8::1",
			want:         "2001:db8::1",
		},
		{
			name:         "unparseable target entry falls back to RemoteAddr",
			trustHeaders: "true",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "garbage",
			want:         "10.0.0.1",
		},
		{
			name:         "invalid depth is clamped to 1 (rightmost)",
			trustHeaders: "true",
			depth:        "abc",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "1.1.1.1, 203.0.113.7",
			want:         "203.0.113.7",
		},
		{
			name:         "zero depth is clamped to 1 and does not panic",
			trustHeaders: "true",
			depth:        "0",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "1.1.1.1, 203.0.113.7",
			want:         "203.0.113.7",
		},
		{
			name:         "negative depth is clamped to 1",
			trustHeaders: "true",
			depth:        "-3",
			remoteAddr:   "10.0.0.1:8080",
			xff:          "1.1.1.1, 203.0.113.7",
			want:         "203.0.113.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv also unsets after the test, so cases without these keys
			// see an empty env rather than a value bleeding in from a sibling.
			t.Setenv("TRUST_PROXY_HEADERS", tt.trustHeaders)
			t.Setenv("TRUST_PROXY_DEPTH", tt.depth)

			got := ClientIP(requestWith(tt.remoteAddr, tt.xff))

			if tt.want == "" {
				if got.IsValid() {
					t.Fatalf("expected the zero Addr, got %v", got)
				}
				return
			}
			want := netip.MustParseAddr(tt.want)
			if got != want {
				t.Fatalf("expected %v, got %v", want, got)
			}
		})
	}
}
