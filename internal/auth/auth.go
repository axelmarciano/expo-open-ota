package auth

import (
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Auth struct {
	Secret string
}

func getAdminPassword() string {
	return config.GetEnv("ADMIN_PASSWORD")
}

func isPasswordValid(password string) bool {
	adminPassword := getAdminPassword()
	if adminPassword == "" {
		fmt.Errorf("admin password is not set, all requests will be rejected")
		return false
	}
	return password == getAdminPassword()
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func NewAuth() *Auth {
	return &Auth{Secret: config.GetEnv("JWT_SECRET")}
}

func (a *Auth) generateAuthToken() (*string, error) {
	token, err := services.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":  "admin-dashboard",
		"exp":  time.Now().Add(time.Hour * 2).Unix(),
		"iat":  time.Now().Unix(),
		"type": "token",
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &token, nil
}

func (a *Auth) generateRefreshToken() (*string, error) {
	refreshToken, err := services.GenerateJWTToken(a.Secret, jwt.MapClaims{
		"sub":  "admin-dashboard",
		"exp":  time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat":  time.Now().Unix(),
		"type": "refreshToken",
	})
	if err != nil {
		return nil, fmt.Errorf("error while generating the jwt token: %w", err)
	}
	return &refreshToken, nil
}

func (a *Auth) LoginWithPassword(password string) (*AuthResponse, error) {
	if !isPasswordValid(password) {
		return nil, errors.New("invalid password")
	}
	token, err := a.generateAuthToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token:        *token,
		RefreshToken: *refreshToken,
	}, nil
}

func (a *Auth) ValidateToken(tokenString string) (*jwt.Token, error) {
	claims := jwt.MapClaims{}
	token, err := services.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
	if err != nil {
		return nil, err
	}
	if claims["type"] != "token" {
		return nil, errors.New("invalid token type")
	}
	if claims["sub"] != "admin-dashboard" {
		return nil, errors.New("invalid token subject")
	}
	return token, nil
}

func (a *Auth) RefreshToken(tokenString string) (*AuthResponse, error) {
	claims := jwt.MapClaims{}
	_, err := services.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
	if err != nil {
		return nil, err
	}
	if claims["type"] != "refreshToken" {
		return nil, errors.New("invalid token type")
	}
	if claims["sub"] != "admin-dashboard" {
		return nil, errors.New("invalid token subject")
	}
	newToken, err := a.generateAuthToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		Token:        *newToken,
		RefreshToken: *refreshToken,
	}, nil
}
