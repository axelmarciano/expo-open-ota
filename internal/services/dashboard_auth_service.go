package services

import (
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/crypto"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DashboardAuthService owns the admin dashboard's own session credentials: a
// short-lived session JWT and a long-lived refresh JWT, both minted only after
// an ADMIN_PASSWORD login. It has no notion of apps and no repository — the
// credentials a CLI client presents for an app are a separate concern, see
// CliAuthService.
type DashboardAuthService struct {
	Secret string
}

// DashboardSession is the JWT pair handed to the dashboard on login or refresh.
type DashboardSession struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// dashboardSubject scopes every dashboard JWT to the dashboard itself. Both
// validators below reject any other subject, which is what keeps the upload
// tokens minted by localBucket — signed with the same JWT_SECRET — from being
// accepted here. Changing this value invalidates sessions already in the wild.
const dashboardSubject = "admin-dashboard"

func NewDashboardAuthService() *DashboardAuthService {
	return &DashboardAuthService{
		Secret: config.GetEnv("JWT_SECRET"),
	}
}

func getAdminPassword() string {
	return config.GetEnv("ADMIN_PASSWORD")
}

func isPasswordValid(password string) bool {
	adminPassword := getAdminPassword()
	if adminPassword == "" {
		fmt.Println("admin password is not set, all requests will be rejected")
		return false
	}
	return password == getAdminPassword()
}

func (a *DashboardAuthService) generateSessionToken() (*string, error) {
	token, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":  dashboardSubject,
		"exp":  time.Now().Add(time.Hour * 2).Unix(),
		"iat":  time.Now().Unix(),
		"type": "token",
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &token, nil
}

func (a *DashboardAuthService) generateRefreshToken() (*string, error) {
	refreshToken, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":  dashboardSubject,
		"exp":  time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat":  time.Now().Unix(),
		"type": "refreshToken",
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &refreshToken, nil
}

func (a *DashboardAuthService) LoginWithPassword(password string) (*DashboardSession, error) {
	if !isPasswordValid(password) {
		return nil, errors.New("invalid password")
	}
	token, err := a.generateSessionToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &DashboardSession{
		Token:        *token,
		RefreshToken: *refreshToken,
	}, nil
}

// ValidateSession accepts only a dashboard session JWT — not a refresh token,
// and not any other JWT signed with the same secret.
func (a *DashboardAuthService) ValidateSession(tokenString string) (*jwt.Token, error) {
	claims := jwt.MapClaims{}
	token, err := crypto.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
	if err != nil {
		return nil, err
	}
	if claims["type"] != "token" {
		return nil, errors.New("invalid token type")
	}
	if claims["sub"] != dashboardSubject {
		return nil, errors.New("invalid token subject")
	}
	return token, nil
}

// RefreshSession accepts only a dashboard refresh JWT and mints a fresh pair.
func (a *DashboardAuthService) RefreshSession(tokenString string) (*DashboardSession, error) {
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
	newToken, err := a.generateSessionToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, err
	}
	return &DashboardSession{
		Token:        *newToken,
		RefreshToken: *refreshToken,
	}, nil
}
