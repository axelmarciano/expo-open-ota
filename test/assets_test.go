package test

import (
	"bytes"
	"compress/gzip"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/update"
	"github.com/andybalholm/brotli"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io"
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

func TestToRetrieveBundleAsset(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})
	handlers.AssetsHandler(w, r)
	assert.Equal(t, 200, w.Code, "Expected status code 200")
	assert.Equal(t, "application/javascript", w.Header().Get("Content-Type"), "Expected 'application/javascript' content type")
}

func TestToRetrieveBundleAssetWithGzipCompression(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})

	handlers.AssetsHandler(w, r)

	assert.Equal(t, 200, w.Code, "Expected status code 200")

	assert.Equal(t, "application/javascript", w.Header().Get("Content-Type"), "Expected 'application/javascript' content type")

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "Expected 'gzip' content encoding")

	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	decompressedBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}

	expectedContent, err := os.Open(filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951/bundles/ios-9d01842d6ee1224f7188971c5d397115.js"))
	if err != nil {
		t.Fatalf("Failed to open expected content: %v", err)
	}
	expectedContentBytes, err := io.ReadAll(expectedContent)
	if err != nil {
		t.Fatalf("Failed to read expected content: %v", err)
	}
	assert.Equal(t, string(expectedContentBytes), string(decompressedBody), "Expected content does not match decompressed content")
}

func TestToRetrieveBundleAssetWithBrotliCompression(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "br")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})

	handlers.AssetsHandler(w, r)

	assert.Equal(t, 200, w.Code, "Expected status code 200")

	assert.Equal(t, "application/javascript", w.Header().Get("Content-Type"), "Expected 'application/javascript' content type")

	assert.Equal(t, "br", w.Header().Get("Content-Encoding"), "Expected 'br' content encoding")

	decompressedBody := new(bytes.Buffer)
	brReader := brotli.NewReader(w.Body)
	_, err = io.Copy(decompressedBody, brReader)
	if err != nil {
		t.Fatalf("Failed to decompress Brotli content: %v", err)
	}

	expectedContentPath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951/bundles/ios-9d01842d6ee1224f7188971c5d397115.js")
	expectedContent, err := os.Open(expectedContentPath)
	if err != nil {
		t.Fatalf("Failed to open expected content: %v", err)
	}
	defer expectedContent.Close()

	expectedContentBytes, err := io.ReadAll(expectedContent)
	if err != nil {
		t.Fatalf("Failed to read expected content: %v", err)
	}

	assert.Equal(t, string(expectedContentBytes), decompressedBody.String(), "Expected content does not match decompressed content")
}

func TestToRetrievePNGAssetWithGzipCompression(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "staging", "assets/4f1cb2cac2370cd5050681232e8575a8", "1", "ios")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "staging",
	})

	handlers.AssetsHandler(w, r)

	assert.Equal(t, 200, w.Code, "Expected status code 200")

	assert.Equal(t, "image/png", w.Header().Get("Content-Type"), "Expected 'application/javascript' content type")

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "Expected 'gzip' content encoding")

	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

}
