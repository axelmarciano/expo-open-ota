package services

import (
	"context"
	"crypto/sha256"
	"expo-open-ota/config"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/keyStore"
	"expo-open-ota/internal/store"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

type AppService struct {
	appRepo AppRepository
}

type AppRepository interface {
	InsertApp(ctx context.Context, app store.InsertAppParameters) (string, error)
	DeleteAppByID(ctx context.Context, id string) error
	GetApps(ctx context.Context) ([]config.AppDescriptor, error)
	UpdateAppNameByID(ctx context.Context, id string, newName string) error
	GetAppByID(ctx context.Context, id string) (config.AppConfig, error)
}

func NewAppService(appRepo AppRepository) *AppService {
	return &AppService{
		appRepo: appRepo,
	}
}

func (s *AppService) CreateApp(ctx context.Context, displayName string, keysConfig config.KeysConfig) (string, error) {
	// Enforce presence of keys in environment variables for the environment keys mode for ValidateKeys function
	if keysConfig.Mode == config.KeysModeEnvironment {
		keysConfig.PublicB64 = config.GetEnv("PUBLIC_EXPO_KEY_B64")
		keysConfig.PrivateB64 = config.GetEnv("PRIVATE_EXPO_KEY_B64")
	}
	if err := config.ValidateKeys(&keysConfig, "keysConfig"); err != nil {
		return "", fmt.Errorf("invalid keys configuration: %w", err)
	}
	appId := uuid.New()
	modeStr := string(keysConfig.Mode)
	params := store.InsertAppParameters{
		ID:       appId.String(),
		Name:     displayName,
		KeysMode: &modeStr,
	}
	switch keysConfig.Mode {
	case config.KeysModeDatabase:
		masterKeyStr := keyStore.ReadControlPlaneMasterKey()
		if masterKeyStr == "" {
			return "", fmt.Errorf("master key is required for database keys mode")
		}
		masterKeyBytes := []byte(masterKeyStr)
		if masterKeyBytes == nil {
			return "", fmt.Errorf("invalid base64 configuration for master key")
		}
		if len(masterKeyBytes) != 32 {
			return "", fmt.Errorf("decoded master key must be exactly 32 bytes (got %d)", len(masterKeyBytes))
		}
		pubPEM, privPEM, err := crypto.GenerateRSAKeyPair()
		if err != nil {
			return "", fmt.Errorf("failed to generate application signing keys: %w", err)
		}
		sealedPublicKey, err := crypto.SealAESGCM([]byte(pubPEM), masterKeyBytes)
		if err != nil {
			return "", fmt.Errorf("failed to seal public key: %w", err)
		}
		sealedPrivateKey, err := crypto.SealAESGCM([]byte(privPEM), masterKeyBytes)
		if err != nil {
			return "", fmt.Errorf("failed to seal private key: %w", err)
		}
		params.SealedPublicKey = &sealedPublicKey
		params.SealedPrivateKey = &sealedPrivateKey

	case config.KeysModeLocal:
		params.PathPublicKey = &keysConfig.PublicPath
		params.PathPrivateKey = &keysConfig.PrivatePath

	case config.KeysModeAWSSM:
		params.AwsSecretIDPublic = &keysConfig.PublicSecretId
		params.AwsSecretIDPrivate = &keysConfig.PrivateSecretId

	case config.KeysModeEnvironment:
		// Replaced by database keystore mode
	default:
		return "", fmt.Errorf("invalid keys mode %q", keysConfig.Mode)
	}

	insertedAppId, err := s.appRepo.InsertApp(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to save app record to database: %w", err)
	}
	return insertedAppId, nil
}

func (s *AppService) DeleteApp(ctx context.Context, appId string) error {
	err := s.appRepo.DeleteAppByID(ctx, appId)
	if err != nil {
		return err
	}
	return nil
}

func (s *AppService) GetApps(ctx context.Context) ([]config.AppDescriptor, error) {
	return s.appRepo.GetApps(ctx)
}

func (s *AppService) GetAppByID(ctx context.Context, appId string) (config.AppConfig, error) {
	app, err := s.appRepo.GetAppByID(ctx, appId)
	if err != nil {
		return config.AppConfig{}, err
	}
	app.AccessToken = ""
	app.Keys.SealedPrivateKey = ""
	app.Keys.SealedPublicKey = ""
	app.Keys.PrivateB64 = ""
	app.Keys.PublicB64 = ""
	if app.Keys.Mode == config.KeysModeLocal {
		app.Keys.PrivatePath = helpers.MaskKeyPath(app.Keys.PrivatePath)
		app.Keys.PublicPath = helpers.MaskKeyPath(app.Keys.PublicPath)
	}
	return app, nil
}

func (s *AppService) UpdateApp(ctx context.Context, appId string, newName string) error {
	err := s.appRepo.UpdateAppNameByID(ctx, appId, newName)
	if err != nil {
		return err
	}
	return nil
}

func (s *AppService) RetrieveAppCertificate(ctx context.Context, appId string) (string, error) {
	app, err := s.appRepo.GetAppByID(ctx, appId)
	if err != nil {
		return "", err
	}
	if app.Keys.Mode != config.KeysModeDatabase {
		return "", fmt.Errorf("app with id %s does not use database keys mode", appId)
	}
	publicKey := keyStore.GetPublicExpoKey(app)
	privateKey := keyStore.GetPrivateExpoKey(app)
	// Deterministic serial number by hashing the public key so it never changes
	hash := sha256.Sum256([]byte(publicKey))
	serialNumber := new(big.Int).SetBytes(hash[:8])

	// 2. Deterministic Validity: Use the database app creation timestamp
	// Convert your 13-digit millisecond timestamp safely
	notBefore := time.UnixMilli(int64(app.CreatedAt)).UTC().Add(-1 * time.Hour)
	pemCertificateString, err := crypto.GenerateSelfSignedCodeSigningCertificateFromPEM(publicKey, privateKey, app.Name, serialNumber, notBefore)
	if err != nil {
		return "", fmt.Errorf("failed to wrap public key in self-signed certificate: %w", err)
	}
	return pemCertificateString, nil
}
