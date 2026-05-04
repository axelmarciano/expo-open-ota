package cdn

import (
	"os"
	testing2 "testing"
)

func TestGetCDNReturnsGCSDirectWhenGCSConfigured(t *testing2.T) {
	os.Setenv("STORAGE_MODE", "gcs")
	os.Setenv("GCS_BUCKET_NAME", "test-bucket")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_B64", "e3ZhbHVlOiAxfQ==")
	ResetCDNInstance()
	c := GetCDN()
	if c == nil {
		t.Fatalf("expected CDN instance, got nil")
	}
	if _, ok := c.(*GCSDirectCDN); !ok {
		t.Fatalf("expected *GCSDirectCDN, got %T", c)
	}
}

func TestGetCDNReturnsGenericWhenGenericConfigured(t *testing2.T) {
	os.Setenv("STORAGE_MODE", "s3")
	os.Setenv("S3_BUCKET_NAME", "test-bucket")
	os.Setenv("S3_CDN_PREFIX", "https://cdn.example.com")
	ResetCDNInstance()
	c := GetCDN()
	if c == nil {
		t.Fatalf("expected GenericCDN instance, got nil")
	}
	if _, ok := c.(*GenericCDN); !ok {
		t.Fatalf("expected *GenericCDN, got %T", c)
	}
}
