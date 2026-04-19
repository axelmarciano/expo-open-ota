package test

import (
	"encoding/json"
	"expo-open-ota/internal/bucket"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/cdn"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/types"
	"github.com/jarcoal/httpmock"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func setup(t *testing.T) func() {
	GlobalBeforeEach()
	httpmock.Activate()
	SetValidConfiguration()
	metrics.InitMetrics()
	return func() {
		GlobalAfterEach(t)
		defer httpmock.DeactivateAndReset()
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
		// Clean both legacy path (./updates/DO_NOT_USE) and v2 multi-app path
		// (./updates/test-app-id/DO_NOT_USE) — tests mix both depending on how
		// they set LOCAL_BUCKET_BASE_PATH.
		for _, updatesPath := range []string{
			filepath.Join(projectRoot, "./updates/DO_NOT_USE"),
			filepath.Join(projectRoot, "./updates/test-app-id/DO_NOT_USE"),
		} {
			updates, err := os.ReadDir(updatesPath)
			if err != nil {
				continue
			}
			for _, update := range updates {
				if update.IsDir() {
					err = os.RemoveAll(filepath.Join(updatesPath, update.Name()))
					if err != nil {
						t.Errorf("Error removing update directory: %v", err)
					}
				}
			}
		}
		// Also remove all folders > 1674170951 in ./test/test-updates/test-app-id/branch-1/1
		fixturePath := filepath.Join(projectRoot, "./test/test-updates/test-app-id/branch-1/1")
		fixtureUpdates, err := os.ReadDir(fixturePath)
		if err != nil {
			t.Errorf("Error reading updates directory: %v", err)
		}
		for _, update := range fixtureUpdates {
			if update.IsDir() {
				updateTime, err := strconv.Atoi(update.Name())
				if err != nil {
					continue
				}
				if updateTime > 1674170951 {
					err = os.RemoveAll(filepath.Join(fixturePath, update.Name()))
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

func MockExpoChannelMapping(updateBranches []map[string]interface{}, updateChannelByName map[string]interface{}) (*http.Response, error) {
	return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"app": map[string]interface{}{
				"byId": map[string]interface{}{
					"id":                  "EXPO_APP_ID",
					"updateBranches":      updateBranches,
					"updateChannelByName": updateChannelByName,
				},
			},
		},
	})
}

func MockExpoBranchesMappingResponse(updateBranches []map[string]interface{}, updateChannelByName []map[string]interface{}) (*http.Response, error) {
	return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"app": map[string]interface{}{
				"byId": map[string]interface{}{
					"id":             "EXPO_APP_ID",
					"updateBranches": updateBranches,
					"updateChannels": updateChannelByName,
				},
			},
		},
	})
}

func MockExpoBranchesResponse(updateBranches []map[string]interface{}) (*http.Response, error) {
	return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"app": map[string]interface{}{
				"byId": map[string]interface{}{
					"id":             "EXPO_APP_ID",
					"updateBranches": updateBranches,
				},
			},
		},
	})
}

func MockExpoAccountResponse(me map[string]interface{}) (*http.Response, error) {
	return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"me": me,
		},
	})
}

func StringifyBranchMapping(branchMapping map[string]interface{}) string {
	branchMappingString, err := json.Marshal(branchMapping)
	if err != nil {
		panic(err)
	}
	return string(branchMappingString)
}

func mockWorkingExpoResponse(channelName string) {
	httpmock.RegisterResponder("POST", "https://api.expo.dev/graphql",
		func(req *http.Request) (*http.Response, error) {
			isFetchSelfExpoUsername := req.Header.Get("operationName") == "FetchExpoUserAccountInformations"
			isFetchExpoChannelMapping := req.Header.Get("operationName") == "FetchExpoChannelMapping"
			isFetchBranches := req.Header.Get("operationName") == "FetchExpoBranches"
			isCreateBranch := req.Header.Get("operationName") == "CreateBranch"
			if isFetchBranches {
				return MockExpoBranchesResponse([]map[string]interface{}{
					{
						"id":   "branch-1-id",
						"name": "branch-1",
					},
					{
						"id":   "branch-2-id",
						"name": "branch-2",
					},
				})
			}
			if isCreateBranch {
				return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"data": map[string]interface{}{
						"updateBranch": map[string]interface{}{
							"createUpdateBranchForApp": map[string]interface{}{
								"id":   "created-branch-id",
								"name": "created-branch",
							},
						},
					},
				})
			}
			if isFetchSelfExpoUsername {
				return MockExpoAccountResponse(map[string]interface{}{
					"id":       "test_id",
					"username": "test_username",
					"email":    "test_email",
				})
			}
			if isFetchExpoChannelMapping {
				return MockExpoChannelMapping(
					[]map[string]interface{}{
						{
							"id":   "branch-1-id",
							"name": "branch-1",
						},
						{
							"id":   "branch-2-id",
							"name": "branch-2",
						},
					},
					map[string]interface{}{
						"id":   channelName + "-id",
						"name": channelName,
						"branchMapping": StringifyBranchMapping(map[string]interface{}{
							"version": 0,
							"data": []map[string]interface{}{
								{
									"branchId":           "branch-1-id",
									"branchMappingLogic": "true",
								},
								{
									"branchId":           "branch-2-id",
									"branchMappingLogic": "false",
								},
							},
						}),
					},
				)
			}

			return httpmock.NewStringResponse(404, "Unknown operation"), nil
		})
}

