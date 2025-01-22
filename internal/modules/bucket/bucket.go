package bucket

import (
	"bytes"
	"expo-open-ota/internal/modules/types"
	"fmt"
	"io"
)

type Bucket interface {
	GetUpdates(environment string, runtimeVersion string) ([]types.Update, error)
	GetFile(update types.Update, assetPath string) (types.BucketFile, error)
}

type BucketType string

const (
	S3BucketType BucketType = "s3"
)

func GetBucket(bucketType BucketType) (Bucket, error) {
	switch bucketType {
	case S3BucketType:
		return &S3Bucket{}, nil
	default:
		return nil, fmt.Errorf("unknown bucket type: %s", bucketType)
	}
}

func ConvertReadCloserToBytes(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rc); err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du ReadCloser : %w", err)
	}
	return buf.Bytes(), nil
}
