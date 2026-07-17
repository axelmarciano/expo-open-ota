package services

import (
	"context"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/validation"
	"fmt"
	"strconv"
	"time"
)

var ErrUnauthorized = fmt.Errorf("unauthorized")

// CliAuthRepository validates the credential a CLI client presents for an app,
// and stores the API keys that back it. ValidateCliCredential means a different
// thing per mode, which is why it is the repository's job and not the service's:
//   - Postgres (DB mode): hashes the presented eoo_ key and looks it up in the
//     api_keys table, scoped to appId.
//   - Bucket (stateless mode): no API keys exist; the credential is an Expo
//     token or session, verified against the Expo API.
type CliAuthRepository interface {
	ValidateCliCredential(ctx context.Context, appId string, auth types.Auth) error
	InsertApiKey(ctx context.Context, appId string, name string, hint string, hashedKey string) error
	GetApiKeysMetadataByAppID(ctx context.Context, appId string) ([]pgdb.GetApiKeysMetadataByAppIDRow, error)
	RevokeApiKeyByID(ctx context.Context, apiKeyId int64, appId string) error
}

// CliAuthService authenticates CLI clients (eoas) against a given app and
// manages the API keys backing that access in DB mode. The dashboard's own
// login and session tokens are a separate concern — see DashboardAuthService.
type CliAuthService struct {
	authRepo CliAuthRepository
}

func NewCliAuthService(authRepo CliAuthRepository) *CliAuthService {
	return &CliAuthService{
		authRepo: authRepo,
	}
}

func (s *CliAuthService) ValidateCliCredential(ctx context.Context, appId string, auth types.Auth) error {
	err := s.authRepo.ValidateCliCredential(ctx, appId, auth)
	if err != nil {
		return fmt.Errorf("failed to validate auth: %w", err)
	}
	return nil
}

func (s *CliAuthService) GenerateAPIKey(ctx context.Context, appId string, name string) (string, error) {
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

func (s *CliAuthService) GetApiKeysMetadata(ctx context.Context, appId string) ([]types.ApiKeyMetadata, error) {
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

func (s *CliAuthService) RevokeApiKey(ctx context.Context, appId string, apiKeyId string) error {
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
