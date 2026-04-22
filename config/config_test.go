package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	testing2 "testing"
)

func setup(t *testing2.T) func() {
	return func() {
	}
}
func TestNotValidStorage(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	isValid := validateStorageMode("bag")
	assert.False(t, isValid)
}

func TestValidLocalStorage(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	isValid := validateStorageMode("local")
	assert.True(t, isValid)
}

func TestNotValidEmptyBaseUrl(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	isValid := validateBaseUrl("")
	assert.False(t, isValid)
}

func TestNotValidBaseUrl(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	isValid := validateBaseUrl("test.com")
	assert.False(t, isValid)
}

func TestMissingBucketParamsForS3(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("S3_BUCKET_NAME", "")
	bucketParams := validateBucketParams("s3")
	assert.False(t, bucketParams)
}

func TestMissingBucketParamsForLocal(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	bucketParams := validateBucketParams("local")
	// Should be set as ./updates by default config values
	assert.True(t, bucketParams)
}

func TestValidBaseUrl(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	isValid := validateBaseUrl("http://test.com")
	assert.True(t, isValid)
}

func TestNotValidConfigStorage(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "bag")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		LoadConfig()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNotValidConfig")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()

	assert.Error(t, err)
	exitError, ok := err.(*exec.ExitError)
	assert.True(t, ok)
	assert.Equal(t, 1, exitError.ExitCode())
}

func TestValidConfig(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	LoadConfig()
}

func TestFallbackDefaultEnv(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	LoadConfig()
	localBucketBasePath := GetEnv("LOCAL_BUCKET_BASE_PATH")
	assert.Equal(t, DefaultEnvValues["LOCAL_BUCKET_BASE_PATH"], localBucketBasePath)
}

func TestNotSetEnv(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	LoadConfig()
	assert.Empty(t, GetEnv("NOT_FOUND"))
}


func TestAwsBaseEndpointSet(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	
	expectedEndpoint := "https://test-account.r2.cloudflarestorage.com"
	os.Setenv("AWS_BASE_ENDPOINT", expectedEndpoint)
	LoadConfig()
	actualEndpoint := GetEnv("AWS_BASE_ENDPOINT")
	assert.Equal(t, expectedEndpoint, actualEndpoint)
}

func TestAwsBaseEndpointNotSet(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EXPO_APPS_JSON", `[{"id":"test","accessToken":"test","keys":{"mode":"environment","publicB64":"dGVzdA==","privateB64":"dGVzdA=="}}]`)
	ResetAppsForTest()
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	os.Unsetenv("AWS_BASE_ENDPOINT")
	LoadConfig()
	endpoint := GetEnv("AWS_BASE_ENDPOINT")
	assert.Equal(t, DefaultEnvValues["AWS_BASE_ENDPOINT"], endpoint)
	assert.Empty(t, endpoint)
}

func TestTestMode(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	testMode := IsTestMode()
	assert.True(t, testMode)
}

