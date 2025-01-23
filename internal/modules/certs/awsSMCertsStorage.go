package certs

import "expo-open-ota/internal/services"

type AWSSMCertsStorage struct {
	publicKeySecretID  string
	privateKeySecretID string
}

func (c *AWSSMCertsStorage) GetPublicExpoCert() string {
	if c.publicKeySecretID == "" {
		return ""
	}
	return services.FetchSecret(c.publicKeySecretID)
}

func (c *AWSSMCertsStorage) GetPrivateExpoCert() string {
	if c.privateKeySecretID == "" {
		return ""
	}
	return services.FetchSecret(c.privateKeySecretID)
}
