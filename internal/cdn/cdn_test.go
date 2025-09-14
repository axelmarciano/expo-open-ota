package cdn

import (
    "os"
    testing2 "testing"
)

func TestGetCDNReturnsGCSDirectWhenGCSConfigured(t *testing2.T) {
    os.Setenv("STORAGE_MODE", "gcs")
    os.Setenv("GCS_BUCKET_NAME", "test-bucket")
    os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/fake.json")
    ResetCDNInstance()
    c := GetCDN()
    if c == nil {
        t.Fatalf("expected CDN instance, got nil")
    }
    if _, ok := c.(*GCSDirectCDN); !ok {
        t.Fatalf("expected *GCSDirectCDN, got %T", c)
    }
}
