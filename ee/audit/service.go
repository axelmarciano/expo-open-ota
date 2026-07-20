// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package audit

import (
	"context"
	"errors"
	"expo-open-ota/ee/licensing"
	"log"
	"time"
)

// ActorType distinguishes who acted: a signed-in dashboard account, an
// app-scoped API credential (CLI publishes), or the server itself (SSO
// provisioning, scheduled jobs).
type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorAPIKey ActorType = "api_key"
	ActorSystem ActorType = "system"
)

// Outcome is the result of the recorded action. Denied comes from the RBAC
// middleware refusing a permission; failure is reserved for actions that ran
// and failed in a security-relevant way (bad login).
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeDenied  Outcome = "denied"
	OutcomeFailure Outcome = "failure"
)

// Action names follow target.action, past tense. This catalog is the
// documented contract of the audit log (mirrored in the dashboard's filter
// dropdown and in the public docs): call sites must use these constants, and
// adding one here means documenting it.
type Action string

const (
	// Auth and account.
	ActionUserLogin           Action = "user.login"
	ActionUserSSOLogin        Action = "user.sso_login"
	ActionUserPasswordChanged Action = "user.password_changed"

	// Users and privileges.
	ActionUserCreated        Action = "user.created"
	ActionUserUpdated        Action = "user.updated"
	ActionUserDeleted        Action = "user.deleted"
	ActionUserSSOProvisioned Action = "user.sso_provisioned"
	ActionUserApproved       Action = "user.approved"
	ActionUserAdminGranted   Action = "user.admin_granted"
	ActionUserAdminRevoked   Action = "user.admin_revoked"
	ActionRoleCreated        Action = "role.created"
	ActionRoleUpdated        Action = "role.updated"
	ActionRoleDeleted        Action = "role.deleted"
	ActionUserGrantsUpdated  Action = "user.grants_updated"

	// Enterprise administration.
	ActionLicenseActivated Action = "license.activated"
	ActionLicenseRemoved   Action = "license.removed"
	ActionSSOConfigSaved   Action = "sso_config.saved"
	ActionSSOConfigDeleted Action = "sso_config.deleted"

	// App management.
	ActionAppCreated           Action = "app.created"
	ActionAppRenamed           Action = "app.renamed"
	ActionAppDeleted           Action = "app.deleted"
	ActionChannelCreated       Action = "channel.created"
	ActionChannelDeleted       Action = "channel.deleted"
	ActionChannelBranchMapped  Action = "channel_branch.mapped"
	ActionBranchCreated        Action = "branch.created"
	ActionBranchDeleted        Action = "branch.deleted"

	// Delivery.
	ActionUpdatePublished       Action = "update.published"
	ActionUpdateRollback        Action = "update.rollback"
	ActionUpdateRepublished     Action = "update.republished"
	ActionChannelRolloutStarted Action = "channel_rollout.started"
	ActionChannelRolloutUpdated Action = "channel_rollout.updated"
	ActionChannelRolloutEnded   Action = "channel_rollout.ended"
	ActionUpdateRolloutSet      Action = "update_rollout.set"
	ActionUpdateRolloutReverted Action = "update_rollout.reverted"

	// Credentials and key material.
	ActionAPIKeyCreated             Action = "apikey.created"
	ActionAPIKeyRevoked             Action = "apikey.revoked"
	ActionAPIKeyRestrictionsUpdated Action = "apikey.restrictions_updated"
	ActionBranchProtectionUpdated   Action = "branch_protection.updated"
	ActionCertificateDownloaded     Action = "certificate.downloaded"

	// Access control and the log itself.
	ActionPermissionDenied Action = "permission.denied"
	ActionAuditExported    Action = "audit.exported"
)

