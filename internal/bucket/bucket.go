package bucket

import (
	"bytes"
	"expo-open-ota/config"
	"expo-open-ota/internal/types"
	"fmt"
	"io"
	"path/filepath"
	"sync"
)

type RuntimeVersionWithStats struct {
	RuntimeVersion  string `json:"runtimeVersion"`
	LastUpdatedAt   string `json:"lastUpdatedAt"`
	CreatedAt       string `json:"createdAt"`
	NumberOfUpdates int    `json:"numberOfUpdates"`
}

type Bucket interface {
	GetBranches() ([]string, error)
	GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error)
	GetUpdates(branch string, runtimeVersion string) ([]types.Update, error)
	GetFile(update types.Update, assetPath string) (*types.BucketFile, error)
	RequestUploadUrlForFileUpdate(branch string, runtimeVersion string, updateId string, fileName string) (string, error)
	UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error
	DeleteUpdateFolder(branch string, runtimeVersion string, updateId string) error
	CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error)
	RetrieveMigrationHistory() ([]string, error)
	ApplyMigration(migrationId string) error
	RemoveMigrationFromHistory(migrationId string) error
}

type BucketType string

const (
	S3BucketType    BucketType = "s3"
	LocalBucketType BucketType = "local"
	GCSBucketType   BucketType = "gcs"
)

func ResolveBucketType() BucketType {
	storageMode := config.GetEnv("STORAGE_MODE")
	switch storageMode {
	case "local", "":
		return LocalBucketType
	case "s3":
		return S3BucketType
	case "gcs":
		return GCSBucketType
	default:
		return LocalBucketType
	}
}

var (
	bucketInstance Bucket
	once           sync.Once
)

func GetBucket() Bucket {
	once.Do(func() {
		if bucketInstance == nil {
			bucketType := ResolveBucketType()
			switch bucketType {
			case S3BucketType:
				bucketName := config.GetEnv("S3_BUCKET_NAME")
				bucketInstance = &S3Bucket{
					BucketName: bucketName,
				}
			case GCSBucketType:
				bucketName := config.GetEnv("GCS_BUCKET_NAME")
				bucketInstance = &GCSBucket{
					BucketName: bucketName,
				}
			case LocalBucketType:
				basePath := config.GetEnv("LOCAL_BUCKET_BASE_PATH")
				bucketInstance = &LocalBucket{
					BasePath: basePath,
				}
			default:
				panic(fmt.Sprintf("Unknown bucket type: %s", bucketType))
			}
		}
	})
	return bucketInstance
}

func ConvertReadCloserToBytes(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rc); err != nil {
		return nil, fmt.Errorf("error copying file to buffer: %w", err)
	}
	return buf.Bytes(), nil
}

func ResetBucketInstance() {
	bucketInstance = nil
	once = sync.Once{}
}

type FileUploadRequest struct {
	RequestUploadUrl string `json:"requestUploadUrl"`
	FileName         string `json:"fileName"`
	FilePath         string `json:"filePath"`
}

func RequestUploadUrlsForFileUpdates(branch string, runtimeVersion string, updateId string, fileNames []string) ([]FileUploadRequest, error) {
	uniqueFileNames := make(map[string]struct{})
	for _, fileName := range fileNames {
		uniqueFileNames[fileName] = struct{}{}
	}

	bucket := GetBucket()

	var requests []FileUploadRequest
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(uniqueFileNames))

	wg.Add(len(uniqueFileNames))
	for fileName := range uniqueFileNames {
		go func(fileName string) {
			defer wg.Done()
			requestUploadUrl, err := bucket.RequestUploadUrlForFileUpdate(branch, runtimeVersion, updateId, fileName)
			if err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			requests = append(requests, FileUploadRequest{
				RequestUploadUrl: requestUploadUrl,
				FileName:         filepath.Base(fileName),
				FilePath:         fileName,
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
