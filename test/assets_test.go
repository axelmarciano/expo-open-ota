package test

import (
	"bytes"
	"compress/gzip"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/assets"
	"expo-open-ota/internal/modules/update"
	"github.com/andybalholm/brotli"
	"github.com/aws/aws-lambda-go/events"
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

	req := assets.AssetsRequest{
		Environment:    "badenv",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "1",
		Platform:       "ios",
		RequestID:      "test",
	}

	testInvalidEnvironment := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		res, err := handlerFunc(req)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 400, res.StatusCode, "Expected status code 400 for an invalid environment")
		assert.Equal(t, "Invalid environment", string(res.Body), "Expected 'Invalid environment' message")
	}

	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testInvalidEnvironment(t, assets.HandleAssetsWithFile)
	})

	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testInvalidEnvironment(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	request := assets.AssetsRequest{
		Environment:    "staging",
		AssetName:      "",
		RuntimeVersion: "1",
		Platform:       "ios",
		RequestID:      "test",
	}
	testEmptyAssetName := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		response, err := handlerFunc(request)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 400, response.StatusCode, "Expected status code 400 for an empty asset name")
		assert.Equal(t, "No asset name provided", string(response.Body), "Expected 'No asset name provided' message")
	}
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testEmptyAssetName(t, assets.HandleAssetsWithFile)
	})

	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testEmptyAssetName(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	request := assets.AssetsRequest{
		Environment:    "staging",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "1",
		Platform:       "blackberry",
		RequestID:      "test",
	}
	testInvalidPlatform := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		response, err := handlerFunc(request)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 400, response.StatusCode, "Expected status code 400 for an invalid platform")
		assert.Equal(t, "Invalid platform", string(response.Body), "Expected 'Invalid platform' message")
	}
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testInvalidPlatform(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testInvalidPlatform(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	request := assets.AssetsRequest{
		Environment:    "staging",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "",
		Platform:       "ios",
		RequestID:      "test",
	}
	testMissingRuntimeVersion := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		response, err := handlerFunc(request)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 400, response.StatusCode, "Expected status code 400 for a missing runtime version")
		assert.Equal(t, "No runtime version provided", string(response.Body), "Expected 'No runtime version provided' message")
	}
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testMissingRuntimeVersion(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testMissingRuntimeVersion(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	request := assets.AssetsRequest{
		Environment:    "emptyruntime",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "1",
		Platform:       "ios",
		RequestID:      "test",
	}
	testEmptyUpdates := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		response, err := handlerFunc(request)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 404, response.StatusCode, "Expected status code 404 for an empty update")
		assert.Equal(t, "No update found", string(response.Body), "Expected 'No update found' message")
	}
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testEmptyUpdates(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testEmptyUpdates(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	request := assets.AssetsRequest{
		Environment:    "staging",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "never",
		Platform:       "ios",
		RequestID:      "test",
	}
	testBadRuntimeVersion := func(t *testing.T, handlerFunc func(assets.AssetsRequest) (assets.AssetsResponse, error)) {
		response, err := handlerFunc(request)
		assert.Nil(t, err, "Expected no error")
		assert.Equal(t, 404, response.StatusCode, "Expected status code 404 for a bad runtime version")
		assert.Equal(t, "No update found", string(response.Body), "Expected 'No update found' message")
	}
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testBadRuntimeVersion(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testBadRuntimeVersion(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			return assets.HandleAssetsWithURL(req, "https://cdn.expoopenota.com")
		})
	})
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
	asset := assets.AssetsRequest{
		Environment:    "staging",
		AssetName:      "bundles/ios-9d01842d6ee1224f7188971c5d397115.js",
		RuntimeVersion: "1",
		Platform:       "ios",
		RequestID:      "test",
	}
	response, err := assets.HandleAssetsWithFile(asset)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 200, response.StatusCode, "Expected status code 200")
	assert.Equal(t, "application/javascript", response.ContentType, "Expected content type 'application/javascript'")
	assert.Empty(t, response.URL, "Expected URL to be empty")
	responseWithUrl, err := assets.HandleAssetsWithURL(asset, "https://cdn.expoopenota.com")
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 200, responseWithUrl.StatusCode, "Expected status code 200")
	assert.Equal(t, "application/javascript", responseWithUrl.ContentType, "Expected content type 'application/javascript'")
	assert.Empty(t, responseWithUrl.Body, "Expected empty body")
	assert.Equal(t, responseWithUrl.URL, "https://cdn.expoopenota.com/staging/bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "Expected URL to be 'https://cdn.expoopenota.com'")
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

func TestToRetrievePNGFromLambda(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "emptyruntime,staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "/test/test-updates"))
	os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Accept-Encoding": "gzip",
		},
		PathParameters: map[string]string{
			"environment": "staging",
		},
		QueryStringParameters: map[string]string{
			"asset":          "assets/4f1cb2cac2370cd5050681232e8575a8",
			"runtimeVersion": "1",
			"platform":       "ios",
		},
	}
	res, err := handlers.LambdaAssetsHandler(request)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 302, res.StatusCode, "Expected status code 200")
	assert.Equal(t, "https://cdn.expoopenota.com/staging/assets/4f1cb2cac2370cd5050681232e8575a8", res.Headers["Location"], "Expected location to be 'https://cdn.expoopenota.com'")
	assert.Equal(t, "image/png", res.Headers["Content-Type"], "Expected 'image/png' content type")
	assert.Equal(t, "1", res.Headers["expo-protocol-version"], "Expected protocol version 1")
	assert.Equal(t, "0", res.Headers["expo-sfv-version"], "Expected SFV version 0")
}
