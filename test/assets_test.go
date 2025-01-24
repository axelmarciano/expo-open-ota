package test

import (
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/update"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNotValidEnvironmentForAssets(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "badenv", "/assets/4f1cb2cac2370cd5050681232e8575a8", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "bad_env",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400 for an invalid environment")
	assert.Equal(t, "Invalid environment\n", w.Body.String(), "Expected 'Invalid environment' message")
}

func TestEmptyAssetNameForAssets(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400")
	assert.Equal(t, "No asset name provided\n", w.Body.String(), "Expected 'No asset name provided' message")
}

func TestBadPlatformForAssets(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "/assets/4f1cb2cac2370cd5050681232e8575a8", "1", "blackberry")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400")
	assert.Equal(t, "Invalid platform\n", w.Body.String(), "Expected 'Invalid platform' message")
}

func TestMissingRuntimeVersionForAssets(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "/assets/4f1cb2cac2370cd5050681232e8575a8", "", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 400, w.Code, "Expected status code 400")
	assert.Equal(t, "No runtime version provided\n", w.Body.String(), "Expected 'No runtime version provided' message")
}

func TestEmptyUpdatesForAssets(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "emptyruntime", "/assets/4f1cb2cac2370cd5050681232e8575a8", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "emptyruntime",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 404, w.Code, "Expected status code 404")
	assert.Equal(t, "No update found\n", w.Body.String(), "Expected 'No update found' message")
}

func TestBadRuntimeVersion(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "/assets/4f1cb2cac2370cd5050681232e8575a8", "never", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 404, w.Code, "Expected status code 404")
	assert.Equal(t, "No update found\n", w.Body.String(), "Expected 'No update found' message")
}
