package cdn

import (
	"bytes"
	"crypto"
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/keyStore"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"time"
)

type CloudfrontCDN struct{}

func getCloudfrontDomain() string {
	return config.GetEnv("CLOUDFRONT_DOMAIN")
}

func getCloudfrontKeyPairId() string {
	return config.GetEnv("CLOUDFRONT_KEY_PAIR_ID")
}

func (c *CloudfrontCDN) isCDNAvailable() bool {
	privateCloudfrontCert := keyStore.GetPrivateCloudfrontKey()
	domain := getCloudfrontDomain()
	keyPairId := getCloudfrontKeyPairId()
	return privateCloudfrontCert != "" && domain != "" && keyPairId != ""
}

func getSigner(key string) (crypto.Signer, error) {
	privateKey, err := sign.LoadPEMPrivKeyPKCS8AsSigner(bytes.NewReader([]byte(key)))
	if err != nil {
		// A fresh reader is required: the failed PKCS#8 attempt above has
		// already consumed the previous one, so reusing it would make this
		// fallback always fail with "no valid PEM data provided".
		privateKey, err = sign.LoadPEMPrivKey(bytes.NewReader([]byte(key)))
		if err != nil {
			return nil, fmt.Errorf("error parsing private key: %w", err)
		}
	}
	return privateKey, nil
}

func (c *CloudfrontCDN) ComputeRedirectionURLForAsset(branch, runtimeVersion, updateId, asset string) (string, error) {
	domain := getCloudfrontDomain()
	keyPairId := getCloudfrontKeyPairId()
	privateCloudfrontCert := keyStore.GetPrivateCloudfrontKey()

	if domain == "" || keyPairId == "" || privateCloudfrontCert == "" {
		return "", errors.New("CloudFront configuration is incomplete")
	}

	privateKey, err := getSigner(privateCloudfrontCert)
	if err != nil {
		return "", fmt.Errorf("error parsing private key: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s/%s/%s", branch, runtimeVersion, updateId, asset)
	resource := fmt.Sprintf("%s/%s", domain, endpoint)

	policy := sign.NewCannedPolicy(resource, time.Now().Add(10*time.Minute))
	signer := sign.NewURLSigner(keyPairId, privateKey)
	signedUrl, err := signer.SignWithPolicy(resource, policy)
	return signedUrl, err
}
