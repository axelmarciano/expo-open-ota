package config

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotValidStorage(t *testing.T) {
	isValid := validateStorageMode("bag")
	assert.False(t, isValid)
}

func TestValidLocalStorage(t *testing.T) {
	isValid := validateStorageMode("local")
	assert.True(t, isValid)
}

func TestNotValidEmptyBaseUrl(t *testing.T) {
	isValid := validateBaseUrl("")
	assert.False(t, isValid)
}

func TestNotValidBaseUrl(t *testing.T) {
	isValid := validateBaseUrl("test.com")
	assert.False(t, isValid)
}

func TestMissingBucketParamsForS3(t *testing.T) {
	os.Setenv("S3_BUCKET_NAME", "")
	bucketParams := validateBucketParams("s3")
	assert.False(t, bucketParams)
}

func TestMissingBucketParamsForLocal(t *testing.T) {
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	bucketParams := validateBucketParams("local")
	assert.True(t, bucketParams)
}

func TestValidBaseUrl(t *testing.T) {
	isValid := validateBaseUrl("http://test.com")
	assert.True(t, isValid)
}

func TestNotValidConfigStorage(t *testing.T) {
	os.Setenv("STORAGE_MODE", "bag")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
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

func TestValidConfig(t *testing.T) {
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	LoadConfig()
}

func TestFallbackDefaultEnv(t *testing.T) {
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	LoadConfig()
	localBucketBasePath := GetEnv("LOCAL_BUCKET_BASE_PATH")
	assert.Equal(t, DefaultEnvValues["LOCAL_BUCKET_BASE_PATH"], localBucketBasePath)
}

func TestNotSetEnv(t *testing.T) {
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "")
	LoadConfig()
	assert.Empty(t, GetEnv("NOT_FOUND"))
}

func TestAwsBaseEndpointSet(t *testing.T) {
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	expectedEndpoint := "https://test-account.r2.cloudflarestorage.com"
	os.Setenv("AWS_BASE_ENDPOINT", expectedEndpoint)
	LoadConfig()
	actualEndpoint := GetEnv("AWS_BASE_ENDPOINT")
	assert.Equal(t, expectedEndpoint, actualEndpoint)
}

func TestAwsBaseEndpointNotSet(t *testing.T) {
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("BASE_URL", "http://test.com")
	os.Setenv("EOAS_API_KEY", "test")
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "./updates")
	os.Unsetenv("AWS_BASE_ENDPOINT")
	LoadConfig()
	endpoint := GetEnv("AWS_BASE_ENDPOINT")
	assert.Equal(t, DefaultEnvValues["AWS_BASE_ENDPOINT"], endpoint)
	assert.Empty(t, endpoint)
}

func TestTestMode(t *testing.T) {
	testMode := IsTestMode()
	assert.True(t, testMode)
}
