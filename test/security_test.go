package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"expo-open-ota/internal/handlers"
	infrastructure "expo-open-ota/internal/router"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestManifestRejectsPathTraversalInChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/manifest"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-channel-name", "../etc/passwd")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestManifestRejectsSlashInChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/manifest"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-channel-name", "my/channel")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestManifestRejectsEmptyChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/manifest"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-channel-name", "")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "cannot be empty")
}

func TestManifestRejectsPathTraversalInRuntimeVersion(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	setupChannelMapping("staging", "branch-1")

	q := "http://localhost:3000/manifest"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "../../etc/passwd")
	r.Header.Add("expo-channel-name", "staging")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestManifestAcceptsValidSpecialCharsInRuntimeVersion(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	setupChannelMapping("staging", "branch-1")

	q := "http://localhost:3000/manifest"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "v2.1.2+build123")
	r.Header.Add("expo-channel-name", "staging")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 200, w.Code, "v2.1.2+build123 should be accepted as a valid runtime version")
}

func TestUploadRejectsPathTraversalInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/..%2F..%2Fetc?runtimeVersion=1&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "../..%2Fetc"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestUploadRejectsSlashInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/my/branch?runtimeVersion=1&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "my/branch"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestUploadRejectsPathTraversalInRuntimeVersion(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/main?runtimeVersion=../../../etc&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "main"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestUploadRejectsNullByteInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/branch%00evil?runtimeVersion=1&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "branch\x00evil"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestRollbackRejectsPathTraversalInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/rollback/..%2F..%2Fetc?platform=ios&runtimeVersion=1&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "../..%2Fetc"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RollbackHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestRepublishRejectsPathTraversalInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/republish/..%2F..%2Fetc?platform=ios&runtimeVersion=1&commitHash=abc&updateId=123"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "../..%2Fetc"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RepublishHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestDashboardRuntimeVersionsRejectsInvalidBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/.hidden/runtimeVersions", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "contains invalid characters")
}

func TestDashboardUpdatesRejectsInvalidBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/.hidden/runtimeVersion/1/updates", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "contains invalid characters")
}

func TestDashboardPathTraversalBlockedByRouter(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/..%2F..%2Fetc/runtimeVersions", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.NotEqual(t, 200, respRec.Code, "Path traversal should be blocked by router or handler")
}

func TestDashboardAcceptsValidBranchWithSpecialChars(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/staging%2B2/runtimeVersions", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.NotContains(t, respRec.Body.String(), "contains invalid characters", "staging+2 should pass validation")
}

func TestChannelMappingRejectsInvalidBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "production"})
	req, _ := http.NewRequest("POST", "/api/branch/.hidden/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "contains invalid characters")
}

func TestChannelMappingRejectsPathTraversalInChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "../../../etc/passwd"})
	req, _ := http.NewRequest("POST", "/api/branch/main/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "contains invalid characters")
}

func TestChannelMappingRejectsSlashInChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "my/channel"})
	req, _ := http.NewRequest("POST", "/api/branch/main/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "contains invalid characters")
}

func TestChannelMappingRejectsEmptyChannel(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": ""})
	req, _ := http.NewRequest("POST", "/api/branch/main/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 400, respRec.Code)
	assert.Contains(t, respRec.Body.String(), "cannot be empty")
}

func TestChannelMappingSuccess(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "production"})
	req, _ := http.NewRequest("POST", "/api/branch/branch-1/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 200, respRec.Code)
}

func TestChannelMappingRequiresAuth(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "production"})
	req, _ := http.NewRequest("POST", "/api/branch/branch-1/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 401, respRec.Code)
}

func TestChannelMappingEndToEnd(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"releaseChannel": "e2e-channel"})
	req, _ := http.NewRequest("POST", "/api/branch/branch-1/updateChannelBranchMapping", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+login().Token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(respRec, req)
	assert.Equal(t, 200, respRec.Code)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://localhost:3000/manifest", nil)
	r.Header.Add("expo-platform", "android")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r.Header.Add("expo-channel-name", "e2e-channel")
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 200, w.Code, "Manifest should work with newly created channel mapping")
}

func TestURLEncodedBranchInUpload(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/staging%2B2?runtimeVersion=1&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "staging%2B2"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 200, w.Code, "staging+2 (URL-encoded as staging%%2B2) should be accepted")
}

func TestURLEncodedBranchInDashboard(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	router := infrastructure.NewRouter()
	respRec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branch/staging%2B2/runtimeVersions", nil)
	req.Header.Set("Authorization", "Bearer "+login().Token)
	router.ServeHTTP(respRec, req)
	assert.NotContains(t, respRec.Body.String(), "contains invalid characters", "staging+2 (URL-encoded) should pass validation")
}

func TestMarkUploadedRejectsPathTraversalInBranch(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	q := "http://localhost:3000/markUpdateAsUploaded/..%2Fetc?platform=ios&runtimeVersion=1&updateId=123"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "../etc"})
	r.Header.Set("Authorization", "Bearer EOAS_API_KEY")
	handlers.MarkUpdateAsUploadedHandler(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "contains invalid characters")
}

func TestUploadRejectsWhenEOASAPIKeyUnset(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	os.Unsetenv("EOAS_API_KEY")

	body, _ := json.Marshal(handlers.FileNamesRequest{FileNames: []string{"test.js"}})
	q := "http://localhost:3000/requestUploadUrl/main?runtimeVersion=1&platform=ios&commitHash=abc"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, io.NopCloser(bytes.NewReader(body)))
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "main"})
	r.Header.Set("Authorization", "Bearer some-token")
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, 401, w.Code, "Should reject when EOAS_API_KEY is not set")
}

func TestMarkUploadedRejectsWhenEOASAPIKeyUnset(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	os.Unsetenv("EOAS_API_KEY")

	q := "http://localhost:3000/markUpdateAsUploaded/main?platform=ios&runtimeVersion=1&updateId=123"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{"BRANCH": "main"})
	r.Header.Set("Authorization", "Bearer some-token")
	handlers.MarkUpdateAsUploadedHandler(w, r)
	assert.Equal(t, 401, w.Code, "Should reject when EOAS_API_KEY is not set")
}
