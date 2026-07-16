package services

import (
	"context"
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/validation"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = fmt.Errorf("unauthorized")

type AuthRepository interface {
	ValidateAuth(ctx context.Context, appId string, auth types.Auth) error
	InsertApiKey(ctx context.Context, appId string, name string, hint string, hashedKey string) error
	GetApiKeysMetadataByAppID(ctx context.Context, appId string) ([]pgdb.GetApiKeysMetadataByAppIDRow, error)
	RevokeApiKeyByID(ctx context.Context, apiKeyId int64, appId string) error
}

type AuthService struct {
	authRepo AuthRepository
	Secret   string
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func NewAuthService(authRepo AuthRepository) *AuthService {
	return &AuthService{
		authRepo: authRepo,
		Secret:   config.GetEnv("JWT_SECRET"),
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

func (a *AuthService) generateAuthToken() (*string, error) {
	token, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
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

func (a *AuthService) generateRefreshToken() (*string, error) {
	refreshToken, err := crypto.GenerateJWTToken(a.Secret, jwt.MapClaims{
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

func (a *AuthService) LoginWithPassword(password string) (*AuthResponse, error) {
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

func (a *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	claims := jwt.MapClaims{}
	token, err := crypto.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
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

func (a *AuthService) RefreshToken(tokenString string) (*AuthResponse, error) {
	claims := jwt.MapClaims{}
	_, err := crypto.DecodeAndExtractJWTToken(a.Secret, tokenString, &claims)
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

func (s *AuthService) ValidateAuth(ctx context.Context, appId string, auth types.Auth) error {
	err := s.authRepo.ValidateAuth(ctx, appId, auth)
	if err != nil {
		return fmt.Errorf("failed to validate auth: %w", err)
	}
	return nil
}

func (s *AuthService) GenerateAPIKey(ctx context.Context, appId string, name string) (string, error) {
	if err := validation.DisplayName("name", name); err != nil {
		return "", err
	}
	apiKey, err := crypto.GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	hashedKey, err := crypto.HashPlaintextAPIKey(apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	lastFour := apiKey[len(apiKey)-4:]
	hint := fmt.Sprintf("%s*******%s", crypto.PrefixActive, lastFour)
	// Store only the hashed version of the API key in the database for security reasons
	err = s.authRepo.InsertApiKey(ctx, appId, name, hint, hashedKey)
	if err != nil {
		return "", fmt.Errorf("failed to insert API key into database: %w", err)
	}
	return apiKey, nil
}

func (s *AuthService) GetApiKeysMetadata(ctx context.Context, appId string) ([]types.ApiKeyMetadata, error) {
	rows, err := s.authRepo.GetApiKeysMetadataByAppID(ctx, appId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve API keys metadata from database: %w", err)
	}
	apiKeysMetadata := make([]types.ApiKeyMetadata, len(rows))
	for i, row := range rows {
		var lastUsedAtStr *string
		if row.LastUsedAt.Valid {
			timeStr := row.LastUsedAt.Time.Format(time.RFC3339)
			lastUsedAtStr = &timeStr
		}
		apiKeysMetadata[i] = types.ApiKeyMetadata{
			ID:         strconv.FormatInt(row.ID, 10),
			Name:       row.Name,
			Hint:       row.Hint,
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
			LastUsedAt: lastUsedAtStr,
		}
	}
	return apiKeysMetadata, nil
}

func (s *AuthService) RevokeApiKey(ctx context.Context, appId string, apiKeyId string) error {
	if err := validation.NumericID("apiKeyId", apiKeyId); err != nil {
		return err
	}
	apiKeyIdInt, err := strconv.ParseInt(apiKeyId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid API key ID: %w", err)
	}
	err = s.authRepo.RevokeApiKeyByID(ctx, apiKeyIdInt, appId)
	if err != nil {
		return err
	}
	return nil
}
