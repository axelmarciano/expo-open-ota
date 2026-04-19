package bucket

import (
	"bufio"
	"errors"
	"strings"
	"expo-open-ota/config"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"io/fs"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

type LocalBucket struct {
	BasePath  string
	KeyPrefix string
}

func (b *LocalBucket) rootPath() string {
	if b.KeyPrefix == "" {
		return b.BasePath
	}
	return filepath.Join(b.BasePath, b.KeyPrefix)
}

func (b *LocalBucket) DeleteUpdateFolder(branch string, runtimeVersion string, updateId string) error {
	if b.BasePath == "" {
		return errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.rootPath(), branch, runtimeVersion, updateId)
	return os.RemoveAll(dirPath)
}

func (b *LocalBucket) RequestUploadUrlForFileUpdate(branch string, runtimeVersion string, updateId string, fileName string) (string, error) {
	if b.BasePath == "" {
		return "", errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.rootPath(), branch, runtimeVersion, updateId)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	token, err := services.GenerateJWTToken(config.GetEnv("JWT_SECRET"), jwt.MapClaims{
		"sub":      services.FetchSelfExpoUsername(),
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

func (b *LocalBucket) GetUpdates(branch string, runtimeVersion string) ([]types.Update, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.rootPath(), branch, runtimeVersion)
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

	expectedBase := filepath.Join(b.rootPath(), update.Branch, update.RuntimeVersion, update.UpdateId)
	filePath := filepath.Join(expectedBase, assetPath)
	// Use filepath.Rel so sibling dirs sharing a string prefix (e.g. ".../123" vs ".../1234")
	// aren't treated as nested, and so "." (the base itself) is accepted.
	rel, err := filepath.Rel(expectedBase, filePath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return nil, errors.New("invalid asset path")
	}

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
	entries, err := os.ReadDir(b.rootPath())
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, entry := range entries {
		if entry.IsDir() {
			branches = append(branches, entry.Name())
		}
	}
	return branches, nil
}

func (b *LocalBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.rootPath(), branch)
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
	filePath := filepath.Join(b.rootPath(), update.Branch, update.RuntimeVersion, update.UpdateId, fileName)
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
	if sub != services.FetchSelfExpoUsername() {
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

	previousUpdatePath := filepath.Join(b.rootPath(), previousUpdate.Branch, previousUpdate.RuntimeVersion, previousUpdate.UpdateId)
	newUpdatePath := filepath.Join(b.rootPath(), previousUpdate.Branch, previousUpdate.RuntimeVersion, newUpdateId)

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
	migrationHistoryPath := filepath.Join(b.rootPath(), ".migrationhistory")
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

	migrationHistoryPath := filepath.Join(b.rootPath(), ".migrationhistory")

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

	migrationHistoryPath := filepath.Join(b.rootPath(), ".migrationhistory")

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
