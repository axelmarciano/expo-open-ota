package bucket

import (
	"bufio"
	"encoding/json"
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LocalBucket struct {
	BasePath string
}

func (b *LocalBucket) DeleteUpdateFolder(branch string, runtimeVersion string, updateId string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, branch, runtimeVersion, updateId)
	return os.RemoveAll(dirPath)
}

func (b *LocalBucket) RequestUploadUrlForFileUpdate(branch string, runtimeVersion string, updateId string, fileName string) (string, error) {
	if b.BasePath == "" {
		return "", errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, branch, runtimeVersion, updateId)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	token, err := services.GenerateJWTToken(config.GetEnv("JWT_SECRET"), jwt.MapClaims{
		"sub":      "eoas",
		"exp":      time.Now().Add(time.Minute * 10).Unix(),
		"filePath": filepath.Join(dirPath, fileName),
		"action":   "uploadLocalFile",
	})
	if err != nil {
		return "", err
	}
	parsedURL, err := url.Parse(config.GetEnv("BASE_URL"))
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	parsedURL.Path, err = url.JoinPath(parsedURL.Path, "uploadLocalFile")
	if err != nil {
		return "", fmt.Errorf("error joining path: %w", err)
	}
	query := url.Values{}
	query.Set("token", token)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func (b *LocalBucket) UpsertBranch(branch string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}
	keepFilePath := filepath.Join(b.BasePath, branch, ".keep")
	err := os.MkdirAll(filepath.Dir(keepFilePath), os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(keepFilePath)
	if err != nil {
		return err
	}
	return file.Close()
}

func (b *LocalBucket) SetChannelMapping(channel, branch string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}
	channelsPath := filepath.Join(b.BasePath, ".channels")
	if err := os.MkdirAll(channelsPath, os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(channelsPath, channel+".json")
	data, err := json.Marshal(map[string]string{"branch": branch})
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (b *LocalBucket) GetChannelMapping(channel string) (string, error) {
	if b.BasePath == "" {
		return "", errors.New("BasePath not set")
	}
	filePath := filepath.Join(b.BasePath, ".channels", channel+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("channel %s not found", channel)
		}
		return "", err
	}
	var mapping struct {
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return "", fmt.Errorf("error decoding channel mapping: %w", err)
	}
	return mapping.Branch, nil
}

func (b *LocalBucket) DeleteChannel(channel string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}
	filePath := filepath.Join(b.BasePath, ".channels", channel+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (b *LocalBucket) GetChannels() ([]string, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	channelsPath := filepath.Join(b.BasePath, ".channels")
	entries, err := os.ReadDir(channelsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var channels []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			channelName := strings.TrimSuffix(entry.Name(), ".json")
			channels = append(channels, channelName)
		}
	}
	return channels, nil
}

func (b *LocalBucket) GetUpdates(branch string, runtimeVersion string) ([]types.Update, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, branch, runtimeVersion)
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
					Branch:         branch,
					RuntimeVersion: runtimeVersion,
					UpdateId:       strconv.FormatInt(updateId, 10),
					CreatedAt:      time.Duration(updateId) * time.Millisecond,
				})
			}
		}
	}
	return updates, nil
}

func (b *LocalBucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}

	filePath := filepath.Join(b.BasePath, update.Branch, update.RuntimeVersion, update.UpdateId, assetPath)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &types.BucketFile{
		Reader:    file,
		CreatedAt: info.ModTime(),
	}, nil
}
func (b *LocalBucket) GetBranches() ([]string, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	entries, err := os.ReadDir(b.BasePath)
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			branches = append(branches, entry.Name())
		}
	}
	return branches, nil
}

func (b *LocalBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, branch)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var runtimeVersions []RuntimeVersionWithStats
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runtimeVersion := entry.Name()
		updatesPath := filepath.Join(dirPath, runtimeVersion)
		updates, err := os.ReadDir(updatesPath)
		if err != nil {
			continue
		}
		var updateTimestamps []int64
		for _, update := range updates {
			if !update.IsDir() {
				continue
			}
			timestamp, err := strconv.ParseInt(update.Name(), 10, 64)
			if err != nil {
				continue
			}
			updateTimestamps = append(updateTimestamps, timestamp)
		}
		if len(updateTimestamps) == 0 {
			continue
		}

		sort.Slice(updateTimestamps, func(i, j int) bool { return updateTimestamps[i] < updateTimestamps[j] })

		runtimeVersions = append(runtimeVersions, RuntimeVersionWithStats{
			RuntimeVersion:  runtimeVersion,
			CreatedAt:       time.UnixMilli(updateTimestamps[0]).UTC().Format(time.RFC3339),
			LastUpdatedAt:   time.UnixMilli(updateTimestamps[len(updateTimestamps)-1]).UTC().Format(time.RFC3339),
			NumberOfUpdates: len(updateTimestamps),
		})
	}

	return runtimeVersions, nil
}

func (b *LocalBucket) UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error {
	filePath := filepath.Join(b.BasePath, update.Branch, update.RuntimeVersion, update.UpdateId, fileName)
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return err
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		return err
	}
	return nil
}

