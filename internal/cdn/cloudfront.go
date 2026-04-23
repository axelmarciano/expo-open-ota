package cdn

import (
	"bytes"
	"crypto"
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
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
	reader := bytes.NewReader([]byte(key))
	privateKey, err := sign.LoadPEMPrivKeyPKCS8AsSigner(reader)
	if err != nil {
		privateKey, err = sign.LoadPEMPrivKey(reader)
		if err != nil {
			return nil, fmt.Errorf("error parsing private key: %w", err)
		}
	}
	return privateKey, nil
}

func (c *CloudfrontCDN) ComputeRedirectionURLForAsset(appId, branch, runtimeVersion, updateId, asset string) (string, error) {
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

	// Must match the v3 bucket layout exactly — if the CloudFront origin is
	// an S3 bucket, the object sits at {BUCKET_KEY_PREFIX}{appId}/{branch}/…
	// Operators using BUCKET_KEY_PREFIX must NOT also configure a
	// CloudFront Origin Path equal to the prefix; the path is part of the
	// signed resource and would be applied twice.
	endpoint := bucket.ResolveKeyPrefix() + fmt.Sprintf("%s/%s/%s/%s/%s", appId, branch, runtimeVersion, updateId, asset)
	resource := fmt.Sprintf("%s/%s", domain, endpoint)

	policy := sign.NewCannedPolicy(resource, time.Now().Add(10*time.Minute))
	signer := sign.NewURLSigner(keyPairId, privateKey)
	signedUrl, err := signer.SignWithPolicy(resource, policy)
	return signedUrl, err
}
