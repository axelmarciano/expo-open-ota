// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyscopes

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"testing"
)

type fakeScopeRepo struct {
	scopes []ApiKeyScopes

	replacedAppID      string
	replacedApiKeyID   int64
	replacedChannelIDs []int64
	replacedAllowedIps []netip.Prefix
	replaceCalls       int
}

func (f *fakeScopeRepo) GetScopesByAppID(ctx context.Context, appID string) ([]ApiKeyScopes, error) {
	return f.scopes, nil
}

func (f *fakeScopeRepo) ReplaceScopes(ctx context.Context, appID string, apiKeyID int64, channelIDs []int64, allowedIps []netip.Prefix) error {
	f.replaceCalls++
	f.replacedAppID = appID
	f.replacedApiKeyID = apiKeyID
	f.replacedChannelIDs = channelIDs
	f.replacedAllowedIps = allowedIps
	return nil
}

func serviceWith(repo ApiKeyScopeRepository, licensed bool) *ApiKeyScopeService {
	service := NewApiKeyScopeService(repo)
	service.licenseValid = func() bool { return licensed }
	return service
}

func TestStatelessModeAnswersControlPlaneError(t *testing.T) {
	service := serviceWith(nil, true)
	if _, err := service.GetScopes(context.Background(), "app"); !errors.Is(err, ErrRequiresControlPlane) {
		t.Fatalf("expected ErrRequiresControlPlane, got %v", err)
	}
	if err := service.SetScopes(context.Background(), "app", 1, nil, nil); !errors.Is(err, ErrRequiresControlPlane) {
		t.Fatalf("expected ErrRequiresControlPlane, got %v", err)
	}
}

func TestSetScopesRequiresValidLicense(t *testing.T) {
	repo := &fakeScopeRepo{}
	service := serviceWith(repo, false)
	if err := service.SetScopes(context.Background(), "app", 1, []int64{2}, nil); !errors.Is(err, ErrRequiresValidLicense) {
		t.Fatalf("expected ErrRequiresValidLicense, got %v", err)
	}
	if repo.replaceCalls != 0 {
		t.Fatal("repository must not be touched without a valid license")
	}
}

// Reads stay open without a license so the dashboard can always show which
// restrictions exist on a key, even after the license lapsed.
func TestGetScopesDoesNotRequireLicense(t *testing.T) {
	repo := &fakeScopeRepo{scopes: []ApiKeyScopes{{ApiKeyID: 7}}}
	service := serviceWith(repo, false)
	scopes, err := service.GetScopes(context.Background(), "app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scopes) != 1 || scopes[0].ApiKeyID != 7 {
		t.Fatalf("unexpected scopes: %+v", scopes)
	}
}

func TestSetScopesNormalizesAndDeduplicates(t *testing.T) {
	repo := &fakeScopeRepo{}
	service := serviceWith(repo, true)
	err := service.SetScopes(context.Background(), "app", 1,
		[]int64{3, 2, 3},
		[]string{" 192.168.1.5/24 ", "10.1.2.3", "2001:db8::1", "192.168.1.0/24", ""},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(repo.replacedChannelIDs, []int64{3, 2}) {
		t.Fatalf("unexpected channel ids: %v", repo.replacedChannelIDs)
	}
	// 192.168.1.5/24 masks to 192.168.1.0/24 (postgres cidr rejects host
	// bits), the duplicate masked entry collapses, bare addresses become
	// single-host prefixes.
	expected := []netip.Prefix{
		netip.MustParsePrefix("192.168.1.0/24"),
		netip.MustParsePrefix("10.1.2.3/32"),
		netip.MustParsePrefix("2001:db8::1/128"),
	}
	if !reflect.DeepEqual(repo.replacedAllowedIps, expected) {
		t.Fatalf("unexpected allowed ips: %v", repo.replacedAllowedIps)
	}
}

func TestSetScopesRejectsInvalidCidr(t *testing.T) {
	repo := &fakeScopeRepo{}
	service := serviceWith(repo, true)
	err := service.SetScopes(context.Background(), "app", 1, nil, []string{"not-an-ip"})
	if !errors.Is(err, ErrInvalidCidr) {
		t.Fatalf("expected ErrInvalidCidr, got %v", err)
	}
	if repo.replaceCalls != 0 {
		t.Fatal("repository must not be touched on invalid input")
	}
}

// An empty allowlist must reach the repository as nil so the column stores
// NULL (no restriction) instead of an empty array.
func TestSetScopesEmptyAllowlistIsNil(t *testing.T) {
	repo := &fakeScopeRepo{}
	service := serviceWith(repo, true)
	if err := service.SetScopes(context.Background(), "app", 1, nil, []string{"", "  "}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.replacedAllowedIps != nil {
		t.Fatalf("expected nil allowlist, got %v", repo.replacedAllowedIps)
	}
}
