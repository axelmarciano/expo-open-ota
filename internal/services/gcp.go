package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"expo-open-ota/config"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

var (
	gcsClient     *storage.Client
	gcsClientErr  error
	initGCSClient sync.Once
)

func GetGCSClient() (*storage.Client, error) {
	initGCSClient.Do(func() {
		ctx := context.Background()
		gcsClient, gcsClientErr = storage.NewClient(ctx)
		if gcsClientErr != nil {
			gcsClientErr = fmt.Errorf("error initializing GCS client: %w", gcsClientErr)
		}
	})
	return gcsClient, gcsClientErr
}

type googleCreds struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func loadGoogleCreds() (*googleCreds, error) {
	b64Creds := config.GetEnv("GOOGLE_APPLICATION_CREDENTIALS_B64")
	if b64Creds == "" {
		return nil, errors.New("GOOGLE_APPLICATION_CREDENTIALS_B64 not set")
	}
	creds, err := base64.StdEncoding.DecodeString(b64Creds)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GOOGLE_APPLICATION_CREDENTIALS_B64: %w", err)
	}
	var c googleCreds
	if err := json.Unmarshal(creds, &c); err != nil {
		return nil, fmt.Errorf("parse GOOGLE_APPLICATION_CREDENTIALS: %w", err)
	}
	if c.ClientEmail == "" || c.PrivateKey == "" {
		return nil, errors.New("credentials missing client_email or private_key")
	}
	return &c, nil
}

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