func ValidateUploadTokenAndResolveFilePath(token string) (string, error) {
	claims := jwt.MapClaims{}
	decodedToken, err := services.DecodeAndExtractJWTToken(config.GetEnv("JWT_SECRET"), token, claims)
	if err != nil {
		return "", err
	}
	if !decodedToken.Valid {
		return "", errors.New("invalid token")
	}
	action := claims["action"].(string)
	filePath := claims["filePath"].(string)
	sub := claims["sub"].(string)
	if sub != "eoas" {
		return "", errors.New("invalid token sub")
	}
	if action != "uploadLocalFile" {
		return "", errors.New("invalid token action")
	}
	return filePath, nil
}

func HandleUploadFile(filePath string, body multipart.File) (bool, error) {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return false, err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	_, err = io.Copy(file, body)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b *LocalBucket) CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error) {
	if previousUpdate == nil {
		return nil, errors.New("previousUpdate is nil")
	}
	if previousUpdate.UpdateId == "" {
		return nil, errors.New("previousUpdate.UpdateId is empty")
	}
	if newUpdateId == "" {
		return nil, errors.New("newUpdateId is empty")
	}

	previousUpdatePath := filepath.Join(b.BasePath, previousUpdate.Branch, previousUpdate.RuntimeVersion, previousUpdate.UpdateId)
	newUpdatePath := filepath.Join(b.BasePath, previousUpdate.Branch, previousUpdate.RuntimeVersion, newUpdateId)

	err := os.MkdirAll(newUpdatePath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(previousUpdatePath)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(entries))
	sem := make(chan struct{}, runtime.NumCPU())

	for _, entry := range entries {
		name := entry.Name()
		if name == "update-metadata.json" || name == ".check" {
			continue
		}

		srcPath := filepath.Join(previousUpdatePath, name)
		dstPath := filepath.Join(newUpdatePath, name)

		wg.Add(1)
		go func(entry fs.DirEntry, src, dst string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var err error
			if entry.IsDir() {
				err = copyDirParallel(src, dst)
			} else {
				err = copyFile(src, dst)
			}
			if err != nil {
				errChan <- err
			}
		}(entry, srcPath, dstPath)
	}

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return nil, e
		}
	}

	updateId, err := strconv.ParseInt(newUpdateId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing update ID: %w", err)
	}
	return &types.Update{
		Branch:         previousUpdate.Branch,
		RuntimeVersion: previousUpdate.RuntimeVersion,
		UpdateId:       newUpdateId,
		CreatedAt:      time.Duration(updateId) * time.Millisecond,
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

func (b *LocalBucket) RetrieveMigrationHistory() ([]string, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	migrationHistoryPath := filepath.Join(b.BasePath, ".migrationhistory")
	file, err := os.Open(migrationHistoryPath)
	if err != nil {
		return nil, nil
	}
	defer file.Close()
	var migrations []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		migrations = append(migrations, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}

func (b *LocalBucket) ApplyMigration(migrationId string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}

	migrationHistoryPath := filepath.Join(b.BasePath, ".migrationhistory")

	migrations, err := b.RetrieveMigrationHistory()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
	}
	for _, id := range migrations {
		if id == migrationId {
			return nil
		}
	}

	file, err := os.OpenFile(migrationHistoryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open .migrationhistory error: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(migrationId + "\n"); err != nil {
		return fmt.Errorf("write .migrationhistory error: %w", err)
	}

	return nil
}

func (b *LocalBucket) RemoveMigrationFromHistory(migrationId string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}

	migrationHistoryPath := filepath.Join(b.BasePath, ".migrationhistory")

	migrations, err := b.RetrieveMigrationHistory()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
	}
	hasMigration := false
	for _, id := range migrations {
		if id == migrationId {
			hasMigration = true
			break
		}
	}
	if !hasMigration {
		return nil
	}

	var newMigrations []string
	for _, id := range migrations {
		if id != migrationId {
			newMigrations = append(newMigrations, id)
		}
	}

	file, err := os.OpenFile(migrationHistoryPath, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open .migrationhistory error: %w", err)
	}
	defer file.Close()

	for _, id := range newMigrations {
		if _, err := file.WriteString(id + "\n"); err != nil {
			return fmt.Errorf("write .migrationhistory error: %w", err)
		}
	}

	return nil
}

func copyDirParallel(srcDir, dstDir string) error {
	err := os.MkdirAll(dstDir, os.ModePerm)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(entries))
	sem := make(chan struct{}, runtime.NumCPU())
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		wg.Add(1)
		go func(entry fs.DirEntry, src, dst string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var err error
			if entry.IsDir() {
				err = copyDirParallel(src, dst)
			} else {
				err = copyFile(src, dst)
			}
			if err != nil {
				errChan <- err
			}
		}(entry, srcPath, dstPath)
	}
	wg.Wait()
	close(errChan)
	for e := range errChan {
		if e != nil {
			return e
		}
	}
	return nil
}
