package services

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/base64"
	"expo-open-ota/config"
	"fmt"
	"google.golang.org/api/option"
	"sync"
)

var (
	storageClient     *storage.Client
	initStorageClient sync.Once
)

func GetFirebaseStorageClient() (*storage.Client, error) {
	var err error
	initStorageClient.Do(func() {
		ctx := context.Background()
		if b64 := config.GetEnv("FIREBASE_SERVICE_ACCOUNT_B64"); b64 != "" {
			raw, decErr := base64.StdEncoding.DecodeString(b64)
			if decErr != nil {
				err = fmt.Errorf("decoding FIREBASE_SERVICE_ACCOUNT_B64: %w", decErr)
				return
			}
			storageClient, err = storage.NewClient(ctx, option.WithCredentialsJSON(raw))
			return
		}
		storageClient, err = storage.NewClient(ctx)
	})

	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase Storage client: %w", err)
	}
	return storageClient, nil
}
