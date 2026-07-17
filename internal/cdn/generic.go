package cdn

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	"net/url"
	"strings"
)

type GenericCDN struct{}

func (c *GenericCDN) isCDNAvailable() bool {
	return config.GetEnv("STORAGE_MODE") == "s3" && config.GetEnv("S3_BUCKET_NAME") != "" && config.GetEnv("S3_CDN_PREFIX") != ""
}

func (c *GenericCDN) ComputeRedirectionURLForAsset(appId, branch, runtimeVersion, updateId, asset string) (string, error) {
	prefix := config.GetEnv("S3_CDN_PREFIX")
	keyPrefix := strings.TrimSuffix(bucket.ResolveKeyPrefix(), "/")
	elems := []string{appId, branch, runtimeVersion, updateId, asset}
	if keyPrefix != "" {
		elems = append([]string{keyPrefix}, elems...)
	}
	cdnUrl, err := url.JoinPath(prefix, elems...)
	if err != nil {
		return "", err
	}
	return cdnUrl, nil
}
