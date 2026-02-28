package test

import (
	"encoding/json"
	"expo-open-ota/internal/bucket"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/cdn"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/types"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func setup(t *testing.T) func() {
	GlobalBeforeEach()
	SetValidConfiguration()
	metrics.InitMetrics()
	return func() {
		GlobalAfterEach(t)
	}
}

func GlobalBeforeEach() {
	metrics.CleanupMetrics()
	cache := cache2.GetCache()
	_ = cache.Clear()
	newTime := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)

	ChangeModTimeRecursively(os.Getenv("LOCAL_BUCKET_BASE_PATH"), newTime)
}

func GlobalAfterEach(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		bucket.ResetBucketInstance()
		cdn.ResetCDNInstance()
		projectRoot, err := findProjectRoot()
		if err != nil {
			t.Errorf("Error finding project root: %v", err)
		}
		// Clean up .channels dir created during tests
		channelsPath := filepath.Join(projectRoot, "test/test-updates/.channels")
		os.RemoveAll(channelsPath)

		// Clean up branches created during tests (keep only fixture branches)
		fixtureBranches := map[string]bool{
			"branch-1": true,
			"branch-2": true,
			"branch-3": true,
			"branch-4": true,
		}
		testUpdatesPath := filepath.Join(projectRoot, "test/test-updates")
		entries, err := os.ReadDir(testUpdatesPath)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !fixtureBranches[entry.Name()] {
					os.RemoveAll(filepath.Join(testUpdatesPath, entry.Name()))
				}
			}
		}

		updatesPath := filepath.Join(projectRoot, "./updates/DO_NOT_USE")
		updates, err := os.ReadDir(updatesPath)
		if err != nil {
			t.Errorf("Error reading updates directory: %v", err)
		}
		for _, update := range updates {
			if update.IsDir() {
				err = os.RemoveAll(filepath.Join(updatesPath, update.Name()))
				if err != nil {
					t.Errorf("Error removing update directory: %v", err)
				}
			}
		}
		// Also remove all folders > 1674170951 in ./test/test-updates/branch-1/1
		updatesPath = filepath.Join(projectRoot, "./test/test-updates/branch-1/1")
		updates, err = os.ReadDir(updatesPath)
		if err != nil {
			t.Errorf("Error reading updates directory: %v", err)
		}
		for _, update := range updates {
			if update.IsDir() {
				updateTime, err := strconv.Atoi(update.Name())
				if err != nil {
					continue
				}
				if updateTime > 1674170951 {
					err = os.RemoveAll(filepath.Join(updatesPath, update.Name()))
					if err != nil {
						t.Errorf("Error removing update directory: %v", err)
					}
				}
			}
		}
	})

}

func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return "", os.ErrNotExist
}

func setupChannelMapping(channelName, branchName string) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		panic(err)
	}
	channelsDir := filepath.Join(projectRoot, "test/test-updates/.channels")
	os.MkdirAll(channelsDir, 0755)
	mapping := map[string]string{"branch": branchName}
	data, err := json.Marshal(mapping)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filepath.Join(channelsDir, channelName+".json"), data, 0644)
	if err != nil {
		panic(err)
	}
}

func ComputeUploadRequestsInput(dirPath string) handlers.FileNamesRequest {
	metadataFilePath := filepath.Join(dirPath, "metadata.json")
	metadataFile, err := os.Open(metadataFilePath)
	if err != nil {
		panic(err)
	}
	defer metadataFile.Close()
	var metadataObject types.MetadataObject
	err = json.NewDecoder(metadataFile).Decode(&metadataObject)
	if err != nil {
		panic(err)
	}
	fileNames := make([]string, 0)
	for _, asset := range metadataObject.FileMetadata.IOS.Assets {
		fileNames = append(fileNames, asset.Path)
	}
	for _, asset := range metadataObject.FileMetadata.Android.Assets {
		fileNames = append(fileNames, asset.Path)
	}
	if metadataObject.FileMetadata.Android.Bundle != "" {
		fileNames = append(fileNames, metadataObject.FileMetadata.Android.Bundle)
	}
	if metadataObject.FileMetadata.IOS.Bundle != "" {
		fileNames = append(fileNames, metadataObject.FileMetadata.IOS.Bundle)
	}
	// Add metadata.json & expoConfig.json
	fileNames = append(fileNames, "metadata.json")
	fileNames = append(fileNames, "expoConfig.json")
	return handlers.FileNamesRequest{FileNames: fileNames}
}

func ChangeModTime(filePath string, newTime time.Time) error {
	// Ouvre le fichier
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = os.Chtimes(filePath, newTime, newTime)
	if err != nil {
		return err
	}

	return nil
}

func ChangeModTimeRecursively(dir string, newTime time.Time) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			err := ChangeModTime(path, newTime)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func SetValidConfiguration() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		panic(err)
	}
	os.Setenv("BASE_URL", "http://localhost:3000")
	os.Setenv("PUBLIC_LOCAL_EXPO_KEY_PATH", filepath.Join(projectRoot, "/test/keys/public-key-test.pem"))
	os.Setenv("PRIVATE_LOCAL_EXPO_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	os.Setenv("EOAS_API_KEY", "EOAS_API_KEY")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", "")
	os.Setenv("CLOUDFRONT_DOMAIN", "")
	os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "")
	os.Setenv("USE_DASHBOARD", "true")
	os.Setenv("ADMIN_PASSWORD", "admin")
}
