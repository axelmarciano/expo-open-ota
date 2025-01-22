package certs

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
)

func GetPublicExpoCert() string {
	publicKeySecretID := config.GetEnv("PUBLIC_KEY_SECRET_ID")
	return services.FetchSecret(publicKeySecretID)
}

func GetPrivateExpoCert() string {
	privateKeySecretId := config.GetEnv("PRIVATE_KEY_SECRET_ID")
	return services.FetchSecret(privateKeySecretId)
}