func mockExpoForRequestUploadUrlTest(channelName string) {
	httpmock.RegisterResponder("POST", "https://api.expo.dev/graphql",
		func(req *http.Request) (*http.Response, error) {
			isFetchSelfExpoUsername := req.Header.Get("operationName") == "FetchExpoUserAccountInformations"
			isFetchExpoChannelMapping := req.Header.Get("operationName") == "FetchExpoChannelMapping"
			isFetchBranches := req.Header.Get("operationName") == "FetchExpoBranches"
			isCreateBranch := req.Header.Get("operationName") == "CreateBranch"
			if isFetchBranches {
				return MockExpoBranchesResponse([]map[string]interface{}{
					{
						"id":   "branch-1-id",
						"name": "branch-1",
					},
					{
						"id":   "branch-2-id",
						"name": "branch-2",
					},
					{
						"id":   "do-not-use",
						"name": "DO_NOT_USE",
					},
				})
			}
			if isCreateBranch {
				return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"data": map[string]interface{}{
						"updateBranch": map[string]interface{}{
							"createUpdateBranchForApp": map[string]interface{}{
								"id":   "created-branch-id",
								"name": "created-branch",
							},
						},
					},
				})
			}
			if isFetchSelfExpoUsername {
				authHeader := req.Header.Get("Authorization")
				if authHeader != "" {
					if authHeader == "Bearer expo_test_token" || authHeader == "Bearer EXPO_ACCESS_TOKEN" {
						return MockExpoAccountResponse(map[string]interface{}{
							"id":       "123",
							"username": "test_username",
							"email":    "test@example.com",
						})
					}
					if authHeader == "Bearer expo_alternative_token" {
						return MockExpoAccountResponse(map[string]interface{}{
							"id":       "1234",
							"username": "test_alternative_username",
							"email":    "test_alternative@example.com",
						})
					}
					if authHeader != "Bearer expo_test_token" {
						return httpmock.NewStringResponse(http.StatusUnauthorized, `{"error": "Unauthorized"}`), nil
					}
				}
				expoSession := req.Header.Get("expo-session")
				if expoSession != "" {
					if expoSession == "expo_test_session" {
						return MockExpoAccountResponse(map[string]interface{}{
							"id":       "123",
							"username": "test_username",
							"email":    "text@example.com",
						})
					}
					return httpmock.NewStringResponse(http.StatusUnauthorized, `{"error": "Unauthorized"}`), nil
				}
				return MockExpoAccountResponse(map[string]interface{}{
					"id":       "123",
					"username": "test_username",
					"email":    "test@example.com",
				})
			}

			if isFetchExpoChannelMapping {
				return MockExpoChannelMapping(
					[]map[string]interface{}{
						{
							"id":   "branch-1-id",
							"name": "branch-1",
						},
						{
							"id":   "branch-2-id",
							"name": "branch-2",
						},
						{
							"id":   "do-not-use",
							"name": "DO_NOT_USE",
						},
					},
					map[string]interface{}{
						"id":   channelName + "-id",
						"name": channelName,
						"branchMapping": StringifyBranchMapping(map[string]interface{}{
							"version": 0,
							"data": []map[string]interface{}{
								{
									"branchId":           "do-not-use",
									"branchMappingLogic": "true",
								},
							},
						}),
					},
				)
			}

			return httpmock.NewStringResponse(404, "Unknown operation"), nil
		})
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
	os.Setenv("EXPO_APP_ID", "EXPO_APP_ID")
	os.Setenv("EXPO_ACCESS_TOKEN", "EXPO_ACCESS_TOKEN")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", "")
	os.Setenv("CLOUDFRONT_DOMAIN", "")
	os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "")
	os.Setenv("USE_DASHBOARD", "true")
	os.Setenv("ADMIN_PASSWORD", "admin")
}
