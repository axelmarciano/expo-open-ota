package bucket

import (
	"bytes"
	"expo-open-ota/config"
	"expo-open-ota/internal/modules/types"
	"fmt"
	"io"
	"sync"
)

type Bucket interface {
	GetUpdates(environment string, runtimeVersion string) ([]types.Update, error)
	GetFile(update types.Update, assetPath string) (types.BucketFile, error)
	RequestUploadUrlForFileUpdate(environment string, runtimeVersion string, updateId string, fileName string) (string, error)
}

type BucketType string

const (
	S3BucketType    BucketType = "s3"
	LocalBucketType BucketType = "local"
)

func ResolveBucketType() BucketType {
	bucketType := config.GetEnv("STORAGE_MODE")
	if bucketType == "" || bucketType == "local" {
		return LocalBucketType
	}
	return S3BucketType
}

func GetBucket() (Bucket, error) {
	bucketType := ResolveBucketType()
	switch bucketType {
	case S3BucketType:
		bucketName := config.GetEnv("S3_BUCKET_NAME")
		if bucketName == "" {
			return nil, fmt.Errorf("S3_BUCKET_NAME not set in environment")
		}
		return &S3Bucket{
			BucketName: bucketName,
		}, nil
	case LocalBucketType:
		basePath := config.GetEnv("LOCAL_BUCKET_BASE_PATH")
		if basePath == "" {
			return nil, fmt.Errorf("LOCAL_BUCKET_BASE_PATH not set in environment")
		}
		return &LocalBucket{
			BasePath: basePath,
		}, nil
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

type FileUploadRequest struct {
	RequestUploadUrl string `json:"requestUploadUrl"`
	FileName         string `json:"fileName"`
}

func RequestUploadUrlsForFileUpdates(environment string, runtimeVersion string, updateId string, fileNames []string) ([]FileUploadRequest, error) {
	bucket, err := GetBucket()
	if err != nil {
		return nil, err
	}

	var requests []FileUploadRequest
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(fileNames))

	wg.Add(len(fileNames))
	for _, fileName := range fileNames {
		go func(fileName string) {
			defer wg.Done()
			requestUploadUrl, err := bucket.RequestUploadUrlForFileUpdate(environment, runtimeVersion, updateId, fileName)
			if err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			requests = append(requests, FileUploadRequest{
				RequestUploadUrl: requestUploadUrl,
				FileName:         fileName,
			})
			mu.Unlock()
		}(fileName)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return requests, nil
}
