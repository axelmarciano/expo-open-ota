package cdn

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"fmt"
	"time"
)

type GCSDirectCDN struct{}

func (c *GCSDirectCDN) isCDNAvailable() bool {
	return config.GetEnv("STORAGE_MODE") == "gcs" && config.GetEnv("GCS_BUCKET_NAME") != "" && config.GetEnv("GOOGLE_APPLICATION_CREDENTIALS_B64") != ""
}

func (c *GCSDirectCDN) ComputeRedirectionURLForAsset(appId, branch, runtimeVersion, updateId, asset string) (string, error) {
	bucket := config.GetEnv("GCS_BUCKET_NAME")
	// Same layout as the bucket backend: {appId}/{branch}/{rv}/{updateId}/{asset}.
	key := fmt.Sprintf("%s/%s/%s/%s/%s", appId, branch, runtimeVersion, updateId, asset)
	return services.GCSSignedURL(bucket, key, "GET", "", 15*time.Minute)
}
