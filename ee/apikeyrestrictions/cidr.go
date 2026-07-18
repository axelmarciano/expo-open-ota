// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyrestrictions

import (
	"fmt"
	"net/netip"
	"strings"
)

// This file holds both halves of the allowlist normalization contract: what
// an admin enters (parseCidrs) and what a caller is matched against
// (ipAllowed). The canonical form on both sides is the unmapped address, the
// same form helpers.ClientIP resolves; if one half changes the other must
// follow.

// ipAllowed reports whether the caller's address falls in one of the
// allowlisted ranges. An unresolvable address never passes an allowlist.
func ipAllowed(clientIP netip.Addr, allowed []netip.Prefix) bool {
	if !clientIP.IsValid() {
		return false
	}
	clientIP = clientIP.Unmap()
	for _, prefix := range allowed {
		if prefix.Contains(clientIP) {
			return true
		}
	}
	return false
}

// parseCidrs validates and normalizes user-entered allowlist entries. A bare
// address is treated as a single-host range. Returns nil for an empty list so
// the column stores NULL (= no IP restriction) rather than an empty array.
func parseCidrs(entries []string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	seen := make(map[netip.Prefix]bool)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		var prefix netip.Prefix
		if strings.Contains(entry, "/") {
			parsed, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("%w: %q", ErrInvalidCidr, entry)
			}
			// An IPv4-mapped ::ffff: prefix is stored in IPv4 form: ipAllowed
			// compares callers unmapped, so an IPv6-family prefix would never
			// match them. The first 96 bits are the mapped block itself, the
			// rest is the IPv4 part; a prefix wider than /96 leaks outside
			// the block and has no IPv4 equivalent.
			if parsed.Addr().Is4In6() {
				if parsed.Bits() < 96 {
					return nil, fmt.Errorf("%w: %q", ErrInvalidCidr, entry)
				}
				parsed = netip.PrefixFrom(parsed.Addr().Unmap(), parsed.Bits()-96)
			}
			prefix = parsed.Masked()
		} else {
			addr, err := netip.ParseAddr(entry)
			if err != nil {
				return nil, fmt.Errorf("%w: %q", ErrInvalidCidr, entry)
			}
			// Unmap so ::ffff:10.1.2.3 and 10.1.2.3 collapse into the same
			// single-host entry, in the form the unmapped caller will carry.
			addr = addr.Unmap()
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		if !seen[prefix] {
			seen[prefix] = true
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes, nil
}
