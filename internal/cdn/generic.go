package cdn

import (
	"expo-open-ota/config"
	"fmt"
)

type GenericCDN struct{}

func (c *GenericCDN) isCDNAvailable() bool {
	return config.GetEnv("STORAGE_MODE") == "s3" && config.GetEnv("S3_BUCKET_NAME") != "" && config.GetEnv("S3_CDN_PREFIX") != ""
}

func (c *GenericCDN) ComputeRedirectionURLForAsset(branch, runtimeVersion, updateId, asset string) (string, error) {
	prefix := config.GetEnv("S3_CDN_PREFIX")
	url := fmt.Sprintf("%s/%s/%s/%s/%s", prefix, branch, runtimeVersion, updateId, asset)
	return url, nil
}
