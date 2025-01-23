package test

import (
	"encoding/json"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/types"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNotValidEnvironment(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	q := "http://localhost:3000/manifest/bad_env"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "bad_env",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400 for an invalid environment")
	assert.Equal(t, "Invalid environment\n", w.Body.String(), "Expected 'Invalid environment' message")
}

func TestNotValidProtocolVersion(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "invalid")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400 for an invalid protocole version")
	assert.Equal(t, "Invalid protocol version\n", w.Body.String(), "Expected 'Invalid protocol version' message")
}

func TestNotValidPlatform(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "bad-platform")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400 for an invalid platform")
	assert.Equal(t, "Invalid platform\n", w.Body.String(), "Expected 'IInvalid platform' message")
}

func TestNotValidRuntimeVersion(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400 when runtime version is not provided")
	assert.Equal(t, "No runtime version provided\n", w.Body.String(), "Expected 'No runtime version provided' message")
}

func TestNotValidCertificates(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/not.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/exists.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)

	assert.Equal(t, 500, w.Code, "Expected status code 200 when manifest is retrieved")
	assert.Equal(t, "Error signing content\n", w.Body.String(), "Expected 'Error signing content' message")
}

func TestNoUpdates(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "nop")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 404, w.Code, "Expected status code 404")
	assert.Equal(t, "No update found\n", w.Body.String(), "Expected 'No updates found' message")
}

func TestValidRequestForStagingManifest(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	q := "http://localhost:3000/manifest/staging"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", q, nil)
	r.Header.Add("expo-platform", "ios")
	r.Header.Add("expo-runtime-version", "1")
	r.Header.Add("expo-protocol-version", "1")
	r.Header.Add("expo-expect-signature", "true")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.ManifestHandler(w, r)
	assert.Equal(t, 200, w.Code, "Expected status code 200 when manifest is retrieved")
	parts, err := ParseMultipartMixedResponse(w.Header().Get("Content-Type"), w.Body.Bytes())
	if err != nil {
		t.Errorf("Error parsing response: %v", err)
	}
	assert.Equal(t, 1, len(parts), "Expected 1 parts in the response")

	manifestPart := parts[0]

	assert.Equal(t, IsMultipartPartWithName(manifestPart, "manifest"), true, "Expected a part with name 'manifest'")
	body := manifestPart.Body

	var updateManifest types.UpdateManifest
	err = json.Unmarshal([]byte(body), &updateManifest)
	if err != nil {
		t.Errorf("Error parsing json body: %v", err)
	}
	assert.Equal(t, updateManifest.CreatedAt, "2025-01-21T00:07:00.912Z", "Expected a specific created at date")
	assert.Equal(t, updateManifest.RunTimeVersion, "1", "Expected a specific runtime version")
	assert.Equal(t, updateManifest.Metadata, json.RawMessage("{}"), "Expected empty metadata")
}
