package bucket

import (
	"errors"
	"expo-open-ota/internal/modules/types"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type LocalBucket struct {
	BasePath string
}

func (b *LocalBucket) GetUpdates(environment string, runtimeVersion string) ([]types.Update, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, environment, runtimeVersion)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return []types.Update{}, nil
	}
	var updates []types.Update
	for _, entry := range entries {
		if entry.IsDir() {
			updateId, err := strconv.ParseInt(entry.Name(), 10, 64)
			if err == nil {
				updates = append(updates, types.Update{
					Environment:    environment,
					RuntimeVersion: runtimeVersion,
					UpdateId:       strconv.FormatInt(updateId, 10),
					CreatedAt:      time.Duration(updateId) * time.Millisecond,
				})
			}
		}
	}
	return updates, nil
}

func (b *LocalBucket) GetFile(update types.Update, assetPath string) (types.BucketFile, error) {
	if b.BasePath == "" {
		return types.BucketFile{}, errors.New("BasePath not set")
	}

	filePath := filepath.Join(b.BasePath, update.Environment, update.RuntimeVersion, update.UpdateId, assetPath)

	file, err := os.Open(filePath)
	if err != nil {
		return types.BucketFile{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return types.BucketFile{}, err
	}
	return types.BucketFile{
		Reader:    file,
		CreatedAt: fileInfo.ModTime(),
	}, nil
}
