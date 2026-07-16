package cdn

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/providers/gcp"
	"fmt"
	"time"
)

type GCSDirectCDN struct{}

func (c *GCSDirectCDN) isCDNAvailable() bool {
	return config.GetEnv("STORAGE_MODE") == "gcs" && config.GetEnv("GCS_BUCKET_NAME") != "" && config.GetEnv("GOOGLE_APPLICATION_CREDENTIALS_B64") != ""
}

func (c *GCSDirectCDN) ComputeRedirectionURLForAsset(appId, branch, runtimeVersion, updateId, asset string) (string, error) {
	bucketName := config.GetEnv("GCS_BUCKET_NAME")
	// Must match the full object key that the bucket backend writes:
	// {BUCKET_KEY_PREFIX}{appId}/{branch}/{rv}/{updateId}/{asset}. Dropping
	// the prefix here points the signed URL at a non-existent object.
	key := bucket.ResolveKeyPrefix() + fmt.Sprintf("%s/%s/%s/%s/%s", appId, branch, runtimeVersion, updateId, asset)
	return gcp.SignedURL(bucketName, key, "GET", "", 15*time.Minute)
}
