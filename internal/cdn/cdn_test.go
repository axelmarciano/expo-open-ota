package cdn

import (
	testing2 "testing"
)

func clearCDNEnv(t *testing2.T) {
	t.Setenv("STORAGE_MODE", "")
	t.Setenv("S3_BUCKET_NAME", "")
	t.Setenv("GCS_BUCKET_NAME", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS_B64", "")
	t.Setenv("CDN_BASE_URL", "")
	t.Setenv("S3_CDN_PREFIX", "")
}

func TestGetCDNReturnsGCSDirectWhenGCSConfigured(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("STORAGE_MODE", "gcs")
	t.Setenv("GCS_BUCKET_NAME", "test-bucket")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS_B64", "e3ZhbHVlOiAxfQ==")
	ResetCDNInstance()
	c := GetCDN()
	if c == nil {
		t.Fatalf("expected CDN instance, got nil")
	}
	if _, ok := c.(*GCSDirectCDN); !ok {
		t.Fatalf("expected *GCSDirectCDN, got %T", c)
	}
}

func TestGetCDNReturnsGenericWithLegacyS3CDNPrefix(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("STORAGE_MODE", "s3")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("S3_CDN_PREFIX", "https://cdn.example.com")
	ResetCDNInstance()
	c := GetCDN()
	if c == nil {
		t.Fatalf("expected GenericCDN instance, got nil")
	}
	if _, ok := c.(*GenericCDN); !ok {
		t.Fatalf("expected *GenericCDN, got %T", c)
	}
}

func TestGetCDNReturnsGenericWithCDNBaseURLOnGCS(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("STORAGE_MODE", "gcs")
	t.Setenv("GCS_BUCKET_NAME", "test-bucket")
	// gcs-direct is also available here; the explicit base URL must win.
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS_B64", "e3ZhbHVlOiAxfQ==")
	t.Setenv("CDN_BASE_URL", "https://cdn.example.com")
	ResetCDNInstance()
	c := GetCDN()
	if c == nil {
		t.Fatalf("expected CDN instance, got nil")
	}
	if _, ok := c.(*GenericCDN); !ok {
		t.Fatalf("expected *GenericCDN, got %T", c)
	}
}

func TestGenericCDNUnavailableWithLocalStorage(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("STORAGE_MODE", "local")
	t.Setenv("CDN_BASE_URL", "https://cdn.example.com")
	ResetCDNInstance()
	if c := GetCDN(); c != nil {
		t.Fatalf("expected no CDN with local storage, got %T", c)
	}
}

func TestResolveCDNBaseURLPrefersNewVariable(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("CDN_BASE_URL", "https://new.example.com")
	t.Setenv("S3_CDN_PREFIX", "https://legacy.example.com")
	if got := ResolveCDNBaseURL(); got != "https://new.example.com" {
		t.Fatalf("expected CDN_BASE_URL to win, got %q", got)
	}
	t.Setenv("CDN_BASE_URL", "")
	if got := ResolveCDNBaseURL(); got != "https://legacy.example.com" {
		t.Fatalf("expected S3_CDN_PREFIX fallback, got %q", got)
	}
}
