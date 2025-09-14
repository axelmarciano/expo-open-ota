package services

import (
    "cloud.google.com/go/storage"
    "context"
    "encoding/json"
    "errors"
    "expo-open-ota/config"
    "fmt"
    "os"
    "sync"
    "time"
)

var (
    gcsClient     *storage.Client
    initGCSClient sync.Once
)

func GetGCSClient() (*storage.Client, error) {
    var err error
    initGCSClient.Do(func() {
        ctx := context.Background()
        gcsClient, err = storage.NewClient(ctx)
    })
    if err != nil {
        return nil, fmt.Errorf("error initializing GCS client: %w", err)
    }
    return gcsClient, nil
}

type googleCreds struct {
    ClientEmail string `json:"client_email"`
    PrivateKey  string `json:"private_key"`
}

func loadGoogleCreds() (*googleCreds, error) {
    path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
    if path == "" {
        return nil, errors.New("GOOGLE_APPLICATION_CREDENTIALS not set")
    }
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read GOOGLE_APPLICATION_CREDENTIALS: %w", err)
    }
    var c googleCreds
    if err := json.Unmarshal(b, &c); err != nil {
        return nil, fmt.Errorf("parse GOOGLE_APPLICATION_CREDENTIALS: %w", err)
    }
    if c.ClientEmail == "" || c.PrivateKey == "" {
        return nil, errors.New("credentials missing client_email or private_key")
    }
    return &c, nil
}

// GCSSignedURL generates a V4 signed URL for a GCS object using service account credentials.
// method is typically "PUT" or "GET".
func GCSSignedURL(bucket, key, method, contentType string, expires time.Duration) (string, error) {
    creds, err := loadGoogleCreds()
    if err != nil {
        return "", err
    }
    opts := &storage.SignedURLOptions{
        Scheme:         storage.SigningSchemeV4,
        Method:         method,
        Expires:        time.Now().Add(expires),
        GoogleAccessID: creds.ClientEmail,
        PrivateKey:     []byte(creds.PrivateKey),
    }
    if contentType != "" {
        opts.ContentType = contentType
    }
    return storage.SignedURL(bucket, key, opts)
}

func IsGCSConfigured() bool {
    // Basic runtime check: bucket exists in env and client can be created.
    if config.GetEnv("GCS_BUCKET_NAME") == "" {
        return false
    }
    _, err := GetGCSClient()
    return err == nil
}
