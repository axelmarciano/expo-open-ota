package cdn

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"fmt"
	"os"
	"time"
)

type GCSDirectCDN struct{}

func (c *GCSDirectCDN) isCDNAvailable() bool {
	return config.GetEnv("STORAGE_MODE") == "gcs" && config.GetEnv("GCS_BUCKET_NAME") != "" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_B64") != ""
}

func (c *GCSDirectCDN) ComputeRedirectionURLForAsset(branch, runtimeVersion, updateId, asset string) (string, error) {
	bucket := config.GetEnv("GCS_BUCKET_NAME")
	key := fmt.Sprintf("%s/%s/%s/%s", branch, runtimeVersion, updateId, asset)
	return services.GCSSignedURL(bucket, key, "GET", "", 15*time.Minute)
}
