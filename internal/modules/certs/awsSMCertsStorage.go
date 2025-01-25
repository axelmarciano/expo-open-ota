package certs

import "expo-open-ota/internal/services"

type AWSSMCertsStorage struct {
	publicKeySecretID            string
	privateKeySecretID           string
	privateCloudfrontKeySecretID string
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

func (c *AWSSMCertsStorage) GetPrivateCloudfrontCert() string {
	if c.privateCloudfrontKeySecretID == "" {
		return ""
	}
	return services.FetchSecret(c.privateCloudfrontKeySecretID)
}
