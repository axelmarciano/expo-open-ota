package services

import (
	"context"
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/auditlog"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/store"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DashboardAuthService owns the admin dashboard's own session credentials: a
// short-lived session JWT and a long-lived refresh JWT, both minted after an
// email/password login. In control-plane mode the credentials are checked
// against the users table (userRepo); in stateless mode userRepo is nil and
// the single account comes from ADMIN_EMAIL/ADMIN_PASSWORD. It has no notion
// of apps — the credentials a CLI client presents for an app are a separate
// concern, see CliAuthService.
type DashboardAuthService struct {
	Secret   string
	userRepo UserRepository
	// onAuditEvent is the audit emission seam; nil (community) means sign-ins
	// are not recorded. Only the password login path emits here — the refresh
	// path is session upkeep, not an authentication event.
	onAuditEvent auditlog.RecordFunc
	// ssoEnforced reports whether SSO is currently active (configured, enabled
	// and licensed). Injected by the enterprise wiring; nil means never
	// enforced, so the community edition is untouched. While enforced, member
	// accounts must sign in through SSO and only admins keep the password
	// login as a break-glass access.
	ssoEnforced func(context.Context) bool
}

// DashboardSession is the JWT pair handed to the dashboard on login or refresh.
type DashboardSession struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// DashboardPrincipal identifies who is behind a validated dashboard session.
// UserId is empty in stateless mode, where the ADMIN_EMAIL account is not a
// database row and is always an admin.
type DashboardPrincipal struct {
	UserId  string
	Email   string
	IsAdmin bool
}

// ErrAdminEmailNotSet is surfaced verbatim to the login response: an operator
// hitting it needs the instruction, not a generic 401.
var ErrAdminEmailNotSet = errors.New("ADMIN_EMAIL is not set: stateless mode logs into the dashboard with ADMIN_EMAIL and ADMIN_PASSWORD. Set ADMIN_EMAIL on the server and retry")

// ErrPasswordLoginDisabledBySSO is returned to member accounts that present a
// valid password while SSO is enforced. It is only surfaced after the
// password verified, so the actionable message never leaks account existence.
var ErrPasswordLoginDisabledBySSO = errors.New("password sign-in is disabled while SSO is active: sign in with SSO instead")

// ErrAccountPendingApproval is returned for an account whose enabled flag is
// off: either an admin revoked its access, or SSO manual validation is on and
// the account has not been approved yet. Like ErrPasswordLoginDisabledBySSO it
// is only reached after the credentials verified, so it leaks nothing about
// which accounts exist.
var ErrAccountPendingApproval = errors.New("this account is waiting for an administrator to approve it")

// dashboardSubject scopes every dashboard JWT to the dashboard itself. Both
// validators below reject any other subject, which is what keeps the upload
// tokens minted by localBucket — signed with the same JWT_SECRET — from being
// accepted here. Changing this value invalidates sessions already in the wild.
const dashboardSubject = "admin-dashboard"

// NewDashboardAuthService accepts a nil repository (stateless mode); logins
// are then checked against ADMIN_EMAIL/ADMIN_PASSWORD.
func NewDashboardAuthService(userRepo UserRepository) *DashboardAuthService {
	return &DashboardAuthService{
		Secret:   config.GetEnv("JWT_SECRET"),
		userRepo: userRepo,
	}
}

// SetSSOEnforced injects the live "SSO is active" signal (see the field doc).
func (a *DashboardAuthService) SetSSOEnforced(enforced func(context.Context) bool) {
	a.ssoEnforced = enforced
}

func (a *DashboardAuthService) generateSessionToken(principal DashboardPrincipal) (*string, error) {
	token, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":     dashboardSubject,
		"exp":     time.Now().Add(time.Hour * 2).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "token",
		"userId":  principal.UserId,
		"email":   principal.Email,
		"isAdmin": principal.IsAdmin,
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &token, nil
}

// generateRefreshToken carries only the account's identity, not its isAdmin
// snapshot: RefreshSession re-resolves the account so a revoked flag or a
// deleted user takes effect at the next refresh instead of surviving 7 days.
func (a *DashboardAuthService) generateRefreshToken(principal DashboardPrincipal) (*string, error) {
	refreshToken, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":    dashboardSubject,
		"exp":    time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat":    time.Now().Unix(),
		"type":   "refreshToken",
		"userId": principal.UserId,
		"email":  principal.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &refreshToken, nil
}

func (a *DashboardAuthService) generateSessionPair(principal DashboardPrincipal) (*DashboardSession, error) {
	token, err := a.generateSessionToken(principal)
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.generateRefreshToken(principal)
	if err != nil {
		return nil, err
	}
	return &DashboardSession{
		Token:        *token,
		RefreshToken: *refreshToken,
	}, nil
}

// resolveStatelessPrincipal checks credentials against ADMIN_EMAIL and
// ADMIN_PASSWORD. When password is nil only the account's existence is
// resolved (the refresh path, where possession of the JWT is the credential).
func resolveStatelessPrincipal(email string, password *string) (*DashboardPrincipal, error) {
	adminEmail := store.NormalizeEmail(config.GetEnv("ADMIN_EMAIL"))
	if adminEmail == "" {
		return nil, ErrAdminEmailNotSet
	}
	adminPassword := config.GetEnv("ADMIN_PASSWORD")
	if adminPassword == "" {
		return nil, errors.New("admin password is not set, all requests will be rejected")
	}
	if store.NormalizeEmail(email) != adminEmail {
		return nil, errors.New("invalid credentials")
	}
	if password != nil && *password != adminPassword {
		return nil, errors.New("invalid credentials")
	}
	return &DashboardPrincipal{Email: adminEmail, IsAdmin: true}, nil
}

// unknownUserPasswordHash is a throwaway bcrypt hash checked when a login
// names an email with no account, so unknown and known emails cost the same
// bcrypt comparison and response timing cannot enumerate accounts.
const unknownUserPasswordHash = "$2a$10$RTxsxJsH5d9yZcM.fDe/kOv28rciQYAnNBOrK0frmWJPZGH1pTzhO"

// ErrAuthUnavailable marks login/refresh failures caused by the account store
// being unreachable — an infrastructure problem the handlers must surface as
// a 500, never as invalid credentials.
var ErrAuthUnavailable = errors.New("could not verify the account against the database")

// principalForUser records the connection and builds the session principal.
// Both a password login and a refresh prove the account is actively using the
// dashboard. Best-effort: a failed touch must not fail the sign-in.
// principalForUser is the single choke point every database-backed sign-in
// path goes through (password login, SSO callback, token refresh), which is
// why the enabled check lives here: no path can acquire a principal without
// passing it. On the refresh path this is also what makes disabling an account
// effective, since a live session token is never re-read against the database.
func (a *DashboardAuthService) principalForUser(ctx context.Context, user store.User) (*DashboardPrincipal, error) {
	if !user.Enabled {
		return nil, ErrAccountPendingApproval
	}
	if err := a.userRepo.TouchUserLastConnected(ctx, user.Id); err != nil {
		log.Printf("failed to record last connection for user %s: %v", user.Id, err)
	}
	return &DashboardPrincipal{UserId: user.Id, Email: user.Email, IsAdmin: user.IsAdmin}, nil
}

func (a *DashboardAuthService) resolveLoginPrincipal(ctx context.Context, email string, password string) (*DashboardPrincipal, error) {
	if a.userRepo == nil {
		return resolveStatelessPrincipal(email, &password)
	}
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			// Unknown account: burn the same bcrypt cost as a real comparison
			// so response timing cannot enumerate which emails exist.
			crypto.VerifyPassword(unknownUserPasswordHash, password)
			return nil, errors.New("invalid credentials")
		}
		return nil, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	if user.PasswordHash == "" {
		// SSO-provisioned accounts carry an empty hash and can never sign in
		// with a password. bcrypt would reject "" instantly, which would make
		// them enumerable by timing; burn the same cost as any wrong password.
		crypto.VerifyPassword(unknownUserPasswordHash, password)
		return nil, errors.New("invalid credentials")
	}
	if !crypto.VerifyPassword(user.PasswordHash, password) {
		return nil, errors.New("invalid credentials")
	}
	// Checked only after the password verified, so wrong-password responses
	// keep a uniform timing and message whether SSO is enforced or not.
	if a.ssoEnforced != nil && a.ssoEnforced(ctx) && !user.IsAdmin {
		return nil, ErrPasswordLoginDisabledBySSO
	}
	return a.principalForUser(ctx, user)
}

