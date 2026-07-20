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
	t.Setenv("BUCKET_KEY_PREFIX", "")
	t.Setenv("S3_KEY_PREFIX", "")
	t.Setenv("KEYS_STORAGE_TYPE", "")
	t.Setenv("CLOUDFRONT_DOMAIN", "")
	t.Setenv("CLOUDFRONT_KEY_PAIR_ID", "")
	t.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", "")
	t.Setenv("PRIVATE_CLOUDFRONT_KEY_B64", "")
	t.Setenv("AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID", "")
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

func TestGetCDNPrefersCloudfrontOverGeneric(t *testing2.T) {
	clearCDNEnv(t)
	t.Setenv("STORAGE_MODE", "s3")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("CDN_BASE_URL", "https://cdn.example.com")
	t.Setenv("CLOUDFRONT_DOMAIN", "https://cloudfront.example.com")
	t.Setenv("CLOUDFRONT_KEY_PAIR_ID", "test")
	t.Setenv("PRIVATE_CLOUDFRONT_KEY_PATH", "../../test/keys/private-key-cloudfront-test.pem")
	ResetCDNInstance()
	c := GetCDN()
	if _, ok := c.(*CloudfrontCDN); !ok {
		t.Fatalf("expected *CloudfrontCDN, got %T", c)
	}
}

func TestGenericCDNComputeRedirectionURL(t *testing2.T) {
	cases := []struct {
		name      string
		baseURL   string
		legacyURL string
		keyPrefix string
		expected  string
	}{
		{
			name:     "multi segment asset",
			baseURL:  "https://cdn.example.com",
			expected: "https://cdn.example.com/test-app-id/production/1/1674170951/bundles/android-abc.js",
		},
		{
			name:      "bucket key prefix included once",
			baseURL:   "https://cdn.example.com",
			keyPrefix: "prefix",
			expected:  "https://cdn.example.com/prefix/test-app-id/production/1/1674170951/bundles/android-abc.js",
		},
		{
			name:     "base url with path and trailing slash",
			baseURL:  "https://cdn.example.com/my-bucket/",
			expected: "https://cdn.example.com/my-bucket/test-app-id/production/1/1674170951/bundles/android-abc.js",
		},
		{
			name:      "legacy variable drives the url",
			legacyURL: "https://legacy.example.com",
			expected:  "https://legacy.example.com/test-app-id/production/1/1674170951/bundles/android-abc.js",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing2.T) {
			clearCDNEnv(t)
			t.Setenv("CDN_BASE_URL", tc.baseURL)
			t.Setenv("S3_CDN_PREFIX", tc.legacyURL)
			t.Setenv("BUCKET_KEY_PREFIX", tc.keyPrefix)
			c := &GenericCDN{}
			got, err := c.ComputeRedirectionURLForAsset("test-app-id", "production", "1", "1674170951", "bundles/android-abc.js")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
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
