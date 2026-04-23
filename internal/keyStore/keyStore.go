package keyStore

import (
	"encoding/base64"
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"fmt"
	"io"
	"log"
	"os"
)

// GetPublicExpoKey returns the PEM-encoded Expo public signing key for the
// given app, resolving it from the app's KeysConfig (local file, AWS Secrets
// Manager, or inline b64). Returns "" on any read failure — callers that need
// to differentiate "not configured" from "read error" should use
// ReadExpoKey with the "public" selector.
func GetPublicExpoKey(appId string) string {
	app, err := config.GetAppConfig(appId)
	if err != nil {
		return ""
	}
	return readExpoKey(app.Keys, true)
}

// GetPrivateExpoKey mirrors GetPublicExpoKey for the private half.
func GetPrivateExpoKey(appId string) string {
	app, err := config.GetAppConfig(appId)
	if err != nil {
		return ""
	}
	return readExpoKey(app.Keys, false)
}

func readExpoKey(k config.KeysConfig, public bool) string {
	switch k.Mode {
	case config.KeysModeLocal:
		if public {
			return readPEMFile(k.PublicPath)
		}
		return readPEMFile(k.PrivatePath)
	case config.KeysModeAWSSM:
		if public {
			return services.FetchSecret(k.PublicSecretId)
		}
		return services.FetchSecret(k.PrivateSecretId)
	case config.KeysModeEnvironment:
		if public {
			return decodeB64(k.PublicB64)
		}
		return decodeB64(k.PrivateB64)
	}
	return ""
}

// GetPrivateCloudfrontKey is deployment-global (one CDN per server) so it
// stays on plain env vars and is not part of the per-app config. Supported
// sources, in priority order: AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID,
// PRIVATE_CLOUDFRONT_KEY_B64, PRIVATE_CLOUDFRONT_KEY_PATH.
func GetPrivateCloudfrontKey() string {
	if secretId := config.GetEnv("AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"); secretId != "" {
		return services.FetchSecret(secretId)
	}
	if b64 := config.GetEnv("PRIVATE_CLOUDFRONT_KEY_B64"); b64 != "" {
		return decodeB64(b64)
	}
	if path := config.GetEnv("PRIVATE_CLOUDFRONT_KEY_PATH"); path != "" {
		return readPEMFile(path)
	}
	return ""
}

func readPEMFile(path string) string {
	if path == "" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening key file:", err)
		return ""
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading key file:", err)
		return ""
	}
	return string(content)
}

func decodeB64(b64 string) string {
	if b64 == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.Printf("Failed to decode base64 key: %v", err)
		return ""
	}
	return string(decoded)
}
