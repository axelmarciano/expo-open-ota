package test

import (
	"encoding/json"
	"expo-open-ota/internal/auth"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/handlers"
	infrastructure "expo-open-ota/internal/router"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginDashboardNotEnabled(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("USE_DASHBOARD", "false")
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", nil)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusNotFound, respRec.Code)

}

func TestLoginInvalidPassword(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	formData := url.Values{}
	formData.Set("password", "wrongpassword")
	req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestShouldRejectLoginIfAdminPasswordNotSet(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("ADMIN_PASSWORD", "")
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	formData := url.Values{}
	formData.Set("password", "admin")
	req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestLoginValidPassword(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	formData := url.Values{}
	formData.Set("password", "admin")
	req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	// Retrieve token & refreshToken from response
	body := respRec.Body.String()

	var response auth.AuthResponse
	err := json.Unmarshal([]byte(body), &response)
	assert.Nil(t, err)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.RefreshToken)
}

func login() auth.AuthResponse {
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	formData := url.Values{}
	formData.Set("password", "admin")
	req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(respRec, req)
	body := respRec.Body.String()
	var response auth.AuthResponse
	_ = json.Unmarshal([]byte(body), &response)
	return response
}

func TestRefreshToken(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	formData := url.Values{}
	formData.Set("refreshToken", login().RefreshToken)
	req, _ := http.NewRequest("POST", "/auth/refreshToken", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	body := respRec.Body.String()
	var response auth.AuthResponse
	err := json.Unmarshal([]byte(body), &response)
	assert.Nil(t, err)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.RefreshToken)
}

func TestSettings(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/settings", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)

	assert.Equal(t, http.StatusOK, respRec.Code)

	projectRoot, err := os.Getwd()
	assert.Nil(t, err)

	responseBody := strings.TrimSpace(respRec.Body.String())

	responseBody = strings.ReplaceAll(responseBody, projectRoot+"/test-updates", "{PROJECT_ROOT}/test/test-updates")
	responseBody = strings.ReplaceAll(responseBody, projectRoot+"/keys/public-key-test.pem", "{PROJECT_ROOT}/test/keys/public-key-test.pem")
	responseBody = strings.ReplaceAll(responseBody, projectRoot+"/keys/private-key-test.pem", "{PROJECT_ROOT}/test/keys/private-key-test.pem")

	expectedSnapshot := `{"BASE_URL":"http://localhost:3000","EOAS_API_KEY":"***EO","CACHE_MODE":"","REDIS_HOST":"","REDIS_PORT":"","STORAGE_MODE":"local","S3_BUCKET_NAME":"","LOCAL_BUCKET_BASE_PATH":"{PROJECT_ROOT}/test/test-updates","KEYS_STORAGE_TYPE":"local","AWSSM_EXPO_PUBLIC_KEY_SECRET_ID":"","AWSSM_EXPO_PRIVATE_KEY_SECRET_ID":"","PUBLIC_EXPO_KEY_B64":"","PUBLIC_LOCAL_EXPO_KEY_PATH":"{PROJECT_ROOT}/test/keys/public-key-test.pem","PRIVATE_LOCAL_EXPO_KEY_PATH":"{PROJECT_ROOT}/test/keys/private-key-test.pem","AWS_REGION":"eu-west-3","AWS_BASE_ENDPOINT":"","AWS_ACCESS_KEY_ID":"","CLOUDFRONT_DOMAIN":"","CLOUDFRONT_KEY_PAIR_ID":"","CLOUDFRONT_PRIVATE_KEY_B64":"","AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID":"","PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH":"","PROMETHEUS_ENABLED":""}`

	assert.Equal(t, expectedSnapshot, responseBody)
}

func TestSettingsWithoutAuth(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/settings", nil)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestBranches(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	setupChannelMapping("staging", "branch-1")
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branches", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)

	var response []handlers.BranchMapping
	err := json.Unmarshal(respRec.Body.Bytes(), &response)
	assert.Nil(t, err)
	assert.Equal(t, `[{"branchName":"branch-1","releaseChannels":["staging"]},{"branchName":"branch-2","releaseChannels":[]},{"branchName":"branch-3","releaseChannels":[]},{"branchName":"branch-4","releaseChannels":[]}]`, strings.TrimSpace(respRec.Body.String()))
}

func TestBranchesWithoutAuth(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branches", nil)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestRuntimeVersionsWithoutAuth(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-1/runtimeVersions", nil)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestRuntimeVersions(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-1/runtimeVersions", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	var response []bucket.RuntimeVersionWithStats
	err := json.Unmarshal(respRec.Body.Bytes(), &response)
	assert.Nil(t, err)
	assert.Equal(t, "[{\"runtimeVersion\":\"1\",\"lastUpdatedAt\":\"1970-01-20T09:02:50Z\",\"createdAt\":\"1970-01-20T09:02:50Z\",\"numberOfUpdates\":1}]", strings.TrimSpace(respRec.Body.String()))
}

func TestUpdatesWithoutAuth(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-1/runtimeVersion/1/updates", nil)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusUnauthorized, respRec.Code)
}

func TestUpdatesRegularBranch1(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-1/runtimeVersion/1/updates", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	assert.Equal(t, "[{\"updateUUID\":\"04b793a0-b6ab-fd4f-308c-b91d812adec2\",\"updateId\":\"1674170951\",\"createdAt\":\"1970-01-20T09:02:50Z\",\"commitHash\":\"1674170951\",\"platform\":\"android\"}]", strings.TrimSpace(respRec.Body.String()))
}

func TestUpdatesMultiBranch2(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-2/runtimeVersion/1/updates", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	assert.Equal(t, "[{\"updateUUID\":\"68e096e2-a619-9d56-7f7c-89f97bc27312\",\"updateId\":\"1737455526\",\"createdAt\":\"1970-01-21T02:37:35Z\",\"commitHash\":\"\",\"platform\":\"ios\"},{\"updateUUID\":\"fdc14544-9e15-732f-cd9c-e3e26c55cbea\",\"updateId\":\"1674170951\",\"createdAt\":\"1970-01-20T09:02:50Z\",\"commitHash\":\"\",\"platform\":\"android\"},{\"updateUUID\":\"d100f19f-e0be-45c4-212a-27d1f067552b\",\"updateId\":\"1666629107\",\"createdAt\":\"1970-01-20T06:57:09Z\",\"commitHash\":\"1674170951\",\"platform\":\"android\"},{\"updateUUID\":\"Rollback to embedded\",\"updateId\":\"1666629141\",\"createdAt\":\"1970-01-20T06:57:09Z\",\"commitHash\":\"1674170951\",\"platform\":\"ios\"},{\"updateUUID\":\"Rollback to embedded\",\"updateId\":\"1666304169\",\"createdAt\":\"1970-01-20T06:51:44Z\",\"commitHash\":\"1674170951\",\"platform\":\"ios\"}]", strings.TrimSpace(respRec.Body.String()))
}

func TestUpdatesSomeNotValidBranch4(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/branch-4/runtimeVersion/1/updates", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, http.StatusOK, respRec.Code)
	assert.Equal(t, "[{\"updateUUID\":\"3f23a8c4-cd0e-a5a4-63f2-bb2841e95a01\",\"updateId\":\"1674170951\",\"createdAt\":\"1970-01-20T09:02:50Z\",\"commitHash\":\"1674170951\",\"platform\":\"android\"}]", strings.TrimSpace(respRec.Body.String()))
}
