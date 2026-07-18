// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyrestrictions

import (
	"context"
	"errors"
	"expo-open-ota/ee/licensing"
	"expo-open-ota/internal/services"
	"fmt"
	"net/netip"
)

// ApiKeyRestrictions is the enterprise access restrictions attached to one
// API key: whether it may act on protected branches (false by default, an
// admin grants it explicitly) and the source networks it may be used from
// (empty = any address).
type ApiKeyRestrictions struct {
	ApiKeyID                   int64
	CanAccessProtectedBranches bool
	AllowedIps                 []netip.Prefix
}

// ApiKeyRestrictionRepository persists per-key restrictions and the branch
// protection flag. GetRestrictions and IsBranchProtected are the enforcement
// reads on the CLI request hot path.
type ApiKeyRestrictionRepository interface {
	GetRestrictionsByAppID(ctx context.Context, appID string) ([]ApiKeyRestrictions, error)
	SetRestrictions(ctx context.Context, appID string, apiKeyID int64, canAccessProtectedBranches bool, allowedIps []netip.Prefix) error
	GetRestrictions(ctx context.Context, apiKeyID int64) (ApiKeyRestrictions, error)
	SetBranchProtection(ctx context.Context, appID string, branchName string, protected bool) error
	IsBranchProtected(ctx context.Context, appID string, branchName string) (bool, error)
}

var (
	ErrRequiresControlPlane = errors.New("api key restrictions are managed in the database: this deployment runs in stateless mode, which is community edition only")
	ErrRequiresValidLicense = errors.New("api key restrictions require an active enterprise license")
	ErrApiKeyNotFound       = errors.New("api key not found")
	ErrBranchNotFound       = errors.New("branch not found")
	ErrInvalidCidr          = errors.New("invalid IP or CIDR range")

	// Both wrap services.ErrCliAccessDenied so the community handlers can map
	// them to a 403 without knowing anything about this package.
	ErrIpNotAllowed    = fmt.Errorf("%w: this API key cannot be used from this IP address", services.ErrCliAccessDenied)
	ErrBranchProtected = fmt.Errorf("%w: this branch is protected and this API key is not allowed to act on protected branches", services.ErrCliAccessDenied)
)

// ApiKeyRestrictionService owns the management and the enforcement of per-key
// restrictions and branch protection. Mutations are license-gated (no valid
// license, no changes); reads are not, so the dashboard can always show what
// restrictions exist.
type ApiKeyRestrictionService struct {
	repo ApiKeyRestrictionRepository
	// licenseValid is the live licensing state; a field so same-package tests
	// can pin it without minting signed keys.
	licenseValid func() bool
}

// NewApiKeyRestrictionService accepts a nil repository (stateless mode);
// every management method then answers ErrRequiresControlPlane and the
// enforcement is a no-op.
func NewApiKeyRestrictionService(repo ApiKeyRestrictionRepository) *ApiKeyRestrictionService {
	return &ApiKeyRestrictionService{repo: repo, licenseValid: licensing.IsEnterprise}
}

func (s *ApiKeyRestrictionService) GetRestrictionsByApp(ctx context.Context, appID string) ([]ApiKeyRestrictions, error) {
	if s.repo == nil {
		return nil, ErrRequiresControlPlane
	}
	return s.repo.GetRestrictionsByAppID(ctx, appID)
}

// SetRestrictions replaces the restrictions of one API key. CIDR entries are
// normalized (bare addresses become /32 or /128, host bits are masked off)
// because the postgres cidr type rejects unmasked values.
func (s *ApiKeyRestrictionService) SetRestrictions(ctx context.Context, appID string, apiKeyID int64, canAccessProtectedBranches bool, cidrs []string) error {
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
	return s.repo.SetRestrictions(ctx, appID, apiKeyID, canAccessProtectedBranches, allowedIps)
}

func (s *ApiKeyRestrictionService) SetBranchProtection(ctx context.Context, appID string, branchName string, protected bool) error {
	if s.repo == nil {
		return ErrRequiresControlPlane
	}
	if !s.licenseValid() {
		return ErrRequiresValidLicense
	}
	return s.repo.SetBranchProtection(ctx, appID, branchName, protected)
}

// AuthorizeCliRequest implements services.CliAccessPolicy: it enforces the
// authenticated key's restrictions on the CLI request path. Enforcement is an
// enterprise feature, so without a control plane or an active license nothing
// is enforced and every request passes (community behavior).
//
// branchName is empty for requests that do not target a branch (reads, local
// file uploads); those only go through the IP allowlist. A branch that does
// not exist yet is not protected: publishing to a brand-new branch stays
// open, protecting it is an explicit admin action afterwards.
func (s *ApiKeyRestrictionService) AuthorizeCliRequest(ctx context.Context, appID string, apiKeyID int64, branchName string, clientIP netip.Addr) error {
	if s.repo == nil || !s.licenseValid() {
		return nil
	}
	restrictions, err := s.repo.GetRestrictions(ctx, apiKeyID)
	if err != nil {
		return err
	}
	if len(restrictions.AllowedIps) > 0 && !ipAllowed(clientIP, restrictions.AllowedIps) {
		return ErrIpNotAllowed
	}
	if branchName != "" && !restrictions.CanAccessProtectedBranches {
		protected, err := s.repo.IsBranchProtected(ctx, appID, branchName)
		if err != nil {
			return err
		}
		if protected {
			return ErrBranchProtected
		}
	}
	return nil
}
