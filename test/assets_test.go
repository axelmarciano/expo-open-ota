package test

import (
	"bytes"
	"compress/gzip"
	"expo-open-ota/internal/assets"
	"expo-open-ota/internal/cdn"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/update"
	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestEmptyAssetNameForAssets(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	request := assets.AssetsRequest{
		Branch:         "branch-1",
		AssetName:      "",
		RuntimeVersion: "1",
		Platform:       "ios",
		RequestID:      "test",
	}
	projectRoot, _ := findProjectRoot()
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
			os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
			os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
			os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
			return assets.HandleAssetsWithURL(req, &cdn.CloudfrontCDN{})
		})
	})
}

func TestBadPlatformForAssets(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	request := assets.AssetsRequest{
		Branch:         "branch-1",
		AssetName:      "/assets/4f1cb2cac2370cd5050681232e8575a8",
		RuntimeVersion: "1",
		Platform:       "blackberry",
		RequestID:      "test",
	}
	projectRoot, _ := findProjectRoot()
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
			os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
			os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
			os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
			return assets.HandleAssetsWithURL(req, &cdn.CloudfrontCDN{})
		})
	})
}

func TestMissingRuntimeVersionForAssets(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	request := assets.AssetsRequest{
		Branch:         "branch-1",
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
	projectRoot, _ := findProjectRoot()
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testMissingRuntimeVersion(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testMissingRuntimeVersion(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
			os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
			os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
			return assets.HandleAssetsWithURL(req, &cdn.CloudfrontCDN{})
		})
	})
}

func TestEmptyUpdatesForAssets(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	request := assets.AssetsRequest{
		Branch:         "emptyruntime",
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
	projectRoot, _ := findProjectRoot()
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testEmptyUpdates(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testEmptyUpdates(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
			os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
			os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
			return assets.HandleAssetsWithURL(req, &cdn.CloudfrontCDN{})
		})
	})
}

func TestBadRuntimeVersion(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	request := assets.AssetsRequest{
		Branch:         "branch-1",
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
	projectRoot, _ := findProjectRoot()
	t.Run("Test HandleAssetsWithFile", func(t *testing.T) {
		testBadRuntimeVersion(t, assets.HandleAssetsWithFile)
	})
	t.Run("Test HandleAssetsWithURL", func(t *testing.T) {
		testBadRuntimeVersion(t, func(req assets.AssetsRequest) (assets.AssetsResponse, error) {
			os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
			os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
			os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
			return assets.HandleAssetsWithURL(req, &cdn.CloudfrontCDN{})
		})
	})
}

func TestToRetrieveBundleAsset(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	asset := assets.AssetsRequest{
		Branch:         "branch-1",
		AssetName:      "bundles/android-82adadb1fb6e489d04ad95fd79670deb.js",
		RuntimeVersion: "1",
		Platform:       "android",
		RequestID:      "test",
	}
	projectRoot, _ := findProjectRoot()
	os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
	os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
	os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
	response, err := assets.HandleAssetsWithFile(asset)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 200, response.StatusCode, "Expected status code 200")
	assert.Equal(t, "application/javascript", response.ContentType, "Expected content type 'application/javascript'")
	assert.Empty(t, response.URL, "Expected URL to be empty")
	responseWithUrl, err := assets.HandleAssetsWithURL(asset, &cdn.CloudfrontCDN{})
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 200, responseWithUrl.StatusCode, "Expected status code 200")
	assert.Empty(t, responseWithUrl.Body, "Expected empty body")
	parsedUrl, err := url.Parse(responseWithUrl.URL)
	require.NoError(t, err, "Error while parsing the URL")
	expectedBaseURL := "https://cdn.expoopenota.com/branch-1/1/1674170951/bundles/android-82adadb1fb6e489d04ad95fd79670deb.js"
	assert.Equal(t, expectedBaseURL, parsedUrl.Scheme+"://"+parsedUrl.Host+parsedUrl.Path, "URL should match the expected base URL")
	queryParams := parsedUrl.Query()
	assert.NotEmpty(t, queryParams.Get("Policy"), "Policy should not be empty")
	assert.NotEmpty(t, queryParams.Get("Signature"), "Signature should not be empty")
	assert.NotEmpty(t, queryParams.Get("Key-Pair-Id"), "Key-Pair-Id should not be empty")
}

func TestToRetrieveBundleAssetWithGzipCompression(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	projectRoot, _ := findProjectRoot()
	setupChannelMapping("staging", "branch-1")

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "bundles/android-82adadb1fb6e489d04ad95fd79670deb.js", "1", "android", "staging")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("expo-channel-name", "staging")

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

	expectedContent, err := os.Open(filepath.Join(projectRoot, "/test/test-updates/branch-1/1/1674170951/bundles/android-82adadb1fb6e489d04ad95fd79670deb.js"))
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
	teardown := setup(t)
	defer teardown()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	setupChannelMapping("staging", "branch-1")

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "bundles/android-82adadb1fb6e489d04ad95fd79670deb.js", "1", "android", "staging")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "br")
	r.Header.Set("expo-channel-name", "staging")

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

	expectedContentPath := filepath.Join(projectRoot, "/test/test-updates/branch-1/1/1674170951/bundles/android-82adadb1fb6e489d04ad95fd79670deb.js")
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
	teardown := setup(t)
	defer teardown()
	setupChannelMapping("staging", "branch-1")

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "assets/4f1cb2cac2370cd5050681232e8575a8", "1", "android", "staging")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("expo-channel-name", "staging")

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

func TestAutomaticUrlRedirectionIfCDNIsSet(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	projectRoot, _ := findProjectRoot()
	os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
	os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
	os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
	setupChannelMapping("staging", "branch-1")

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "1", "android", "staging")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("expo-channel-name", "staging")

	handlers.AssetsHandler(w, r)

	assert.Equal(t, 302, w.Code, "Expected status code 302")
}

func TestPreventCDNRedirectionHeader(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	projectRoot, _ := findProjectRoot()
	os.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", filepath.Join(projectRoot, "/test/keys/private-key-cloudfront-test.pem"))
	os.Setenv("CLOUDFRONT_DOMAIN", "https://cdn.expoopenota.com")
	os.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
	setupChannelMapping("staging", "branch-1")

	url, _ := update.BuildFinalManifestAssetUrlURL("http://localhost:3000", "bundles/ios-9d01842d6ee1224f7188971c5d397115.js", "1", "android", "staging")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", url, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("prevent-cdn-redirection", "true")
	r.Header.Set("expo-channel-name", "staging")

	handlers.AssetsHandler(w, r)

	assert.Equal(t, 200, w.Code, "Expected status code 200")
}
