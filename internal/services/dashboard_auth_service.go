package services

import (
	"context"
	"errors"
	"expo-open-ota/config"
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

func (a *DashboardAuthService) resolvePrincipal(ctx context.Context, email string, password *string) (*DashboardPrincipal, error) {
	if a.userRepo == nil {
		return resolveStatelessPrincipal(email, password)
	}
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if password != nil {
			crypto.VerifyPassword(unknownUserPasswordHash, *password)
		}
		return nil, errors.New("invalid credentials")
	}
	if password != nil && !crypto.VerifyPassword(user.PasswordHash, *password) {
		return nil, errors.New("invalid credentials")
	}
	// Both a password login and a refresh prove the account is actively using
	// the dashboard. Best-effort: a failed touch must not fail the sign-in.
	if err := a.userRepo.TouchUserLastConnected(ctx, user.Id); err != nil {
		log.Printf("failed to record last connection for user %s: %v", user.Id, err)
	}
	return &DashboardPrincipal{UserId: user.Id, Email: user.Email, IsAdmin: user.IsAdmin}, nil
}

func (a *DashboardAuthService) LoginWithEmailPassword(ctx context.Context, email string, password string) (*DashboardSession, error) {
	principal, err := a.resolvePrincipal(ctx, email, &password)
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
	email, _ := claims["email"].(string)
	principal, err := a.resolvePrincipal(ctx, email, nil)
	if err != nil {
		return nil, err
	}
	return a.generateSessionPair(*principal)
}
