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

// resolveKeyPrefix returns the bucket key prefix, normalized to end with "/"
// when non-empty. It reads BUCKET_KEY_PREFIX first and falls back to the
// legacy S3_KEY_PREFIX env var.
func resolveKeyPrefix() string {
	prefix := config.GetEnv("BUCKET_KEY_PREFIX")
	if prefix == "" {
		// TODO: remove S3_KEY_PREFIX backward-compat once users migrated to BUCKET_KEY_PREFIX
		prefix = config.GetEnv("S3_KEY_PREFIX")
	}
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	return prefix
}

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
			keyPrefix := resolveKeyPrefix()
			switch bucketType {
			case S3BucketType:
				bucketInstance = &S3Bucket{
					BucketName: config.GetEnv("S3_BUCKET_NAME"),
					KeyPrefix:  keyPrefix,
				}
			case GCSBucketType:
				bucketInstance = &GCSBucket{
					BucketName: config.GetEnv("GCS_BUCKET_NAME"),
					KeyPrefix:  keyPrefix,
				}
			case LocalBucketType:
				bucketInstance = &LocalBucket{
					BasePath:  config.GetEnv("LOCAL_BUCKET_BASE_PATH"),
					KeyPrefix: keyPrefix,
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
