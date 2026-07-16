package keyStore

import (
	"encoding/base64"
	"expo-open-ota/config"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/providers/aws"
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
func GetPublicExpoKey(app config.AppConfig) string {
	return readExpoKey(app.Keys, true)
}

// GetPrivateExpoKey mirrors GetPublicExpoKey for the private half.
func GetPrivateExpoKey(app config.AppConfig) string {
	return readExpoKey(app.Keys, false)
}

// ReadDBKeysMasterKey retrieves the 32-byte symmetric master key as a raw
// binary string. To avoid the logistical overhead and lack of native OpenSSL standard
// commands for symmetric PEM files, the secret is strictly managed as a standard 44-character Base64 text string.
// It decodes the payload at runtime into its 32 binary characters, supporting both production
// (AWS Secrets Manager) and local development (flat environment variables) workflows.
func ReadDBKeysMasterKey() string {
	if secretId := config.GetEnv("AWSSM_DB_KEYS_MASTER_KEY_SECRET_ID"); secretId != "" {
		b64Secret := aws.FetchSecret(secretId)
		if b64Secret == "" {
			return ""
		}
		return decodeB64(b64Secret)
	}
	if b64Env := config.GetEnv("DB_KEYS_MASTER_KEY_B64"); b64Env != "" {
		return decodeB64(b64Env)
	}
	return ""
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
			return aws.FetchSecret(k.PublicSecretId)
		}
		return aws.FetchSecret(k.PrivateSecretId)
	case config.KeysModeEnvironment:
		if public {
			return decodeB64(k.PublicB64)
		}
		return decodeB64(k.PrivateB64)
	case config.KeysModeDatabase:
		if public {
			key, err := crypto.UnsealAESGCM(k.SealedPublicKey, []byte(ReadDBKeysMasterKey()))
			if err != nil {
				log.Printf("Failed to unseal sealed public key: %v", err)
				return ""
			}
			return string(key)
		}
		key, err := crypto.UnsealAESGCM(k.SealedPrivateKey, []byte(ReadDBKeysMasterKey()))
		if err != nil {
			log.Printf("Failed to unseal sealed private key: %v", err)
			return ""
		}
		return string(key)
	}
	return ""
}

// GetPrivateCloudfrontKey is deployment-global (one CDN per server) so it
// stays on plain env vars and is not part of the per-app config. Supported
// sources, in priority order: AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID,
// PRIVATE_CLOUDFRONT_KEY_B64, PRIVATE_CLOUDFRONT_KEY_PATH.
func GetPrivateCloudfrontKey() string {
	if secretId := config.GetEnv("AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"); secretId != "" {
		return aws.FetchSecret(secretId)
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
