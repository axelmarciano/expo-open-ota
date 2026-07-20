package azure

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

func sasProtocol(t *testing.T, signedURL string) string {
	t.Helper()
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return parsed.Query().Get("spr")
}

// Azurite well-known development credentials: signing is pure HMAC, so SAS
// generation is fully testable offline.
const (
	azuriteAccountName = "devstoreaccount1"
	azuriteAccountKey  = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

func setAzuriteEnv(t *testing.T) {
	t.Setenv("AZURE_STORAGE_ACCOUNT_NAME", azuriteAccountName)
	t.Setenv("AZURE_STORAGE_ACCOUNT_KEY", azuriteAccountKey)
	t.Setenv("AZURE_BLOB_ENDPOINT", "http://127.0.0.1:10000/devstoreaccount1")
}

func TestSignBlobSASUploadURLShape(t *testing.T) {
	setAzuriteEnv(t)
	signedURL, err := SignBlobSAS("updates", "app/branch/1/1674170951/bundles/android.js", sas.BlobPermissions{Create: true, Write: true}, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPrefix := "http://127.0.0.1:10000/devstoreaccount1/updates/app/branch/1/1674170951/bundles/android.js?"
	if !strings.HasPrefix(signedURL, wantPrefix) {
		t.Fatalf("expected URL to start with %q, got %q", wantPrefix, signedURL)
	}
	for _, param := range []string{"sig=", "se=", "st=", "sp=cw"} {
		if !strings.Contains(signedURL, param) {
			t.Fatalf("expected URL to contain %q, got %q", param, signedURL)
		}
	}
	if got := sasProtocol(t, signedURL); got != "https,http" {
		t.Fatalf("expected both protocols on an http emulator endpoint, got spr=%q", got)
	}
}

func TestSignBlobSASDefaultEndpointIsHTTPS(t *testing.T) {
	setAzuriteEnv(t)
	t.Setenv("AZURE_BLOB_ENDPOINT", "")
	signedURL, err := SignBlobSAS("updates", "app/branch/1/1/assets/a.png", sas.BlobPermissions{Read: true}, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPrefix := "https://devstoreaccount1.blob.core.windows.net/updates/app/branch/1/1/assets/a.png?"
	if !strings.HasPrefix(signedURL, wantPrefix) {
		t.Fatalf("expected URL to start with %q, got %q", wantPrefix, signedURL)
	}
	if !strings.Contains(signedURL, "sp=r") {
		t.Fatalf("expected read permission in URL, got %q", signedURL)
	}
	// HTTPS-only SAS on the public endpoint, both protocols only for local
	// emulators. Exact match: a missing spr would default to both protocols
	// server side.
	if got := sasProtocol(t, signedURL); got != "https" {
		t.Fatalf("expected HTTPS-only protocol on the public endpoint, got spr=%q", got)
	}
}

func TestSignBlobSASEscapesPathSegments(t *testing.T) {
	setAzuriteEnv(t)
	signedURL, err := SignBlobSAS("updates", "app/branch/1/1/assets/my file.png", sas.BlobPermissions{Read: true}, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(signedURL, "/assets/my%20file.png?") {
		t.Fatalf("expected escaped blob path in URL, got %q", signedURL)
	}
}

func TestSignBlobSASRequiresCredentials(t *testing.T) {
	t.Setenv("AZURE_STORAGE_ACCOUNT_NAME", "")
	t.Setenv("AZURE_STORAGE_ACCOUNT_KEY", "")
	t.Setenv("AZURE_BLOB_ENDPOINT", "")
	if _, err := SignBlobSAS("updates", "a/b", sas.BlobPermissions{Read: true}, time.Minute); err == nil {
		t.Fatal("expected an error without credentials")
	}
}