// resolveRefreshPrincipal re-resolves the account behind a refresh token by
// its immutable user id — never by email: a deleted account whose address is
// later reused must not let the old refresh token resurrect into the new one.
func (a *DashboardAuthService) resolveRefreshPrincipal(ctx context.Context, userId string) (*DashboardPrincipal, error) {
	if userId == "" {
		return nil, errors.New("invalid token")
	}
	user, err := a.userRepo.GetUserByID(ctx, userId)
	if err != nil {
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			return nil, errors.New("invalid credentials")
		}
		return nil, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	return a.principalForUser(ctx, user)
}

func (a *DashboardAuthService) LoginWithEmailPassword(ctx context.Context, email string, password string) (*DashboardSession, error) {
	principal, err := a.resolveLoginPrincipal(ctx, email, password)
	if err != nil {
		a.recordLoginFailure(ctx, email, err)
		return nil, err
	}
	a.recordLoginSuccess(ctx, principal)
	return a.generateSessionPair(*principal)
}

// SetOnAuditEvent plugs the audit emission seam (see SetSSOEnforced for the
// pattern; the enterprise wiring passes ee/audit's Record method value).
// Nil-safe: without it, sign-ins simply leave no audit events.
func (a *DashboardAuthService) SetOnAuditEvent(record auditlog.RecordFunc) {
	a.onAuditEvent = record
}

