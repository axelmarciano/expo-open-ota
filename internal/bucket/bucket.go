package bucket

import (
	"bytes"
	"expo-open-ota/config"
	"expo-open-ota/internal/types"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// validateSegment ensures a single-segment identifier (branch, runtimeVersion,
// updateId, migrationId) is safe to embed in a storage path / object key. No
// empties, no path separators, no "." / "..". Defense-in-depth against path
// traversal on the local backend and weird keys on S3/GCS.
func validateSegment(name, value string) error {
	if value == "" {
		return fmt.Errorf("invalid %s: must not be empty", name)
	}
	if strings.ContainsAny(value, "/\\") {
		return fmt.Errorf("invalid %s: must not contain path separators", name)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("invalid %s: reserved name", name)
	}
	return nil
}

// validateRelativePath validates multi-segment paths supplied for fileName /
// assetPath. Nested paths are allowed (e.g. "assets/image.png") but no
// absolute paths and no ".." segments.
func validateRelativePath(name, value string) error {
	if value == "" {
		return fmt.Errorf("invalid %s: must not be empty", name)
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") {
		return fmt.Errorf("invalid %s: must not be absolute", name)
	}
	for _, seg := range strings.Split(value, "/") {
		if seg == ".." {
			return fmt.Errorf("invalid %s: must not contain '..' segments", name)
		}
	}
	return nil
}

func validateUpdate(u *types.Update) error {
	if u == nil {
		return fmt.Errorf("update must not be nil")
	}
	if err := validateSegment("appId", u.AppId); err != nil {
		return err
	}
	if err := validateSegment("branch", u.Branch); err != nil {
		return err
	}
	if err := validateSegment("runtimeVersion", u.RuntimeVersion); err != nil {
		return err
	}
	if err := validateSegment("updateId", u.UpdateId); err != nil {
		return err
	}
	return nil
}

// resolveKeyPrefix returns the bucket key prefix, normalized to end with "/"
// when non-empty. It reads BUCKET_KEY_PREFIX first and falls back to the
// legacy S3_KEY_PREFIX env var. Panics on unsafe values (absolute paths or
// ".." segments) to fail-fast on operator misconfiguration that could let
// the local backend escape its BasePath.
func resolveKeyPrefix() string {
	prefix := config.GetEnv("BUCKET_KEY_PREFIX")
	if prefix == "" {
		// TODO: remove S3_KEY_PREFIX backward-compat once users migrated to BUCKET_KEY_PREFIX
		prefix = config.GetEnv("S3_KEY_PREFIX")
	}
	if prefix == "" {
		return ""
	}
	if strings.HasPrefix(prefix, "/") {
		panic("bucket key prefix must not be absolute (starts with '/')")
	}
	for _, seg := range strings.Split(prefix, "/") {
		if seg == ".." {
			panic("bucket key prefix must not contain '..' segments")
		}
	}
	if prefix[len(prefix)-1] != '/' {
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
	GetBranches(appId string) ([]string, error)
	GetRuntimeVersions(appId string, branch string) ([]RuntimeVersionWithStats, error)
	GetUpdates(appId string, branch string, runtimeVersion string) ([]types.Update, error)
	GetFile(update types.Update, assetPath string) (*types.BucketFile, error)
	RequestUploadUrlForFileUpdate(appId string, branch string, runtimeVersion string, updateId string, fileName string) (string, error)
	UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error
	DeleteUpdateFolder(appId string, branch string, runtimeVersion string, updateId string) error
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
			var inner Bucket
			switch bucketType {
			case S3BucketType:
				inner = &S3Bucket{
					BucketName: config.GetEnv("S3_BUCKET_NAME"),
					KeyPrefix:  keyPrefix,
				}
			case GCSBucketType:
				inner = &GCSBucket{
					BucketName: config.GetEnv("GCS_BUCKET_NAME"),
					KeyPrefix:  keyPrefix,
				}
			case LocalBucketType:
				inner = &LocalBucket{
					BasePath:  config.GetEnv("LOCAL_BUCKET_BASE_PATH"),
					KeyPrefix: keyPrefix,
				}
			default:
				panic(fmt.Sprintf("Unknown bucket type: %s", bucketType))
			}
			bucketInstance = &validatingBucket{Inner: inner}
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

func RequestUploadUrlsForFileUpdates(appId string, branch string, runtimeVersion string, updateId string, fileNames []string) ([]FileUploadRequest, error) {
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
			requestUploadUrl, err := bucket.RequestUploadUrlForFileUpdate(appId, branch, runtimeVersion, updateId, fileName)
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
