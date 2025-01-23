package certs

import (
	"expo-open-ota/config"
	"fmt"
)

type CertsStorageType string

const (
	AWSSecretsManager CertsStorageType = "aws-secrets-manager"
	LocalFiles        CertsStorageType = "local-files"
)

type CertsStorage interface {
	GetPublicExpoCert() string
	GetPrivateExpoCert() string
}

func getStorage() (CertsStorage, error) {
	var storageType CertsStorageType
	if config.GetEnv("CERTS_STORAGE_TYPE") == "aws-secrets-manager" {
		storageType = AWSSecretsManager
	} else {
		storageType = LocalFiles
	}

	switch storageType {
	case AWSSecretsManager:
		publicKeySecretID := config.GetEnv("AWS_CERTS_PUBLIC_KEY_SECRET_ID")
		privateKeySecretID := config.GetEnv("AWS_CERTS_PRIVATE_KEY_SECRET_ID")
		if publicKeySecretID == "" || privateKeySecretID == "" {
			return nil, fmt.Errorf("PUBLIC_KEY_SECRET_ID and PRIVATE_KEY_SECRET_ID must be set in environment")
		}
		return &AWSSMCertsStorage{
			publicKeySecretID:  publicKeySecretID,
			privateKeySecretID: privateKeySecretID,
		}, nil
	case LocalFiles:
		publicKeyPath := config.GetEnv("PUBLIC_CERT_KEY_PATH")
		privateKeyPath := config.GetEnv("PRIVATE_CERT_KEY_PATH")
		if publicKeyPath == "" || privateKeyPath == "" {
			return nil, fmt.Errorf("PUBLIC_KEY_PATH and PRIVATE_KEY_PATH must be set in environment")
		}
		return &LocalCertsStorage{
			publicKeyPath:  publicKeyPath,
			privateKeyPath: privateKeyPath,
		}, nil
	default:
		return nil, fmt.Errorf("unknown certs storage type: %s", storageType)
	}
}

func GetPublicExpoCert() string {
	storage, err := getStorage()
	if err != nil {
		return ""
	}
	return storage.GetPublicExpoCert()
}

func GetPrivateExpoCert() string {
	storage, err := getStorage()
	if err != nil {
		return ""
	}
	return storage.GetPrivateExpoCert()
}
