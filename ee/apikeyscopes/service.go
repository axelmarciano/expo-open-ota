// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyscopes

import (
	"context"
	"errors"
	"expo-open-ota/ee/licensing"
	"fmt"
	"net/netip"
	"strings"
)

// ApiKeyScopes is the enterprise access restrictions attached to one API key:
// the release channels the key may act on (empty = all channels) and the
// source networks it may be used from (empty = any address).
type ApiKeyScopes struct {
	ApiKeyID   int64
	ChannelIDs []int64
	AllowedIps []netip.Prefix
}

// ApiKeyScopeRepository persists per-key restrictions. ReplaceScopes swaps the
// full restriction set of one key atomically; there is no incremental
// add/remove because the dashboard always saves the whole form.
type ApiKeyScopeRepository interface {
	GetScopesByAppID(ctx context.Context, appID string) ([]ApiKeyScopes, error)
	ReplaceScopes(ctx context.Context, appID string, apiKeyID int64, channelIDs []int64, allowedIps []netip.Prefix) error
}

var (
	ErrRequiresControlPlane = errors.New("api key restrictions are managed in the database: this deployment runs in stateless mode, which is community edition only")
	ErrRequiresValidLicense = errors.New("api key restrictions require an active enterprise license")
	ErrApiKeyNotFound       = errors.New("api key not found")
	ErrChannelNotFound      = errors.New("one or more channels do not exist for this app")
	ErrInvalidCidr          = errors.New("invalid IP or CIDR range")
)

// ApiKeyScopeService owns the management of per-key restrictions. Mutations
// are license-gated (no valid license, no changes); reads are not, so the
// dashboard can always show what restrictions exist.
type ApiKeyScopeService struct {
	repo ApiKeyScopeRepository
	// licenseValid is the live licensing state; a field so same-package tests
	// can pin it without minting signed keys.
	licenseValid func() bool
}

// NewApiKeyScopeService accepts a nil repository (stateless mode); every
// method then answers ErrRequiresControlPlane.
func NewApiKeyScopeService(repo ApiKeyScopeRepository) *ApiKeyScopeService {
	return &ApiKeyScopeService{repo: repo, licenseValid: licensing.IsEnterprise}
}

func (s *ApiKeyScopeService) GetScopes(ctx context.Context, appID string) ([]ApiKeyScopes, error) {
	if s.repo == nil {
		return nil, ErrRequiresControlPlane
	}
	return s.repo.GetScopesByAppID(ctx, appID)
}

// SetScopes replaces the restrictions of one API key. CIDR entries are
// normalized (bare addresses become /32 or /128, host bits are masked off)
// because the postgres cidr type rejects unmasked values.
func (s *ApiKeyScopeService) SetScopes(ctx context.Context, appID string, apiKeyID int64, channelIDs []int64, cidrs []string) error {
	if s.repo == nil {
		return ErrRequiresControlPlane
	}
	if !s.licenseValid() {
		return ErrRequiresValidLicense
	}
	allowedIps, err := parseCidrs(cidrs)
	if err != nil {
		return err
	}
	return s.repo.ReplaceScopes(ctx, appID, apiKeyID, dedupeInt64(channelIDs), allowedIps)
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
			prefix = parsed.Masked()
		} else {
			addr, err := netip.ParseAddr(entry)
			if err != nil {
				return nil, fmt.Errorf("%w: %q", ErrInvalidCidr, entry)
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		if !seen[prefix] {
			seen[prefix] = true
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes, nil
}

func dedupeInt64(values []int64) []int64 {
	var out []int64
	seen := make(map[int64]bool)
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