func (a *DashboardAuthService) recordLoginSuccess(ctx context.Context, principal *DashboardPrincipal) {
	if a.onAuditEvent == nil {
		return
	}
	a.onAuditEvent(ctx, auditlog.Event{
		ActorType:     auditlog.ActorUser,
		ActorID:       principal.UserId,
		ActorDisplay:  principal.Email,
		Action:        auditlog.ActionUserLogin,
		TargetType:    "user",
		TargetID:      principal.UserId,
		TargetDisplay: principal.Email,
		Outcome:       auditlog.OutcomeSuccess,
	})
}

// recordLoginFailure records rejected credentials, the brute-force signal a
// security review asks for. Infrastructure failures (database down, missing
// ADMIN_EMAIL) are not sign-in attempts and stay out of the log.
func (a *DashboardAuthService) recordLoginFailure(ctx context.Context, email string, err error) {
	if a.onAuditEvent == nil || errors.Is(err, ErrAuthUnavailable) || errors.Is(err, ErrAdminEmailNotSet) {
		return
	}
	reason := "invalid_credentials"
	switch {
	case errors.Is(err, ErrPasswordLoginDisabledBySSO):
		reason = "sso_enforced"
	case errors.Is(err, ErrAccountPendingApproval):
		reason = "pending_approval"
	}
	// The account may not exist: the attempted email is the only identity a
	// failure can carry, so the actor id stays empty.
	a.onAuditEvent(ctx, auditlog.Event{
		ActorType:     auditlog.ActorUser,
		ActorDisplay:  email,
		Action:        auditlog.ActionUserLogin,
		TargetType:    "user",
		TargetDisplay: email,
		Outcome:       auditlog.OutcomeFailure,
		Metadata:      map[string]any{"reason": reason},
	})
}

// IssueSession mints the standard dashboard JWT pair for an account that was
// authenticated by other means than a password (the enterprise SSO callback).
// It only exists in control-plane mode, where user is always a database row.
func (a *DashboardAuthService) IssueSession(ctx context.Context, user store.User) (*DashboardSession, error) {
	if a.userRepo == nil {
		return nil, errors.New("sessions can only be issued for database-backed accounts")
	}
	principal, err := a.principalForUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return a.generateSessionPair(*principal)
}

// ValidateSession accepts only a dashboard session JWT — not a refresh token,
// and not any other JWT signed with the same secret — and returns who it was
// minted for. Identity is read from the claims, not the database: a session
// token lives 2h and admin-gated routes re-check the flag against the DB.
func (a *DashboardAuthService) ValidateSession(tokenString string) (*DashboardPrincipal, error) {
	claims := jwt.MapClaims{}
	_, err := crypto.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
	if err != nil {
		return nil, err
	}
	if claims["type"] != "token" {
		return nil, errors.New("invalid token type")
	}
	if claims["sub"] != dashboardSubject {
		return nil, errors.New("invalid token subject")
	}
	principal := DashboardPrincipal{}
	if userId, ok := claims["userId"].(string); ok {
		principal.UserId = userId
	}
	if email, ok := claims["email"].(string); ok {
		principal.Email = email
	}
	if isAdmin, ok := claims["isAdmin"].(bool); ok {
		principal.IsAdmin = isAdmin
	}
	return &principal, nil
}

// RefreshSession accepts only a dashboard refresh JWT and mints a fresh pair.
// The account is re-resolved first, so a user that was deleted — or, in
// stateless mode, an ADMIN_EMAIL that changed — cannot refresh its way back in.
func (a *DashboardAuthService) RefreshSession(ctx context.Context, tokenString string) (*DashboardSession, error) {
	claims := jwt.MapClaims{}
	_, err := crypto.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
	if err != nil {
		return nil, err
	}
	if claims["type"] != "refreshToken" {
		return nil, errors.New("invalid token type")
	}
	if claims["sub"] != dashboardSubject {
		return nil, errors.New("invalid token subject")
	}
	var principal *DashboardPrincipal
	if a.userRepo == nil {
		email, _ := claims["email"].(string)
		principal, err = resolveStatelessPrincipal(email, nil)
	} else {
		userId, _ := claims["userId"].(string)
		principal, err = a.resolveRefreshPrincipal(ctx, userId)
	}
	if err != nil {
		return nil, err
	}
	return a.generateSessionPair(*principal)
}
