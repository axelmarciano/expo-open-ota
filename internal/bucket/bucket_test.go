package bucket

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	testing2 "testing"
)

func setup(t *testing2.T) func() {
	return func() {
		ResetBucketInstance()
	}
}

func TestResolveLocalBucketType(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	bucketType := ResolveBucketType()
	assert.Equal(t, LocalBucketType, bucketType)
}



func TestResolveGCSBucketType(t *testing2.T) {
    teardown := setup(t)
    defer teardown()
    os.Setenv("STORAGE_MODE", "gcs")
    bucketType := ResolveBucketType()
    assert.Equal(t, GCSBucketType, bucketType)
}
func TestResolveS3BucketType(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "s3")
	bucketType := ResolveBucketType()
	assert.Equal(t, S3BucketType, bucketType)
}

func TestConvertReadCloserToBytes(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	rc := io.NopCloser(bytes.NewReader([]byte("test")))
	bytes, err := ConvertReadCloserToBytes(rc)
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), bytes)
}

func TestErrorOnConvertReadCloserToBytes(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	errorReader := &ErrorReadCloser{
		ReadErr:  fmt.Errorf("simulated read error"),
		CloseErr: nil,
	}

	_, err := ConvertReadCloserToBytes(errorReader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error copying file to buffer")
	assert.Contains(t, err.Error(), "simulated read error")
}

type ErrorReadCloser struct {
	ReadErr  error
	CloseErr error
}

func (e *ErrorReadCloser) Read(p []byte) (int, error) {
	return 0, e.ReadErr
}

func (e *ErrorReadCloser) Close() error {
	return e.CloseErr
}

func TestGetS3Bucket(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "s3")
	os.Setenv("S3_BUCKET_NAME", "test")
	bucket := GetBucket()
	assert.IsType(t, &S3Bucket{}, bucket)
}

func TestPrefixedKeyWithPrefix(t *testing2.T) {
	b := &S3Bucket{KeyPrefix: "myapp/"}
	assert.Equal(t, "myapp/branch/main", b.prefixedKey("branch/main"))
}

func TestPrefixedKeyWithoutPrefix(t *testing2.T) {
	b := &S3Bucket{KeyPrefix: ""}
	assert.Equal(t, "branch/main", b.prefixedKey("branch/main"))
}

func TestGetS3BucketWithKeyPrefix(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "s3")
	os.Setenv("S3_BUCKET_NAME", "test")
	os.Setenv("S3_KEY_PREFIX", "myapp")
	defer os.Unsetenv("S3_KEY_PREFIX")
	bucket := GetBucket()
	s3b, ok := bucket.(*S3Bucket)
	assert.True(t, ok)
	assert.Equal(t, "myapp/", s3b.KeyPrefix)
}

func TestGetS3BucketKeyPrefixAlreadyHasSlash(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "s3")
	os.Setenv("S3_BUCKET_NAME", "test")
	os.Setenv("S3_KEY_PREFIX", "myapp/")
	defer os.Unsetenv("S3_KEY_PREFIX")
	bucket := GetBucket()
	s3b, ok := bucket.(*S3Bucket)
	assert.True(t, ok)
	assert.Equal(t, "myapp/", s3b.KeyPrefix)
}

func TestGetS3BucketWithoutKeyPrefix(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "s3")
	os.Setenv("S3_BUCKET_NAME", "test")
	os.Unsetenv("S3_KEY_PREFIX")
	bucket := GetBucket()
	s3b, ok := bucket.(*S3Bucket)
	assert.True(t, ok)
	assert.Equal(t, "", s3b.KeyPrefix)
}

func TestGetLocalBucket(t *testing2.T) {
	teardown := setup(t)
	defer teardown()
	os.Setenv("STORAGE_MODE", "local")
	os.Setenv("LOCAL_BUCKET_BASE_PATH", "test")
	bucket := GetBucket()
	assert.IsType(t, &LocalBucket{}, bucket)
}


func TestGetGCSBucket(t *testing2.T) {
    teardown := setup(t)
    defer teardown()
    os.Setenv("STORAGE_MODE", "gcs")
    os.Setenv("GCS_BUCKET_NAME", "test-bucket")
    bucket := GetBucket()
    assert.IsType(t, &GCSBucket{}, bucket)
}