// Event is one audit log entry. Actor and target carry denormalized display
// names taken at write time: entries must keep rendering identically after
// the user, app or key they mention is deleted. AppID empty means the event
// is account-level (users, roles, license, SSO). Metadata never carries
// secrets or full request bodies.
type Event struct {
	ID            int64
	OccurredAt    time.Time
	ActorType     ActorType
	ActorID       string
	ActorDisplay  string
	Action        Action
	TargetType    string
	TargetID      string
	TargetDisplay string
	AppID         string
	Outcome       Outcome
	IP            string
	UserAgent     string
	Metadata      map[string]any
}

// ListFilters are the viewer's optional filters; nil means "any".
type ListFilters struct {
	ActorID *string
	Action  *string
	AppID   *string
	From    *time.Time
	To      *time.Time
}

// ListParams adds keyset pagination to the filters. BeforeID is the cursor
// (nil on the first page); Limit is clamped to [1, MaxPageSize] by the
// service, 0 meaning DefaultPageSize.
type ListParams struct {
	ListFilters
	BeforeID *int64
	Limit    int
}

const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// AuditRepository persists and reads audit entries. Insert is the only write:
// the log is append-only by construction (the retention purge, added with the
// purge job, will be the single exception).
type AuditRepository interface {
	Insert(ctx context.Context, event Event) (Event, error)
	List(ctx context.Context, params ListParams) ([]Event, error)
	Count(ctx context.Context, filters ListFilters) (int64, error)
}

var ErrRequiresControlPlane = errors.New("the audit log lives in the database: this deployment runs in stateless mode, which is community edition only")

// AuditService records and serves the enterprise audit trail. Recording is
// hard-gated on Enabled(): without a control plane and a currently valid
// license, Record is a silent no-op and nothing is ever collected. Reads only
// need the control plane, so a deployment whose license lapsed keeps read
// access to what was collected while it was licensed.
type AuditService struct {
	repo AuditRepository
	// licenseValid is the live licensing state; a field so same-package tests
	// can pin it without minting signed keys.
	licenseValid func() bool
}

// NewAuditService accepts a nil repository (stateless mode); reads then answer
// ErrRequiresControlPlane and Record no-ops.
func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{repo: repo, licenseValid: licensing.IsEnterprise}
}

// Enabled reports whether events are being collected right now.
func (s *AuditService) Enabled() bool {
	return s.repo != nil && s.licenseValid()
}

// Record writes one event, best-effort: a failed insert logs and is dropped
// rather than failing the user-facing request that emitted it. Call sites
// stay unconditional; the enterprise gate lives here.
func (s *AuditService) Record(ctx context.Context, event Event) {
	if !s.Enabled() {
		return
	}
	if event.ActorType == "" {
		event.ActorType = ActorSystem
	}
	if event.Outcome == "" {
		event.Outcome = OutcomeSuccess
	}
	// The entry must land even when the client disconnects right after the
	// mutation: the insert outlives the request context's cancellation.
	if _, err := s.repo.Insert(context.WithoutCancel(ctx), event); err != nil {
		log.Printf("audit: failed to record %q: %v", event.Action, err)
	}
}

// List returns one viewer page, newest first, and the cursor for the next one
// (nil when this page is the last). It fetches one extra row to answer "is
// there more" without a second query.
func (s *AuditService) List(ctx context.Context, params ListParams) ([]Event, *int64, error) {
	if s.repo == nil {
		return nil, nil, ErrRequiresControlPlane
	}
	if params.Limit <= 0 {
		params.Limit = DefaultPageSize
	}
	if params.Limit > MaxPageSize {
		params.Limit = MaxPageSize
	}
	pageSize := params.Limit
	params.Limit = pageSize + 1
	events, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	if len(events) <= pageSize {
		return events, nil, nil
	}
	events = events[:pageSize]
	nextCursor := events[pageSize-1].ID
	return events, &nextCursor, nil
}

// Count is the filtered total shown next to the paginated list.
func (s *AuditService) Count(ctx context.Context, filters ListFilters) (int64, error) {
	if s.repo == nil {
		return 0, ErrRequiresControlPlane
	}
	return s.repo.Count(ctx, filters)
}
